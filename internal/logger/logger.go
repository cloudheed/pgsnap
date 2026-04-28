package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// Logger wraps zerolog.Logger for application logging.
type Logger struct {
	zerolog.Logger
}

// Options configures the logger.
type Options struct {
	Level  string // debug, info, warn, error
	Format string // json, console
	Output io.Writer
}

// New creates a new Logger with the given options.
func New(opts Options) *Logger {
	if opts.Output == nil {
		opts.Output = os.Stderr
	}

	var output io.Writer = opts.Output

	// Use console writer for human-readable output
	if opts.Format == "console" {
		output = zerolog.ConsoleWriter{
			Out:        opts.Output,
			TimeFormat: time.RFC3339,
		}
	}

	level := parseLevel(opts.Level)

	logger := zerolog.New(output).
		Level(level).
		With().
		Timestamp().
		Logger()

	return &Logger{Logger: logger}
}

// Default returns a default console logger at info level.
func Default() *Logger {
	return New(Options{
		Level:  "info",
		Format: "console",
	})
}

func parseLevel(level string) zerolog.Level {
	switch level {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}
