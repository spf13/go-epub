package epub

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-shiori/go-epub/internal/storage"
)

const (
	testSource          = "https://www.gutenberg.org/ebooks/11"
	testSourceTemplate  = `<dc:source>%s</dc:source>`
	testContributor1    = "Jane Smith"
	testContributor1Role = "edt"
	testContributor2    = "AI Illustrator"
	testContributor2Role = "ill"
	testContributorTemplate = `<dc:contributor id="%s">%s</dc:contributor>`
	testMetaProperty    = "schema:accessMode"
	testMetaValue       = "textual"
	testMetaTemplate    = `<meta property="%s">%s</meta>`
)

func TestEpubSetSource(t *testing.T) {
	e, err := NewEpub(testEpubTitle)
	if err != nil {
		t.Fatal(err)
	}

	e.SetSource(testSource)

	tempDir := writeAndExtractEpub(t, e, testEpubFilename)

	contents, err := storage.ReadFile(filesystem, filepath.Join(tempDir, contentFolderName, pkgFilename))
	if err != nil {
		t.Errorf("Unexpected error reading package file: %s", err)
	}

	testSourceElement := "<dc:source>" + testSource + "</dc:source>"
	if !strings.Contains(string(contents), testSourceElement) {
		t.Errorf(
			"Source element not found in package file\n"+
				"Looking for: %s\n"+
				"In: %s",
			testSourceElement,
			contents)
	}

	cleanup(testEpubFilename, tempDir)
}

func TestEpubAddContributor(t *testing.T) {
	e, err := NewEpub(testEpubTitle)
	if err != nil {
		t.Fatal(err)
	}

	e.AddContributor(testContributor1, testContributor1Role)
	e.AddContributor(testContributor2, testContributor2Role)

	tempDir := writeAndExtractEpub(t, e, testEpubFilename)

	contents, err := storage.ReadFile(filesystem, filepath.Join(tempDir, contentFolderName, pkgFilename))
	if err != nil {
		t.Errorf("Unexpected error reading package file: %s", err)
	}

	// Check that both contributors are present
	testContributor1Element := `<dc:contributor id="` + testContributor1Role + `">` + testContributor1 + `</dc:contributor>`
	if !strings.Contains(string(contents), testContributor1Element) {
		t.Errorf(
			"Contributor 1 element not found in package file\n"+
				"Looking for: %s\n"+
				"In: %s",
			testContributor1Element,
			contents)
	}

	testContributor2Element := `<dc:contributor id="` + testContributor2Role + `">` + testContributor2 + `</dc:contributor>`
	if !strings.Contains(string(contents), testContributor2Element) {
		t.Errorf(
			"Contributor 2 element not found in package file\n"+
				"Looking for: %s\n"+
				"In: %s",
			testContributor2Element,
			contents)
	}

	cleanup(testEpubFilename, tempDir)
}

func TestEpubAddMeta(t *testing.T) {
	e, err := NewEpub(testEpubTitle)
	if err != nil {
		t.Fatal(err)
	}

	e.AddMeta(testMetaProperty, testMetaValue)

	tempDir := writeAndExtractEpub(t, e, testEpubFilename)

	contents, err := storage.ReadFile(filesystem, filepath.Join(tempDir, contentFolderName, pkgFilename))
	if err != nil {
		t.Errorf("Unexpected error reading package file: %s", err)
	}

	testMetaElement := `<meta property="` + testMetaProperty + `">` + testMetaValue + `</meta>`
	if !strings.Contains(string(contents), testMetaElement) {
		t.Errorf(
			"Meta element not found in package file\n"+
				"Looking for: %s\n"+
				"In: %s",
			testMetaElement,
			contents)
	}

	cleanup(testEpubFilename, tempDir)
}

