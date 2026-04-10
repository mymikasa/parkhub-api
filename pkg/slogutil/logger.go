package slogutil

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
)

type Options struct {
	Level  string
	Format string
}

func NewLogger(w io.Writer, opts Options) (*slog.Logger, error) {
	level, err := parseLevel(opts.Level)
	if err != nil {
		return nil, err
	}

	handlerOptions := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	switch strings.ToLower(strings.TrimSpace(opts.Format)) {
	case "", "text":
		handler = slog.NewTextHandler(w, handlerOptions)
	case "json":
		handler = slog.NewJSONHandler(w, handlerOptions)
	default:
		return nil, fmt.Errorf("unsupported log format %q", opts.Format)
	}

	return slog.New(NewTraceContextHandler(handler)), nil
}

func parseLevel(value string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("unsupported log level %q", value)
	}
}
