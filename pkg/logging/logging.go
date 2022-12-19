package logging

import (
	"fmt"
	stdlog "log"
	"os"
	"time"

	"github.com/rs/zerolog"
)

type LogConfig struct {
	DefaultOutput *os.File
	Logger        zerolog.Logger
}

func NewLogger() *LogConfig {
	log := new(LogConfig)
	dll := os.Getenv("LOG_LEVEL")
	if dll == "debug" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else if dll == "warn" {
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	} else if dll == "error" {
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.MessageFieldName = "short_message"
	zerolog.TimestampFieldName = "timestamp"
	zerolog.LevelFatalValue = "critical"
	log.Logger = log.Logger.With().Timestamp().Logger()
	stdlog.SetFlags(0)
	stdlog.SetOutput(log.Logger)
	return log
}

func (l *LogConfig) Log(level zerolog.Level, message string, v ...map[string]interface{}) {
	contextLog := l.Logger.WithLevel(level)
	if len(v) > 0 {
		for _, m := range v {
			for k, v := range m {
				if v != nil && k != "" {
					contextLog.Str(k, fmt.Sprintf("%v", v))
				}
			}
		}
	}
	// contextLog.Str("_stack_trace", pandati.Trace())
	contextLog.Msg(message)
}

func (l *LogConfig) Critical(message string, v ...map[string]interface{}) {
	l.Log(zerolog.FatalLevel, message, v...)
}

func (l *LogConfig) Error(message string, v ...map[string]interface{}) {
	l.Log(zerolog.ErrorLevel, message, v...)
}

func (l *LogConfig) Warning(message string, v ...map[string]interface{}) {
	l.Log(zerolog.WarnLevel, message, v...)
}

func (l *LogConfig) Info(message string, v ...map[string]interface{}) {
	l.Log(zerolog.InfoLevel, message, v...)
}

func (l *LogConfig) Debug(message string, v ...map[string]interface{}) {
	l.Log(zerolog.DebugLevel, message, v...)
}
