package epub

import (
	"encoding/xml"
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"strconv"
)

const (
	tocNavBodyTemplate = `
    <nav epub:type="toc">
      <h1>Table of Contents</h1>
      <ol>
      </ol>
    </nav>
`
	tocNavFilename       = "nav.xhtml"
	tocNavItemID         = "nav"
	tocNavItemProperties = "nav"
	tocNavEpubType       = "toc"

	tocNcxFilename = "toc.ncx"
	tocNcxItemID   = "ncx"

	tocNcxMetaUID   = "dtb:uid"
	tocNcxMetaDepth = "dtb:depth"
	tocNcxTemplate  = `
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <head>
    <meta name="dtb:uid" content="" />
    <meta name="dtb:depth" content="" />
  </head>
  <docTitle>
    <text></text>
  </docTitle>
  <docAuthor>
    <text></text>
  </docAuthor>
  <navMap>
  </navMap>
</ncx>`

	xmlnsEpub = "http://www.idpf.org/2007/ops"
)

// toc implements the EPUB table of contents
type toc struct {
	// This holds the body XML for the EPUB v3 TOC file (nav.xhtml). Since this is
	// an XHTML file, the rest of the structure is handled by the xhtml type
	//
	// Spec: http://www.idpf.org/epub/301/spec/epub-contentdocs.html#sec-xhtml-nav
	navXML *tocNavBody

	// This holds the XML for the EPUB v2 TOC file (toc.ncx). This is added so the
	// resulting EPUB v3 file will still work with devices that only support EPUB v2
	//
	// Spec: http://www.idpf.org/epub/20/spec/OPF_2.0.1_draft.htm#Section2.4.1
	ncxXML *tocNcxRoot

	title  string // EPUB title
	author string // EPUB author
	lang   string // language of nav.xhtml's root, like every other document
}

type tocNavBody struct {
	XMLName  xml.Name      `xml:"nav"`
	EpubType string        `xml:"epub:type,attr"`
	H1       string        `xml:"h1"`
	Links    []*tocNavItem `xml:"ol>li"`
}

type tocNavItem struct {
	A        tocNavLink    `xml:"a"`
	Children []*tocNavItem `xml:"ol>li,omitempty"`
}

type tocNavLink struct {
	XMLName xml.Name `xml:"a"`
	Href    string   `xml:"href,attr"`
	Data    string   `xml:",chardata"`
}

type tocNcxRoot struct {
	XMLName xml.Name          `xml:"http://www.daisy.org/z3986/2005/ncx/ ncx"`
	Version string            `xml:"version,attr"`
	Meta    []tocNcxMeta      `xml:"head>meta"`
	Title   string            `xml:"docTitle>text"`
	Author  string            `xml:"docAuthor>text"`
	NavMap  []*tocNcxNavPoint `xml:"navMap>navPoint"`
}

type tocNcxContent struct {
	Src string `xml:"src,attr"`
}

type tocNcxMeta struct {
	Name    string `xml:"name,attr"`
	Content string `xml:"content,attr"`
}

type tocNcxNavPoint struct {
	XMLName  xml.Name          `xml:"navPoint"`
	ID       string            `xml:"id,attr"`
	Text     string            `xml:"navLabel>text"`
	Content  tocNcxContent     `xml:"content"`
	Children []*tocNcxNavPoint `xml:"navPoint,omitempty"`
}

// Constructor for toc
func newToc() (*toc, error) {
	t := &toc{}
	var err error

	t.navXML, err = newTocNavXML()
	if err != nil {
		return nil, fmt.Errorf("can't create navXML: %w", err)
	}

	t.ncxXML, err = newTocNcxXML()
	if err != nil {
		return nil, fmt.Errorf("can't create navXML: %w", err)
	}

	return t, nil
}

// Constructor for tocNavBody
func newTocNavXML() (*tocNavBody, error) {
	b := &tocNavBody{
		EpubType: tocNavEpubType,
	}
	err := xml.Unmarshal([]byte(tocNavBodyTemplate), &b)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshalling tocNavBody: %w\n"+"\ttocNavBody=%#v\n"+"\ttocNavBodyTemplate=%s", err, *b, tocNavBodyTemplate)
	}

	return b, nil
}

// Constructor for tocNcxRoot
func newTocNcxXML() (*tocNcxRoot, error) {
	n := &tocNcxRoot{}

	err := xml.Unmarshal([]byte(tocNcxTemplate), &n)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshalling tocNcxRoot: %w\n"+"\ttocNcxRoot=%#v\n"+"\ttocNcxTemplate=%s", err, *n, tocNcxTemplate)
	}

	return n, nil
}

// TODO: user should not add -1 as filename
// Add a section to the TOC (navXML as well as ncxXML)
func (t *toc) addSubSection(parent string, index int, title string, relativePath string) {
	relativePath = filepath.ToSlash(relativePath)
	if parent == "-1" {

		l := &tocNavItem{
			A: tocNavLink{
				Href: relativePath,
				Data: title,
			},
			Children: nil,
		}

		np := &tocNcxNavPoint{
			ID:   "navPoint-" + strconv.Itoa(index),
			Text: title,
			Content: tocNcxContent{
				Src: relativePath,
			},
			Children: nil,
		}
		t.navXML.Links = append(t.navXML.Links, l)
		t.ncxXML.NavMap = append(t.ncxXML.NavMap, np)
	} else {

		parentRelativePath := filepath.Join(xhtmlFolderName, parent)

		l := &tocNavItem{
			A: tocNavLink{
				Href: relativePath,
				Data: title,
			},
		}
		np := &tocNcxNavPoint{
			ID:   "navPoint-" + strconv.Itoa(index),
			Text: title,
			Content: tocNcxContent{
				Src: relativePath,
			},
			Children: nil,
		}
		err := navAppender(t.navXML.Links, parentRelativePath, l)
		if err != nil {
			log.Println(err)
		}
		err = ncxAppender(t.ncxXML.NavMap, parentRelativePath, np)
		if err != nil {
			log.Println(err)
		}
	}
}

