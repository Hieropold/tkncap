/**
 * package logging
 *
 * <purpose-start>
 * Configures the global slog logger for the tkncap process. The log level is
 * read from the TKNCAP_LOG_LEVEL environment variable at startup. If the
 * variable is absent or empty, logging is disabled (all logs are discarded).
 * Valid values are "debug", "info", "warn", "error" (case-insensitive).
 *
 * Call Setup once at process start (before any slog calls) so that all
 * packages share the same logger configuration.
 * <purpose-end>
 *
 * <inputs-start>
 * - TKNCAP_LOG_LEVEL environment variable (read via os.Getenv internally).
 * <inputs-end>
 *
 * <outputs-start>
 * - Replaces the global slog default logger with a text-format handler at the
 *   configured level, or a discarding handler if logging is not enabled.
 * <side-effects-start>
 * - Calls slog.SetDefault, which affects all subsequent slog calls in the
 *   process.
 * <side-effects-end>
 */
package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

/**
 * Setup
 *
 * <purpose-start>
 * Initialises the global slog logger. Reads TKNCAP_LOG_LEVEL from the
 * environment and maps it to a slog.Level. If missing or empty, configures a
 * discarding handler so no logs are printed. Unrecognised values fall back
 * to slog.LevelInfo. Uses a TextHandler writing to stderr so that
 * log output does not interfere with stdout-rendered quota tables or JSON.
 * <purpose-end>
 *
 * <inputs-start>
 * - None (reads TKNCAP_LOG_LEVEL from os.Getenv internally).
 * <inputs-end>
 *
 * <outputs-start>
 * - None (configures the global logger as a side effect).
 * <outputs-end>
 *
 * <side-effects-start>
 * - Sets the global slog default logger via slog.SetDefault.
 * <side-effects-end>
 */
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
