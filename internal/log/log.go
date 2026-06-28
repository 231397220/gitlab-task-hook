package log

import (
	"io"
	"log/slog"
	"os"
)

// Logger wraps slog.Logger with JSON structured output.
type Logger struct {
	*slog.Logger
}

// New creates a Logger writing to w at the given level string (debug/info/warn/error).
func New(w io.Writer, level string) *Logger {
	var lvl slog.Level
	_ = lvl.UnmarshalText([]byte(level))
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: lvl})
	return &Logger{slog.New(h)}
}

// NewStderr returns a Logger writing to stderr at info level.
func NewStderr() *Logger { return New(os.Stderr, "info") }

// NewFile opens or creates path for appending and returns a Logger writing to it.
// The caller must invoke the returned close function when done.
func NewFile(path, level string) (*Logger, func(), error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0640)
	if err != nil {
		return nil, nil, err
	}
	return New(f, level), func() { f.Close() }, nil
}
