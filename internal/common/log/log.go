// Package log provides structured file+stdout dual-write logging for LightAI Go.
package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Config holds logging configuration.
type Config struct {
	Level         string
	Format        string // "text" or "json", default "text"
	Dir           string
	File          string
	Stdout        bool
	FileEnabled   bool
	Append        bool
	MaxSizeMB     int
	MaxFiles      int
	RetentionDays int
}

// Init initializes the global structured logger.
//
// When FileEnabled is true, a JSON handler writes to the specified log file
// (under Dir).  The log directory is created automatically.  When Append is
// true, the file is opened in append mode; otherwise it is truncated.
// When MaxSizeMB > 0, the file is rotated before writing if it already
// exceeds the limit (old file is renamed with a ".1" / ".2" … suffix,
// keeping at most MaxFiles rotated copies).  When RetentionDays > 0, log
// files older than that many days are removed at startup.
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

	// Choose handler format.
	isJSON := strings.ToLower(cfg.Format) == "json"

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
			// Clean up old log files if retention is configured.
			if cfg.RetentionDays > 0 {
				cleanOldLogs(cfg.Dir, cfg.RetentionDays)
			}

			fpath := filepath.Join(cfg.Dir, cfg.File)
			f, err := openLogFile(fpath, cfg.MaxSizeMB, cfg.MaxFiles, cfg.Append)
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

	var handler slog.Handler
		if isJSON {
			handler = slog.NewJSONHandler(sink, opts)
		} else {
			handler = newHumanHandler(sink, opts)
		}
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

// humanHandler is a slog.Handler that writes compact human-readable log lines.
// Format: "2006-01-02 15:04:05.000 INFO  message"
// In debug mode, key=value pairs are appended after the message.
type humanHandler struct {
	level slog.Level
	w     io.Writer
}

func newHumanHandler(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
	h := &humanHandler{w: w, level: slog.LevelInfo}
	if opts != nil && opts.Level != nil {
		h.level = opts.Level.Level()
	}
	return h
}

func (h *humanHandler) Enabled(_ context.Context, lvl slog.Level) bool {
	return lvl >= h.level
}

func (h *humanHandler) Handle(_ context.Context, r slog.Record) error {
	// Time: "2006-01-02 15:04:05.000"
	t := r.Time.Format("2006-01-02 15:04:05.000")
	// Level: 5-char fixed-width.
	var lvl string
	switch r.Level {
	case slog.LevelDebug:
		lvl = "DEBUG"
	case slog.LevelInfo:
		lvl = "INFO "
	case slog.LevelWarn:
		lvl = "WARN "
	case slog.LevelError:
		lvl = "ERROR"
	default:
		lvl = r.Level.String()
	}

	// Build: "2006-01-02 15:04:05.000 INFO  message"
	buf := make([]byte, 0, 256)
	buf = append(buf, t...)
	buf = append(buf, ' ')
	buf = append(buf, lvl...)
	buf = append(buf, ' ')
	buf = append(buf, r.Message...)

	// Append key=value pairs for context.
	r.Attrs(func(a slog.Attr) bool {
		buf = append(buf, ' ')
		buf = append(buf, a.Key...)
		buf = append(buf, '=')
		buf = append(buf, a.Value.String()...)
		return true
	})

	buf = append(buf, '\n')
	_, err := h.w.Write(buf)
	return err
}

func (h *humanHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *humanHandler) WithGroup(name string) slog.Handler {
	return h
}

// openLogFile opens path.  When append is true the file is opened in append
// mode; otherwise it is truncated.  If maxSizeMB > 0 and the existing file
// exceeds the limit, it rotates old files before opening a fresh one.
func openLogFile(path string, maxSizeMB, maxFiles int, appendMode bool) (*os.File, error) {
	if maxSizeMB > 0 {
		if fi, err := os.Stat(path); err == nil && fi.Size() > int64(maxSizeMB)*1024*1024 {
			rotateLogFiles(path, maxFiles)
		}
	}
	flag := os.O_CREATE | os.O_WRONLY
	if appendMode {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}
	return os.OpenFile(path, flag, 0644)
}

// cleanOldLogs removes log files under dir whose modification time is older
// than retentionDays.  Only files matching the pattern "*.log*" are considered
// (main log and rotated copies).
func cleanOldLogs(dir string, retentionDays int) {
	cutoff := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return // directory may not exist yet — not an error
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".log") && !strings.Contains(name, ".log.") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			fp := filepath.Join(dir, name)
			os.Remove(fp)
		}
	}
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
