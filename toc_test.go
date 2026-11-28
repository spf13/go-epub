package epub

import (
	"encoding/xml"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/go-shiori/go-epub/internal/storage"
)

// testNcxHeader mirrors the parts of toc.ncx that carry header metadata.
type testNcxHeader struct {
	Head struct {
		Meta []struct {
			Name    string `xml:"name,attr"`
			Content string `xml:"content,attr"`
		} `xml:"meta"`
	} `xml:"head"`
}

// testPkgIdentifier mirrors <dc:identifier> in package.opf.
type testPkgIdentifier struct {
	Metadata struct {
		Identifier struct {
			ID   string `xml:"id,attr"`
			Data string `xml:",chardata"`
		} `xml:"identifier"`
	} `xml:"metadata"`
}

func TestTocNcxMetaHeader(t *testing.T) {
	e, err := NewEpub(testEpubTitle)
	if err != nil {
		t.Fatal(err)
	}
	e.SetIdentifier(testEpubIdentifier)

	sectionPath, err := e.AddSection(testSectionBody, testSectionTitle, testSectionFilename, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddSubSection(sectionPath, testSectionBody, "Subsection", "", ""); err != nil {
		t.Fatal(err)
	}

	tempDir := writeAndExtractEpub(t, e, testEpubFilename)

	ncxContents, err := storage.ReadFile(filesystem, filepath.Join(tempDir, contentFolderName, tocNcxFilename))
	if err != nil {
		t.Fatal(err)
	}

	var ncx testNcxHeader
	if err := xml.Unmarshal(ncxContents, &ncx); err != nil {
		t.Fatalf("toc.ncx is not well-formed XML: %s\n%s", err, ncxContents)
	}

	meta := make(map[string]string, len(ncx.Head.Meta))
	for _, m := range ncx.Head.Meta {
		meta[m.Name] = m.Content
	}

	uid, ok := meta[tocNcxMetaUID]
	if !ok {
		t.Fatalf("toc.ncx has no %s meta element:\n%s", tocNcxMetaUID, ncxContents)
	}

	// The NCX identifier must match the package unique-identifier.
	pkgContents, err := storage.ReadFile(filesystem, filepath.Join(tempDir, contentFolderName, pkgFilename))
	if err != nil {
		t.Fatal(err)
	}
	var pkg testPkgIdentifier
	if err := xml.Unmarshal(pkgContents, &pkg); err != nil {
		t.Fatalf("package.opf is not well-formed XML: %s\n%s", err, pkgContents)
	}
	if uid != pkg.Metadata.Identifier.Data {
		t.Errorf(
			"NCX identifier doesn't match the package unique-identifier\n"+
				"Got: %s\n"+
				"Expected: %s",
			uid,
			pkg.Metadata.Identifier.Data)
	}

	// One top-level section with one subsection gives a depth of 2.
	depth, ok := meta[tocNcxMetaDepth]
	if !ok {
		t.Fatalf("toc.ncx has no %s meta element:\n%s", tocNcxMetaDepth, ncxContents)
	}
	if _, err := strconv.Atoi(depth); err != nil {
		t.Errorf("%s content is not numeric: %q\n%s", tocNcxMetaDepth, depth, ncxContents)
	}
	if depth != "2" {
		t.Errorf(
			"NCX depth doesn't match the navMap nesting\n"+
				"Got: %s\n"+
				"Expected: 2",
			depth)
	}
	if strings.Contains(depth, testEpubIdentifier) {
		t.Errorf("%s content holds the identifier instead of a depth: %q", tocNcxMetaDepth, depth)
	}

	cleanup(testEpubFilename, tempDir)
}

func TestTocNcxMetaHeaderWithoutSections(t *testing.T) {
	e, err := NewEpub(testEpubTitle)
	if err != nil {
		t.Fatal(err)
	}

	tempDir := writeAndExtractEpub(t, e, testEpubFilename)

	ncxContents, err := storage.ReadFile(filesystem, filepath.Join(tempDir, contentFolderName, tocNcxFilename))
	if err != nil {
		t.Fatal(err)
	}

	var ncx testNcxHeader
	if err := xml.Unmarshal(ncxContents, &ncx); err != nil {
		t.Fatalf("toc.ncx is not well-formed XML: %s\n%s", err, ncxContents)
	}

	for _, m := range ncx.Head.Meta {
		switch m.Name {
		case tocNcxMetaUID:
			if want := e.Identifier(); m.Content != want {
				t.Errorf("%s content doesn't match the identifier\nGot: %s\nExpected: %s", tocNcxMetaUID, m.Content, want)
			}
		case tocNcxMetaDepth:
			// A TOC without entries still has a depth of 1.
			if m.Content != "1" {
				t.Errorf("%s content doesn't match the minimum depth\nGot: %s\nExpected: 1", tocNcxMetaDepth, m.Content)
			}
		}
	}

	cleanup(testEpubFilename, tempDir)
}

// The nav document's <title> must carry the book title. A rewrite that only
// applied the language attributes silently emptied it, and every reader that
// shows the TOC title rendered a blank heading.
func TestTocNavTitle(t *testing.T) {
	e, err := NewEpub(testEpubTitle)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := e.AddSection(testSectionBody, testSectionTitle, testSectionFilename, ""); err != nil {
		t.Fatal(err)
	}

	// SetTitle is applied at write time, so renaming after the section was
	// added must still reach nav.xhtml.
	const updatedTitle = "A Renamed Book"
	e.SetTitle(updatedTitle)

	tempDir := writeAndExtractEpub(t, e, testEpubFilename)

	navContents, err := storage.ReadFile(filesystem, filepath.Join(tempDir, contentFolderName, tocNavFilename))
	if err != nil {
		t.Fatal(err)
	}

	var nav struct {
		Head struct {
			Title string `xml:"title"`
		} `xml:"head"`
	}
	if err := xml.Unmarshal(navContents, &nav); err != nil {
		t.Fatalf("nav.xhtml is not well-formed XML: %s\n%s", err, navContents)
	}

	if nav.Head.Title != updatedTitle {
		t.Errorf("nav.xhtml title doesn't match the book title\nGot: %q\nExpected: %q\nIn: %s",
			nav.Head.Title, updatedTitle, navContents)
	}

	cleanup(testEpubFilename, tempDir)
}
