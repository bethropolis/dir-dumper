package logger

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/fatih/color"
)

// LogLevel defines log severity levels
type LogLevel int

const (
	// Log levels from least to most restrictive
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelNone
)

// Logger provides structured logging with levels
type Logger struct {
	out         io.Writer
	useColors   bool
	level       LogLevel
	VerboseMode bool // Legacy flag, maps to Debug level
}

// New creates a new Logger with the given settings
func New(out io.Writer, verbose bool, useColors bool) *Logger {
	level := LevelInfo
	if verbose {
		level = LevelDebug
	}

	return &Logger{
		out:         out,
		useColors:   useColors,
		level:       level,
		VerboseMode: verbose,
	}
}

// WithLevel sets the log level and returns the logger
func (l *Logger) WithLevel(level LogLevel) *Logger {
	l.level = level
	// Keep VerboseMode in sync for backward compatibility
	l.VerboseMode = (level <= LevelDebug)
	return l
}

// SetLevel sets the log level
func (l *Logger) SetLevel(levelStr string) {
	level := parseLogLevel(levelStr)
	l.WithLevel(level)
}

// parseLogLevel converts a string level to LogLevel
func parseLogLevel(level string) LogLevel {
	switch strings.ToLower(level) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	case "none", "off":
		return LevelNone
	default:
		return LevelInfo // Default to Info level
	}
}

// Debug logs a debug message if verbose mode is enabled
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.level <= LevelDebug {
		prefix := "DEBUG"
		if l.useColors {
			prefix = color.CyanString(prefix)
		}
		fmt.Fprintf(l.out, "[%s %s] %s\n", timeString(), prefix, fmt.Sprintf(format, args...))
	}
}

// Info logs an informational message (standard level)
func (l *Logger) Info(format string, args ...interface{}) {
	if l.level <= LevelInfo {
		prefix := "INFO"
		if l.useColors {
			prefix = color.BlueString(prefix)
		}
		fmt.Fprintf(l.out, "[%s %s] %s\n", timeString(), prefix, fmt.Sprintf(format, args...))
	}
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	if l.level <= LevelWarn {
		prefix := "WARN"
		if l.useColors {
			prefix = color.YellowString(prefix)
		}
		fmt.Fprintf(l.out, "[%s %s] %s\n", timeString(), prefix, fmt.Sprintf(format, args...))
	}
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	if l.level <= LevelError {
		prefix := "ERROR"
		if l.useColors {
			prefix = color.RedString(prefix)
		}
		fmt.Fprintf(l.out, "[%s %s] %s\n", timeString(), prefix, fmt.Sprintf(format, args...))
	}
}

// timeString returns a formatted time string for the log prefix
func timeString() string {
	return time.Now().Format("15:04:05.000")
}
