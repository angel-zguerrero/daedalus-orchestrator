package dragonboat

import (
	"errors"
	"fmt"

	dblog "github.com/lni/dragonboat/v4/logger"
	"github.com/rs/zerolog/log"
)

type zerologgerLogger struct {
	name string
	dblog.LogLevel
}

func (l *zerologgerLogger) Infof(format string, args ...interface{}) {
	log.Info().Msg(fmt.Sprintf("%s: %s", l.name, fmt.Sprintf(format, args...)))
}

func (l *zerologgerLogger) Warningf(format string, args ...interface{}) {
	log.Warn().Msg(fmt.Sprintf("%s: %s", l.name, fmt.Sprintf(format, args...)))
}

func (l *zerologgerLogger) Errorf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log.Err(errors.New(msg)).Str("source", l.name).Msg(msg)
}

func (l *zerologgerLogger) Debugf(format string, args ...interface{}) {
	log.Debug().Msg(fmt.Sprintf("%s: %s", l.name, fmt.Sprintf(format, args...)))
}
func (l *zerologgerLogger) Panicf(format string, args ...interface{}) {
	log.Panic().Msg(fmt.Sprintf("%s: %s", l.name, fmt.Sprintf(format, args...)))
}

func (l *zerologgerLogger) SetLevel(logLevel dblog.LogLevel) {
	//using zerologger level
}

// LoggerFactory que crea fmtLogger
type fmtLoggerFactory struct{}

func CreateZerologger(name string) dblog.ILogger {
	return &zerologgerLogger{name: name}
}
