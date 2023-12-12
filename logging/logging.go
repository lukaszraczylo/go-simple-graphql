package libpack_logging

import (
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gookit/goutil/envutil"
	"github.com/rs/zerolog"
)

type LogConfig struct {
	baseLogger  zerolog.Logger
	errorLogger zerolog.Logger
	nopLogger   zerolog.Logger
	mu          sync.Mutex
	minLevel    zerolog.Level
}

func init() {
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.MessageFieldName = "msg"
	zerolog.TimestampFieldName = "ts"
	zerolog.LevelFieldName = "level"
	zerolog.LevelFatalValue = "critical"
}

func getMinLogLevel() zerolog.Level {
	levelStr := strings.ToLower(envutil.Getenv("LOG_LEVEL", "info"))
	return matchLogLevel(levelStr)
}

// Add function set the minimum log level instead of using environment variable
func (lc *LogConfig) SetLogLevel(level string) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.minLevel = matchLogLevel(level)
}

// Match the log level provided as string with the log level provided as zerolog.Level
func matchLogLevel(level string) zerolog.Level {
	switch level {
	case "debug":
		return zerolog.DebugLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "critical":
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}

func NewLogger() *LogConfig {
	zl := zerolog.New(os.Stdout).With().Timestamp().Logger()
	// Determine the minimum log level from environment variable
	return &LogConfig{
		baseLogger:  zl,
		errorLogger: zl.Output(os.Stderr),
		nopLogger:   zerolog.Nop(),
		minLevel:    getMinLogLevel(),
	}
}

func (lc *LogConfig) getLogger(level zerolog.Level) zerolog.Logger {
	if level >= zerolog.ErrorLevel {
		return lc.errorLogger
	}
	if level < lc.minLevel {
		return lc.nopLogger
	}
	return lc.baseLogger
}

func (lc *LogConfig) Log(level zerolog.Level, message string, fields []map[string]interface{}) {
	if level < lc.minLevel {
		return
	}
	logger := lc.getLogger(level)
	event := logger.WithLevel(level).CallerSkipFrame(3)
	if len(fields) == 0 {
		event.Msg(message)
		return
	}

	field := fields[0]
	for k, val := range field {
		switch v := val.(type) {
		case string:
			event.Str(k, v)
		case int:
			event.Int(k, v)
		case float64:
			event.Float64(k, v)
		default:
			event.Interface(k, v)
		}
	}
	event.Msg(message)
}

func (lc *LogConfig) Info(message string, fields ...map[string]interface{}) {
	lc.Log(zerolog.InfoLevel, message, fields)
}

func (lc *LogConfig) Debug(message string, fields ...map[string]interface{}) {
	lc.Log(zerolog.DebugLevel, message, fields)
}

func (lc *LogConfig) Warn(message string, fields ...map[string]interface{}) {
	lc.Log(zerolog.WarnLevel, message, fields)
}

// alias Warning to Warn
func (lc *LogConfig) Warning(message string, fields ...map[string]interface{}) {
	lc.Warn(message, fields...)
}

func (lc *LogConfig) Error(message string, fields ...map[string]interface{}) {
	lc.Log(zerolog.ErrorLevel, message, fields)
}

func (lc *LogConfig) Critical(message string, fields ...map[string]interface{}) {
	lc.Log(zerolog.FatalLevel, message, fields)
	os.Exit(1)
}

// alias Fatal to Critical
func (lc *LogConfig) Fatal(message string, fields ...map[string]interface{}) {
	lc.Critical(message, fields...)
}
