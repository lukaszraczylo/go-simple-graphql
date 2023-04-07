package logger

import (
	"reflect"

	"github.com/lukaszraczylo/go-simple-graphql/utils/helpers"
	"github.com/lukaszraczylo/go-simple-graphql/utils/ordered_map"

	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

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

func Serialize(v any) (string, error) {
	marshal, err := json.Marshal(internalSerialize(v))
	if err != nil {
		return "", err
	}
	return helpers.BytesToString(marshal), nil
}

func internalSerialize(v any) any {
	if v != nil {
		switch reflect.TypeOf(v).Kind() {
		case reflect.Struct:
			r := ordered_map.New()
			name := reflect.TypeOf(v).Name()
			if name != "" {
				r.Set("_", reflect.TypeOf(v).String())
			}
			for i := 0; i < reflect.ValueOf(v).NumField(); i++ {
				r.Set(reflect.TypeOf(v).Field(i).Name, internalSerialize(reflect.ValueOf(v).Field(i).Interface()))
			}
			return r
		case reflect.Ptr:
			if !reflect.ValueOf(v).IsNil() {
				return internalSerialize(reflect.ValueOf(v).Elem().Interface())
			}
		case reflect.Slice:
			var r []any
			tmpSlice := reflect.ValueOf(v)
			for i := 0; i < tmpSlice.Len(); i++ {
				r = append(r, internalSerialize(tmpSlice.Index(i).Interface()))
			}
			return r
		}
	}
	return v
}
