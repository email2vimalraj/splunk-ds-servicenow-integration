package logging

import (
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/example/splunk-ds-camr/internal/config"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Init configures global slog default logger with JSON handler and file rotation.
// Returns a cleanup function to close any resources if needed (currently noop).
func Init(c config.LoggingConfig) func() {
	// Build writers
	var writers []io.Writer
	compress := true
	if c.Compress != nil {
		compress = *c.Compress
	}
	stdout := true
	if c.Stdout != nil {
		stdout = *c.Stdout
	}
	// ensure directory exists
	if dir := filepath.Dir(c.File); dir != "." && dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}
	lj := &lumberjack.Logger{
		Filename:   c.File,
		MaxSize:    c.MaxSizeMB,
		MaxBackups: c.MaxBackups,
		MaxAge:     c.MaxAgeDays,
		Compress:   compress,
	}
	writers = append(writers, lj)
	if stdout {
		writers = append(writers, os.Stdout)
	}
	mw := io.MultiWriter(writers...)

	// Level
	var lvl slog.Level
	switch strings.ToLower(c.Level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	h := slog.NewJSONHandler(mw, &slog.HandlerOptions{Level: lvl})
	logger := slog.New(h)
	slog.SetDefault(logger)
	// bridge standard log to slog at the same level
	std := slog.NewLogLogger(h, lvl)
	log.SetOutput(std.Writer())
	log.SetFlags(0)
	return func() {
		// lumberjack doesn't require explicit Close; noop
		_ = lj
	}
}
