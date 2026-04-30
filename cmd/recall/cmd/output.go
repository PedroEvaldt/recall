package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/PedroEvaldt/recall/internal/api"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
)

var errSilent = errors.New("silent error")

const (
	ansiReset  = "\033[0m"
	ansiBold   = "\033[1m"
	ansiRed    = "\033[31m"
	ansiYellow = "\033[33m"
)

func printError(w io.Writer, err error) {
	fmt.Fprintf(w, "%serror%s %v\n", ansiRed+ansiBold, ansiReset, err)
}

func printNoResults(w io.Writer, query string) {
	fmt.Fprintf(w, "%sno results%s no documents found for %q\n", ansiYellow+ansiBold, ansiReset, query)
}

func printMissingContent(w io.Writer, title string) {
	fmt.Fprintf(w, "%smissing content%s document %q was found, but its content is missing\n", ansiYellow+ansiBold, ansiReset, title)
}

func printDocument(w io.Writer, doc api.DocumentResponse, body io.Reader) error {
	if !isMarkdownDocument(doc) {
		_, err := io.Copy(w, body)
		return err
	}

	content, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("read markdown content: %w", err)
	}

	rendered, err := renderMarkdown(string(content))
	if err != nil {
		_, writeErr := io.Copy(w, bytes.NewReader(content))
		if writeErr != nil {
			return writeErr
		}
		return nil
	}

	_, err = fmt.Fprint(w, rendered)
	return err
}

func isMarkdownDocument(doc api.DocumentResponse) bool {
	mimeType := strings.ToLower(doc.MimeType)
	extension := strings.ToLower(filepath.Ext(doc.Filename))

	return strings.Contains(mimeType, "markdown") ||
		mimeType == "text/x-markdown" ||
		extension == ".md" ||
		extension == ".markdown"
}

func renderMarkdown(content string) (string, error) {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStyles(markdownStyle()),
		glamour.WithWordWrap(100),
	)
	if err != nil {
		return "", fmt.Errorf("create markdown renderer: %w", err)
	}

	rendered, err := renderer.Render(content)
	if err != nil {
		return "", fmt.Errorf("render markdown: %w", err)
	}

	return strings.TrimRight(rendered, "\n") + "\n", nil
}

func markdownStyle() ansi.StyleConfig {
	style := styles.DarkStyleConfig
	style.H1.Prefix = ""
	style.H2.Prefix = ""
	style.H3.Prefix = ""
	style.H4.Prefix = ""
	style.H5.Prefix = ""
	style.H6.Prefix = ""
	return style
}