func (t *toc) setIdentifier(identifier string) {
	t.setMeta(tocNcxMetaUID, identifier)
}

// setMeta updates the NCX <meta> entry with the given name, appending it when
// it does not exist yet. Existing entries keep their position.
func (t *toc) setMeta(name string, content string) {
	for i := range t.ncxXML.Meta {
		if t.ncxXML.Meta[i].Name == name {
			t.ncxXML.Meta[i].Content = content
			return
		}
	}
	t.ncxXML.Meta = append(t.ncxXML.Meta, tocNcxMeta{Name: name, Content: content})
}

// navPointDepth returns the nesting depth of the navPoint tree, with a minimum
// of 1. An empty tree still has a depth of 1, per the NCX specification.
func navPointDepth(points []*tocNcxNavPoint) int {
	depth := maxNavPointDepth(points)
	if depth < 1 {
		return 1
	}
	return depth
}

// maxNavPointDepth returns the deepest nesting level in the tree, and 0 when
// the tree is empty.
func maxNavPointDepth(points []*tocNcxNavPoint) int {
	depth := 0
	for _, p := range points {
		if d := 1 + maxNavPointDepth(p.Children); d > depth {
			depth = d
		}
	}
	return depth
}

func (t *toc) setTitle(title string) {
	t.title = title
}

// setLang records the language nav.xhtml's root element must carry, the same
// language every content document's root carries.
func (t *toc) setLang(lang string) {
	t.lang = lang
}

// Write the TOC files
func (t *toc) write(tempDir string) error {
	err := t.writeNavDoc(tempDir)
	if err != nil {
		return err
	}
	err = t.writeNcxDoc(tempDir)
	if err != nil {
		return err
	}
	return nil
}

// Write the the EPUB v3 TOC file (nav.xhtml) to the temporary directory
func (t *toc) writeNavDoc(tempDir string) error {
	navBodyContent, err := xml.MarshalIndent(t.navXML, "    ", "  ")
	if err != nil {
		return fmt.Errorf("Error marshalling XML for EPUB v3 TOC file: %w\n"+"\tXML=%#v", err, t.navXML)
	}

	// subsection without children itself left an empty tag <ol></ol>
	// that not acceptable for epub v3
	// this regex will remove those line from tocnav.
	// TODO: find a better solution
	re := regexp.MustCompile(`\s*<ol>\s*</ol>`)
	bodyWithoutEmptyTags := re.ReplaceAllString(string(navBodyContent), "")

	n, err := newXhtml(bodyWithoutEmptyTags)
	if err != nil {
		return fmt.Errorf("can't create xhtml for TOC file: %w", err)
	}
	n.setXmlnsEpub(xmlnsEpub)
	n.setLang(t.lang)
	n.setTitle(t.title)

	navFilePath := filepath.Join(tempDir, contentFolderName, tocNavFilename)
	err = n.write(navFilePath)
	if err != nil {
		return fmt.Errorf("can't write TOC file: %w", err)
	}
	return nil
}

// Write the EPUB v2 TOC file (toc.ncx) to the temporary directory
func (t *toc) writeNcxDoc(tempDir string) error {
	t.ncxXML.Title = t.title
	t.ncxXML.Author = t.author
	t.setMeta(tocNcxMetaDepth, strconv.Itoa(navPointDepth(t.ncxXML.NavMap)))

	ncxFileContent, err := xml.MarshalIndent(t.ncxXML, "", "  ")
	if err != nil {
		return fmt.Errorf("Error marshalling XML for EPUB v2 TOC file: %w\n"+"+\tXML=%#v", err, t.ncxXML)
	}

	// Add the xml header to the output
	ncxFileContent = append([]byte(xml.Header), ncxFileContent...)
	// It's generally nice to have files end with a newline
	ncxFileContent = append(ncxFileContent, "\n"...)

	ncxFilePath := filepath.Join(tempDir, contentFolderName, tocNcxFilename)
	if err := filesystem.WriteFile(ncxFilePath, []byte(ncxFileContent), filePermissions); err != nil {
		return fmt.Errorf("Error writing EPUB v2 TOC file: %w", err)
	}
	return nil
}

// Append tocNcxNavPoint to parent children for toc in Epub v2
func ncxAppender(t []*tocNcxNavPoint, parentFilename string, targetsection *tocNcxNavPoint) error {
	// Search for the epubSection with filename equal to parentFilename
	for _, ncx := range t {
		if ncx.Content.Src == parentFilename {
			// Append targetsection as children of the found section
			ncx.Children = append(ncx.Children, targetsection)
			return nil
		}
		// Recursively check all children of the current section
		err := ncxAppender(ncx.Children, parentFilename, targetsection)
		if err == nil {
			return nil
		}
	}
	return fmt.Errorf("parent section not found")
}

// Append tocNavItem to parent children for toc in Epub v3
func navAppender(t []*tocNavItem, parentFilename string, targetsection *tocNavItem) error {
	// Search for the epubSection with filename equal to parentFilename
	for _, nav := range t {
		if nav.A.Href == parentFilename {
			// Append targetsection as children of the found section
			nav.Children = append(nav.Children, targetsection)
			return nil
		}
		// Recursively check all children of the current section
		err := navAppender(nav.Children, parentFilename, targetsection)
		if err == nil {
			return nil
		}
	}
	return fmt.Errorf("parent section not found")
}
