package logging

import (
	"io"
	"log/slog"
	"strings"
)

// New returns a *slog.Logger configured by level and format.
// level accepts "debug", "info", "warn", or "error" (defaults to info when empty or unknown).
// format accepts "json" or "text" (defaults to text when empty or unknown).
func New(w io.Writer, level, format string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: lvl}
	if strings.ToLower(format) == "json" {
		return slog.New(slog.NewJSONHandler(w, opts))
	}
	return slog.New(slog.NewTextHandler(w, opts))
}
