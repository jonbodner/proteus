package proteus

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func RegisterLineLogger(t *testing.T) func() []string {
	defaultLogger := slog.Default()
	t.Cleanup(func() {
		slog.SetDefault(defaultLogger)
	})
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)
	return func() []string {
		lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
		buf.Reset()
		return lines
	}
}
