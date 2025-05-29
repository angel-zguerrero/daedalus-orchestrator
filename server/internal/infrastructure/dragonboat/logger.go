package dragonboat

import (
	"errors"
	"fmt" // Used for errors.New(fmt.Sprintf(...))

	dblog "github.com/lni/dragonboat/v4/logger"
	"github.com/rs/zerolog/log"
)

// zerologgerLogger is an adapter that implements Dragonboat's dblog.ILogger interface
// using the zerolog library. This allows Dragonboat internal logs to be processed
// by the application's zerolog setup.
type zerologgerLogger struct {
	name string // The name of the logger, typically identifying the Dragonboat component.
	// LogLevel is embedded from dblog.LogLevel but not actively used by this adapter,
	// as zerolog's global level or context-specific levels are used instead.
	dblog.LogLevel
}

// Infof logs an informational message using zerolog.
// It includes the logger's name as a field.
func (l *zerologgerLogger) Infof(format string, args ...interface{}) {
	log.Info().Str("name", l.name).Msgf(format, args...)
}

// Warningf logs a warning message using zerolog.
// It includes the logger's name as a field.
func (l *zerologgerLogger) Warningf(format string, args ...interface{}) {
	log.Warn().Str("name", l.name).Msgf(format, args...)
}

// Errorf logs an error message using zerolog.
// It creates an error from the format string and arguments and includes the logger's name as a 'source' field.
func (l *zerologgerLogger) Errorf(format string, args ...interface{}) {
	log.Error().Err(errors.New(fmt.Sprintf(format, args...))).Str("source", l.name).Msgf(format, args...)
}

// Debugf logs a debug message using zerolog.
// It includes the logger's name as a field.
func (l *zerologgerLogger) Debugf(format string, args ...interface{}) {
	log.Debug().Str("name", l.name).Msgf(format, args...)
}

// Panicf logs a panic message using zerolog and then panics.
// It includes the logger's name as a field.
func (l *zerologgerLogger) Panicf(format string, args ...interface{}) {
	log.Panic().Str("name", l.name).Msgf(format, args...)
}

// SetLevel is part of the dblog.ILogger interface. In this implementation,
// it's a no-op because zerolog's log level is typically controlled globally or
// through its own context-aware mechanisms, not on a per-logger-instance basis
// in the same way Dragonboat's logger expects.
func (l *zerologgerLogger) SetLevel(logLevel dblog.LogLevel) {
	// Using zerolog's global level or context-based level control.
	// This method is intentionally a no-op.
}

// fmtLoggerFactory is an unused type, presumably from a previous logging approach or example.
// It's not used by CreateZerologger.
type fmtLoggerFactory struct{}

// CreateZerologger is a factory function that creates a new dblog.ILogger
// instance that is backed by zerolog.
// This function is intended to be used with Dragonboat's dblog.SetLoggerFactory.
//
// Parameters:
//   - name: The name for the logger, usually indicating the Dragonboat component (e.g., "raft", "transport").
//
// Returns:
//   - An dblog.ILogger implementation that routes logs to zerolog.
func CreateZerologger(name string) dblog.ILogger {
	return &zerologgerLogger{name: name}
}
