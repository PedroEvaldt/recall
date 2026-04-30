package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/PedroEvaldt/recall/internal/api"
)

func TestPrintDocumentRendersMarkdownFiles(t *testing.T) {
	doc := api.DocumentResponse{
		Filename: "notes.md",
		MimeType: "application/octet-stream",
	}
	body := strings.NewReader("## Titulo\n\n- item\n")

	var out bytes.Buffer
	if err := printDocument(&out, doc, body); err != nil {
		t.Fatalf("printDocument returned error: %v", err)
	}

	got := out.String()
	if strings.Contains(got, "## Titulo") {
		t.Fatalf("expected markdown heading marker to be rendered away, got %q", got)
	}
	if !strings.Contains(got, "Titulo") {
		t.Fatalf("expected rendered output to contain heading text, got %q", got)
	}
}

func TestPrintDocumentKeepsNonMarkdownFilesRaw(t *testing.T) {
	doc := api.DocumentResponse{
		Filename: "notes.txt",
		MimeType: "text/plain",
	}
	body := strings.NewReader("## Titulo\n")

	var out bytes.Buffer
	if err := printDocument(&out, doc, body); err != nil {
		t.Fatalf("printDocument returned error: %v", err)
	}

	if got := out.String(); got != "## Titulo\n" {
		t.Fatalf("expected raw output, got %q", got)
	}
}
