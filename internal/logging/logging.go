// Package logging configures the global slog logger for the tkncap process.
// Call Setup once at process start (before any slog calls) so all packages
// share the same configuration.
package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// Setup reads TKNCAP_LOG_LEVEL from the environment ("debug", "info", "warn",
// "error", case-insensitive) and configures slog accordingly. Logging is
// disabled entirely (a discarding handler) when the variable is absent or
// empty — the AGENTS.md logging convention only applies once a user opts in.
// An unrecognised value falls back to info. Uses a TextHandler writing to
// stderr so log output never corrupts the stdout-rendered table or JSON.
//
// Side effects: replaces the global slog default logger via slog.SetDefault.
func Setup() {
	levelStr := strings.ToLower(strings.TrimSpace(os.Getenv("TKNCAP_LOG_LEVEL")))

	if levelStr == "" {
		handler := slog.NewTextHandler(io.Discard, nil)
		slog.SetDefault(slog.New(handler))
		return
	}

	var level slog.Level
	switch levelStr {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(handler))

	slog.Debug("logging: logger initialised", "level", level.String())
}
