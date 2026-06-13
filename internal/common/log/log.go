// Package log provides structured logging for LightAI Go.
package log

import (
	"log/slog"
	"os"
	"strings"
)

// Init initializes the global logger with the given level.
func Init(level string) {
	var lvl slog.Level
	switch strings.ToLower(level) {
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

	opts := &slog.HandlerOptions{
		Level: lvl,
	}
	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)
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
