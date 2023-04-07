package gql

import (
	"fmt"
	"reflect"
	"time"

	"github.com/lukaszraczylo/go-simple-graphql/utils/logger"

	"github.com/gookit/goutil/timex"
)

func NewLogger(writer Writer, config logger.Config) Logger {
	colorfulPrint := "%s[%s]%s %%s %%s"
	closeColor, debugColor, infoColor, warnColor, errorColor := "\033[0m", "\033[1;34m", "\033[1;32m", "\033[1;33m", "\033[1;31m"
	if !config.Colorful {
		closeColor, debugColor, infoColor, warnColor, errorColor = "", "", "", "", ""
	}
	debugStr := fmt.Sprintf(colorfulPrint, debugColor, "DEBUG", closeColor)
	infoStr := fmt.Sprintf(colorfulPrint, infoColor, "INFO", closeColor)
	warnStr := fmt.Sprintf(colorfulPrint, warnColor, "WARN", closeColor)
	errStr := fmt.Sprintf(colorfulPrint, errorColor, "ERROR", closeColor)

	return &logging{
		Writer:   writer,
		Config:   config,
		infoStr:  infoStr,
		warnStr:  warnStr,
		errStr:   errStr,
		debugStr: debugStr,
	}
}

func setDate() string {
	tx := time.Now()
	return timex.DateFormat(tx, "Y-m-d H:I:S")
}

func (l logging) Debug(client *BaseClient, msg string, data ...any) {
	if l.LogLevel >= logger.Debug {
		t := setDate()
		msg, data = censorSensitiveInfo(client, msg, data...)
		if client != nil {
			l.Printf(l.debugStr, fmt.Sprintf("[%s] %s", t, msg), sprint(data...))
		} else {
			l.Printf(l.debugStr, msg, sprint(data...))
		}
	}
}

func (l logging) Info(client *BaseClient, msg string, data ...any) {
	if l.LogLevel >= logger.Info {
		t := setDate()
		msg, data = censorSensitiveInfo(client, msg, data...)
		if client != nil {
			l.Printf(l.infoStr, fmt.Sprintf("[%s] %s", t, msg), sprint(data...))
		} else {
			l.Printf(l.infoStr, msg, sprint(data...))
		}
	}
}

func (l logging) Warn(client *BaseClient, msg string, data ...any) {
	if l.LogLevel >= logger.Warn {
		t := setDate()
		msg, data = censorSensitiveInfo(client, msg, data...)
		if client != nil {
			l.Printf(l.warnStr, fmt.Sprintf("[%s] %s", t, msg), sprint(data...))
		} else {
			l.Printf(l.warnStr, msg, sprint(data...))
		}
	}
}

func (l logging) Error(client *BaseClient, msg string, data ...any) {
	if l.LogLevel >= logger.Error {
		t := setDate()
		msg, data = censorSensitiveInfo(client, msg, data...)
		if client != nil {
			l.Printf(l.errStr, fmt.Sprintf("[%s] %s", t, msg), sprint(data...))
		} else {
			l.Printf(l.errStr, msg, sprint(data...))
		}
	}
}

type Logger interface {
	Debug(*BaseClient, string, ...any)
	Info(*BaseClient, string, ...any)
	Warn(*BaseClient, string, ...any)
	Error(*BaseClient, string, ...any)
}

type Writer interface {
	Printf(string, ...any)
}

type logging struct {
	Writer
	debugStr string
	infoStr  string
	warnStr  string
	errStr   string
	logger.Config
}

func censorSensitiveInfo(client *BaseClient, a string, b ...any) (string, []any) {
	// if client == nil || client.Token == "" {
	// 	return a, b
	// }
	// tokenFixed := strings.Split(client.Token, ":")[1]
	// for t := range b {
	// 	if b[t] != nil {
	// 		rV := reflect.TypeOf(b[t]).Kind()
	// 		if rV == reflect.String {
	// 			b[t] = strings.ReplaceAll(b[t].(string), tokenFixed, strings.Repeat("X", len(tokenFixed)))
	// 		}
	// 	}
	// }
	// a = strings.ReplaceAll(a, tokenFixed, strings.Repeat("X", len(tokenFixed)))
	// return a, b
	return a, b
}

func sprint(a ...any) string {
	for t := range a {
		if a[t] != nil {
			rV := reflect.TypeOf(a[t]).Kind()
			if rV == reflect.Ptr {
				a[t] = reflect.ValueOf(a[t]).Elem().Interface()
			}
			rV = reflect.TypeOf(a[t]).Kind()
			if rV == reflect.Struct || rV == reflect.Map || rV == reflect.Slice {
				a[t], _ = logger.Serialize(a[t])
			}
		}
	}
	r := fmt.Sprintln(a...)
	r = r[:len(r)-1]
	return r
}
