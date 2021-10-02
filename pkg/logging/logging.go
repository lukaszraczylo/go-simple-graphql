package logging

import (
	stdlog "log"
	"os"
	"time"

	"github.com/lukaszraczylo/pandati"
	"github.com/rs/zerolog"
)

type LogConfig struct {
	DefaultOutput *os.File
	Logger        zerolog.Logger
}

func NewLogger() *LogConfig {
	log := new(LogConfig)
	switch dll := os.Getenv("LOG_LEVEL"); dll {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
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

func (l *LogConfig) Critical(message string, v ...map[string]interface{}) {
	w := l.Logger.Output(os.Stderr)
	log(w.Fatal(), message, v...)
}

func (l *LogConfig) Error(message string, v ...map[string]interface{}) {
	w := l.Logger.Output(os.Stderr)
	log(w.Error(), message, v...)
}

func (l *LogConfig) Warning(message string, v ...map[string]interface{}) {
	w := l.Logger.Output(os.Stdout)
	log(w.Warn(), message, v...)
}

func (l *LogConfig) Info(message string, v ...map[string]interface{}) {
	w := l.Logger.Output(os.Stdout)
	log(w.Info(), message, v...)
}

func (l *LogConfig) Debug(message string, v ...map[string]interface{}) {
	w := l.Logger.Output(os.Stdout)
	log(w.Debug(), message, v...)
}

func log(contextLog *zerolog.Event, message string, v ...map[string]interface{}) {
	if len(v) > 0 {
		for _, m := range v {
			for k, v := range m {
				if !pandati.IsZero(v) {
					contextLog.Str(k, v.(string))
				}
			}
		}
	}
	// contextLog.Str("_stack_trace", pandati.Trace())
	contextLog.Msg(message)
}
