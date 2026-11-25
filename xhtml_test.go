package epub

import (
	"encoding/xml"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-shiori/go-epub/internal/storage"
)

const testZ3998Prefix = "z3998: http://www.daisy.org/z3998/2012/vocab/structure/"

func TestSectionLang(t *testing.T) {
	e, err := NewEpub(testEpubTitle)
	if err != nil {
		t.Fatal(err)
	}

	sectionPath, err := e.AddSection(testSectionBody, testSectionTitle, testSectionFilename, "")
	if err != nil {
		t.Fatal(err)
	}

	// SetLang is applied at write time, so setting it after the section was
	// added must still tag the document root.
	e.SetLang(testEpubLang)

	tempDir := writeAndExtractEpub(t, e, testEpubFilename)

	contents, err := storage.ReadFile(filesystem, filepath.Join(tempDir, contentFolderName, xhtmlFolderName, sectionPath))
	if err != nil {
		t.Fatal(err)
	}

	if err := xml.Unmarshal(contents, new(struct{})); err != nil {
		t.Fatalf("Section file is not well-formed XML: %s\n%s", err, contents)
	}

	testRootElement := `<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="` + xmlnsEpub + `" lang="` + testEpubLang + `" xml:lang="` + testEpubLang + `">`
	if !strings.Contains(string(contents), testRootElement) {
		t.Errorf(
			"Section root element is missing lang/xml:lang\n"+
				"Looking for: %s\n"+
				"In: %s",
			testRootElement,
			contents)
	}

	cleanup(testEpubFilename, tempDir)
}

func TestSectionLangDefaultsToEnglish(t *testing.T) {
	e, err := NewEpub(testEpubTitle)
	if err != nil {
		t.Fatal(err)
	}

	sectionPath, err := e.AddSection(testSectionBody, testSectionTitle, testSectionFilename, "")
	if err != nil {
		t.Fatal(err)
	}

	tempDir := writeAndExtractEpub(t, e, testEpubFilename)

	contents, err := storage.ReadFile(filesystem, filepath.Join(tempDir, contentFolderName, xhtmlFolderName, sectionPath))
	if err != nil {
		t.Fatal(err)
	}

	testRootElement := `lang="` + defaultEpubLang + `" xml:lang="` + defaultEpubLang + `"`
	if !strings.Contains(string(contents), testRootElement) {
		t.Errorf(
			"Section root element is missing the default lang/xml:lang\n"+
				"Looking for: %s\n"+
				"In: %s",
			testRootElement,
			contents)
	}

	cleanup(testEpubFilename, tempDir)
}

func TestSetSectionPrefix(t *testing.T) {
	e, err := NewEpub(testEpubTitle)
	if err != nil {
		t.Fatal(err)
	}

	castPath, err := e.AddSection(testSectionBody, "Dramatis Personae", "voices.xhtml", "")
	if err != nil {
		t.Fatal(err)
	}
	otherPath, err := e.AddSection(testSectionBody, testSectionTitle, testSectionFilename, "")
	if err != nil {
		t.Fatal(err)
	}

	if err := e.SetSectionPrefix(castPath, testZ3998Prefix); err != nil {
		t.Errorf("Error setting section prefix: %s", err)
	}

	tempDir := writeAndExtractEpub(t, e, testEpubFilename)

	castContents, err := storage.ReadFile(filesystem, filepath.Join(tempDir, contentFolderName, xhtmlFolderName, castPath))
	if err != nil {
		t.Fatal(err)
	}
	if err := xml.Unmarshal(castContents, new(struct{})); err != nil {
		t.Fatalf("Cast page is not well-formed XML: %s\n%s", err, castContents)
	}

	testPrefixAttribute := `epub:prefix="` + testZ3998Prefix + `"`
	if !strings.Contains(string(castContents), testPrefixAttribute) {
		t.Errorf(
			"Cast page root element is missing the epub:prefix declaration\n"+
				"Looking for: %s\n"+
				"In: %s",
			testPrefixAttribute,
			castContents)
	}

	otherContents, err := storage.ReadFile(filesystem, filepath.Join(tempDir, contentFolderName, xhtmlFolderName, otherPath))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(otherContents), "epub:prefix") {
		t.Errorf(
			"Section without a prefix declared one\n"+
				"In: %s",
			otherContents)
	}

	cleanup(testEpubFilename, tempDir)
}

func TestSetSectionPrefixSubSection(t *testing.T) {
	e, err := NewEpub(testEpubTitle)
	if err != nil {
		t.Fatal(err)
	}

	parentPath, err := e.AddSection(testSectionBody, testSectionTitle, testSectionFilename, "")
	if err != nil {
		t.Fatal(err)
	}
	subSectionPath, err := e.AddSubSection(parentPath, testSectionBody, "Subsection", "voices.xhtml", "")
	if err != nil {
		t.Fatal(err)
	}

	if err := e.SetSectionPrefix(subSectionPath, testZ3998Prefix); err != nil {
		t.Errorf("Error setting subsection prefix: %s", err)
	}

	tempDir := writeAndExtractEpub(t, e, testEpubFilename)

	contents, err := storage.ReadFile(filesystem, filepath.Join(tempDir, contentFolderName, xhtmlFolderName, subSectionPath))
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(contents), `epub:prefix="`+testZ3998Prefix+`"`) {
		t.Errorf(
			"Subsection root element is missing the epub:prefix declaration\n"+
				"In: %s",
			contents)
	}

	cleanup(testEpubFilename, tempDir)
}

func TestSetSectionPrefixMissingSection(t *testing.T) {
	e, err := NewEpub(testEpubTitle)
	if err != nil {
		t.Fatal(err)
	}

	if err := e.SetSectionPrefix("does-not-exist.xhtml", testZ3998Prefix); err == nil {
		t.Error("Expected an error when setting the prefix of a section that doesn't exist")
	}
}
