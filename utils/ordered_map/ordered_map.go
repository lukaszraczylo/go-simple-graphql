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
	if v == nil {
		e.buf.WriteString("null")
		return nil
	}

	switch val := v.(type) {
	case *OrderedMap:
		data, err := json.Marshal(val.values)
		if err != nil {
			return err
		}
		e.buf.Write(data)
	default:
		data, err := json.Marshal(val)
		if err != nil {
			return err
		}
		e.buf.Write(data)
	}
	return nil
}

func (o OrderedMap) Keys() []string {
	return o.keys
}

func (o OrderedMap) Values() []interface{} {
	var values []interface{}
	for _, k := range o.keys {
		values = append(values, o.values[k])
	}
	return values
}

func (o *OrderedMap) Delete(key string) {
	delete(o.values, key)
	for i, k := range o.keys {
		if k == key {
			o.keys = append(o.keys[:i], o.keys[i+1:]...)
			break
		}
	}
}

func (o OrderedMap) Exists(key string) bool {
	_, exists := o.values[key]
	return exists
}

func (o *OrderedMap) UnmarshalJSON(data []byte) error {
	var values map[string]interface{}
	err := json.Unmarshal(data, &values)
	if err != nil {
		return err
	}

	o.keys = []string{}
	o.values = map[string]interface{}{}
	for k, v := range values {
		o.Set(k, v)
	}

	return nil
}