func TestEpubAccessibilityMetadata(t *testing.T) {
	e, err := NewEpub(testEpubTitle)
	if err != nil {
		t.Fatal(err)
	}

	// Add standard accessibility metadata as recommended by EPUB Accessibility spec
	e.AddMeta("schema:accessMode", "textual")
	e.AddMeta("schema:accessMode", "visual")
	e.AddMeta("schema:accessibilityFeature", "structuralNavigation")
	e.AddMeta("schema:accessibilityFeature", "alternativeText")
	e.AddMeta("schema:accessibilityHazard", "none")
	e.AddMeta("schema:accessibilitySummary", "This publication conforms to WCAG 2.0 Level AA.")

	tempDir := writeAndExtractEpub(t, e, testEpubFilename)

	contents, err := storage.ReadFile(filesystem, filepath.Join(tempDir, contentFolderName, pkgFilename))
	if err != nil {
		t.Errorf("Unexpected error reading package file: %s", err)
	}

	contentStr := string(contents)

	expectedMeta := []string{
		`<meta property="schema:accessMode">textual</meta>`,
		`<meta property="schema:accessMode">visual</meta>`,
		`<meta property="schema:accessibilityFeature">structuralNavigation</meta>`,
		`<meta property="schema:accessibilityFeature">alternativeText</meta>`,
		`<meta property="schema:accessibilityHazard">none</meta>`,
		`<meta property="schema:accessibilitySummary">This publication conforms to WCAG 2.0 Level AA.</meta>`,
	}

	for _, expected := range expectedMeta {
		if !strings.Contains(contentStr, expected) {
			t.Errorf(
				"Expected meta element not found in package file\n"+
					"Looking for: %s\n"+
					"In: %s",
				expected,
				contentStr)
		}
	}

	cleanup(testEpubFilename, tempDir)
}

func TestEpubCompleteMetadata(t *testing.T) {
	e, err := NewEpub(testEpubTitle)
	if err != nil {
		t.Fatal(err)
	}

	// Set all metadata fields
	e.SetAuthor("Lewis Carroll")
	e.SetDescription("A girl falls down a rabbit hole into a fantasy world.")
	e.SetLang("en")
	e.SetPublisher("Brightleaf Editions")
	e.SetRights("Public Domain")
	e.SetDate("2025-01-01")
	e.SetSubjects([]string{"Fiction", "Fantasy", "Classic Literature"})
	e.SetSource("https://www.gutenberg.org/ebooks/11")
	e.AddContributor("Digital Editor", "edt")
	e.AddMeta("schema:accessMode", "textual")

	tempDir := writeAndExtractEpub(t, e, testEpubFilename)

	contents, err := storage.ReadFile(filesystem, filepath.Join(tempDir, contentFolderName, pkgFilename))
	if err != nil {
		t.Errorf("Unexpected error reading package file: %s", err)
	}

	contentStr := string(contents)

	// Verify all elements are present
	checks := map[string]string{
		"author":      `<dc:creator id="creator">Lewis Carroll</dc:creator>`,
		"description": `<dc:description>A girl falls down a rabbit hole into a fantasy world.</dc:description>`,
		"language":    `<dc:language>en</dc:language>`,
		"publisher":   `<dc:publisher>Brightleaf Editions</dc:publisher>`,
		"rights":      `<dc:rights>Public Domain</dc:rights>`,
		"date":        `<dc:date>2025-01-01</dc:date>`,
		"subject1":    `<dc:subject>Fiction</dc:subject>`,
		"subject2":    `<dc:subject>Fantasy</dc:subject>`,
		"subject3":    `<dc:subject>Classic Literature</dc:subject>`,
		"source":      `<dc:source>https://www.gutenberg.org/ebooks/11</dc:source>`,
		"contributor": `<dc:contributor id="edt">Digital Editor</dc:contributor>`,
		"accessMode":  `<meta property="schema:accessMode">textual</meta>`,
	}

	for name, expected := range checks {
		if !strings.Contains(contentStr, expected) {
			t.Errorf(
				"%s element not found in package file\n"+
					"Looking for: %s\n"+
					"In: %s",
				name,
				expected,
				contentStr)
		}
	}

	cleanup(testEpubFilename, tempDir)
}
