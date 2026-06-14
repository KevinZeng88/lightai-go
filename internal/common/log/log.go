// Package log provides structured file+stdout dual-write logging for LightAI Go.
package log

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// Config holds logging configuration.
type Config struct {
	Level       string
	Dir         string
	File        string
	Stdout      bool
	FileEnabled bool
	MaxSizeMB   int
	MaxFiles    int
}

// Init initializes the global structured logger.
//
// When FileEnabled is true, a JSON handler writes to the specified log file
// (under Dir).  The log directory is created automatically.  The file is
// opened in append mode.  When MaxSizeMB > 0, the file is rotated before
// writing if it already exceeds the limit (old file is renamed with a
// ".1" / ".2" … suffix, keeping at most MaxFiles rotated copies).
//
// When Stdout is true, a second JSON handler writes to os.Stdout in
// parallel so that nohup-style wrapper logs still capture everything.
func Init(cfg Config) {
	var lvl slog.Level
	switch strings.ToLower(cfg.Level) {
	case "debug":
		lvl = slog.LevelDebug
	case "info":
		lvl = slog.LevelInfo
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: lvl}

	var writers []io.Writer

	// File output.
	if cfg.FileEnabled {
		if cfg.Dir == "" {
			cfg.Dir = "logs"
		}
		if cfg.File == "" {
			cfg.File = "lightai.log"
		}
		if err := os.MkdirAll(cfg.Dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "log: cannot create log directory %s: %v\n", cfg.Dir, err)
		} else {
			fpath := filepath.Join(cfg.Dir, cfg.File)
			f, err := openLogFile(fpath, cfg.MaxSizeMB, cfg.MaxFiles)
			if err != nil {
				fmt.Fprintf(os.Stderr, "log: cannot open log file %s: %v\n", fpath, err)
			} else {
				writers = append(writers, f)
			}
		}
	}

	// Stdout output.
	if cfg.Stdout {
		writers = append(writers, os.Stdout)
	}

	// Fallback: always write to stderr so logs are never silently lost.
	if len(writers) == 0 {
		writers = append(writers, os.Stderr)
	}

	var sink io.Writer
	if len(writers) == 1 {
		sink = writers[0]
	} else {
		sink = io.MultiWriter(writers...)
	}

	handler := slog.NewJSONHandler(sink, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

// openLogFile opens path for append.  If maxSizeMB > 0 and the existing file
// exceeds the limit, it rotates old files before opening a fresh one.
func openLogFile(path string, maxSizeMB, maxFiles int) (*os.File, error) {
	if maxSizeMB > 0 {
		if fi, err := os.Stat(path); err == nil && fi.Size() > int64(maxSizeMB)*1024*1024 {
			rotateLogFiles(path, maxFiles)
		}
	}
	return os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
}

// rotateLogFiles renames path → path.1, path.1 → path.2, etc.,
// keeping at most keep rotated copies.
func rotateLogFiles(path string, keep int) {
	if keep <= 0 {
		keep = 5
	}
	// Remove oldest.
	oldest := fmt.Sprintf("%s.%d", path, keep)
	os.Remove(oldest)
	// Shift.
	for i := keep - 1; i >= 1; i-- {
		old := fmt.Sprintf("%s.%d", path, i)
		new := fmt.Sprintf("%s.%d", path, i+1)
		os.Rename(old, new)
	}
	os.Rename(path, fmt.Sprintf("%s.1", path))
}

// Debug logs a debug message.
func Debug(msg string, args ...any) {
	slog.Debug(msg, args...)
}

// Info logs an info message.
func Info(msg string, args ...any) {
	slog.Info(msg, args...)
}

// Warn logs a warning message.
func Warn(msg string, args ...any) {
	slog.Warn(msg, args...)
}

// Error logs an error message.
func Error(msg string, args ...any) {
	slog.Error(msg, args...)
}

// Fatal logs an error message and exits.
func Fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}

// With returns a logger with additional context.
func With(args ...any) *slog.Logger {
	return slog.With(args...)
}
