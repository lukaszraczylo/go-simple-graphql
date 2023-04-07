package ordered_map

import (
	"bytes"

	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type OrderedMap struct {
	values map[string]interface{}
	keys   []string
}

func New() *OrderedMap {
	o := OrderedMap{}
	o.keys = []string{}
	o.values = map[string]interface{}{}
	return &o
}

func (o *OrderedMap) Set(key string, value interface{}) {
	_, exists := o.values[key]
	if !exists {
		o.keys = append(o.keys, key)
	}
	o.values[key] = value
}

func (o OrderedMap) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "")
	for i, k := range o.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.WriteByte('"')
		buf.WriteString(k)
		buf.WriteString(`":`)
		customEncoder := &customJsonEncoder{buf: &buf}
		if err := customEncoder.Encode(o.values[k]); err != nil {
			return nil, err
		}
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

type customJsonEncoder struct {
	buf *bytes.Buffer
}

func (e *customJsonEncoder) Encode(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	e.buf.Write(data)
	return nil
}
