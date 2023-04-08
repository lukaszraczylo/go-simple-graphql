package logger

import (
	"encoding/json"
	"reflect"
)

const (
	reset      = "\033[0m"
	redBold    = "\033[1;31m"
	greenBold  = "\033[1;32m"
	yellowBold = "\033[1;33m"
	blueBold   = "\033[1;34m"
)

type LogLevel int

const (
	Silent LogLevel = iota
	Error
	Warn
	Info
	Debug
)

type Config struct {
	Colorful bool
	LogLevel LogLevel
}

func Serialize(v interface{}) ([]byte, error) {
	b, err := json.Marshal(internalSerialize(v))
	if err != nil {
		return nil, err
	}
	return b, nil
}

func internalSerialize(v interface{}) interface{} {
	if v != nil {
		switch reflect.TypeOf(v).Kind() {
		case reflect.Struct:
			r := make(map[string]interface{})
			name := reflect.TypeOf(v).Name()
			if name != "" {
				r["_"] = reflect.TypeOf(v).String()
			}
			for i := 0; i < reflect.ValueOf(v).NumField(); i++ {
				r[reflect.TypeOf(v).Field(i).Name] = internalSerialize(reflect.ValueOf(v).Field(i).Interface())
			}
			return r
		case reflect.Ptr:
			if !reflect.ValueOf(v).IsNil() {
				return internalSerialize(reflect.ValueOf(v).Elem().Interface())
			}
		case reflect.Slice:
			var r []interface{}
			tmpSlice := reflect.ValueOf(v)
			for i := 0; i < tmpSlice.Len(); i++ {
				r = append(r, internalSerialize(tmpSlice.Index(i).Interface()))
			}
			return r
		}
	}
	return v
}
