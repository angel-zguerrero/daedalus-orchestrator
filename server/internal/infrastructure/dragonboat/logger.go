package dragonboat

import (
	"errors"
	"fmt" // Used for errors.New(fmt.Sprintf(...))

	dblog "github.com/lni/dragonboat/v4/logger"
	"github.com/rs/zerolog/log"
)

type zerologgerLogger struct {
	name string
	dblog.LogLevel
}

func (l *zerologgerLogger) Infof(format string, args ...interface{}) {
	log.Info().Str("name", l.name).Msgf(format, args...)
}

func (l *zerologgerLogger) Warningf(format string, args ...interface{}) {
	log.Warn().Str("name", l.name).Msgf(format, args...)
}

func (l *zerologgerLogger) Errorf(format string, args ...interface{}) {
	log.Error().Err(errors.New(fmt.Sprintf(format, args...))).Str("source", l.name).Msgf(format, args...)
}

func (l *zerologgerLogger) Debugf(format string, args ...interface{}) {
	log.Debug().Str("name", l.name).Msgf(format, args...)
}
func (l *zerologgerLogger) Panicf(format string, args ...interface{}) {
	log.Panic().Str("name", l.name).Msgf(format, args...)
}

func (l *zerologgerLogger) SetLevel(logLevel dblog.LogLevel) {
	//using zerologger level
}

// LoggerFactory que crea fmtLogger
type fmtLoggerFactory struct{}

func CreateZerologger(name string) dblog.ILogger {
	return &zerologgerLogger{name: name}
}
