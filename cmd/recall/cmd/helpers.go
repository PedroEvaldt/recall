package cmd

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/PedroEvaldt/recall/internal/api"
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

func isMarkdownDocument(doc api.DocumentResponse) bool {
	mimeType := strings.ToLower(doc.MimeType)
	extension := strings.ToLower(filepath.Ext(doc.Filename))

	return strings.Contains(mimeType, "markdown") ||
		mimeType == "text/x-markdown" ||
		extension == ".md" ||
		extension == ".markdown"
}
