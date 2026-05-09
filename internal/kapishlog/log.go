// Package kapishlog wires log/slog with sensible defaults: JSON output,
// configurable level, optional rotated file output via lumberjack.
package kapishlog

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Options control logger creation.
type Options struct {
	// Level is one of "debug", "info", "warn", "error". Empty is treated as "info".
	Level string

	// Writer is where logs go. If nil, FilePath is used; if both are empty,
	// logs go to stderr.
	Writer io.Writer

	// FilePath, when set and Writer is nil, opens a rotating log file at
	// that path. "-" means stderr (useful for `--log-file -`).
	FilePath string
}

// New returns a *slog.Logger configured per opts.
func New(opts Options) (*slog.Logger, error) {
	level, err := parseLevel(opts.Level)
	if err != nil {
		return nil, err
	}

	w := opts.Writer
	if w == nil {
		switch opts.FilePath {
		case "", "-":
			w = os.Stderr
		default:
			if err := os.MkdirAll(filepath.Dir(opts.FilePath), 0o700); err != nil {
				return nil, fmt.Errorf("kapishlog: mkdir %s: %w", filepath.Dir(opts.FilePath), err)
			}
			w = &lumberjack.Logger{
				Filename:   opts.FilePath,
				MaxSize:    10, // MB
				MaxBackups: 3,
				LocalTime:  true,
				Compress:   false,
			}
		}
	}

	h := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})
	return slog.New(h), nil
}

func parseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, errors.New("kapishlog: unknown log level " + s)
	}
}
