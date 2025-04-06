// Package utils provides common utilities shared across packages
package utils

// Logger defines a common logging interface used throughout the application
type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
}

// NoopLogger is a logger implementation that does nothing
type NoopLogger struct{}

func (l NoopLogger) Debug(format string, args ...interface{}) {}
func (l NoopLogger) Info(format string, args ...interface{})  {}
func (l NoopLogger) Warn(format string, args ...interface{})  {}
func (l NoopLogger) Error(format string, args ...interface{}) {}
