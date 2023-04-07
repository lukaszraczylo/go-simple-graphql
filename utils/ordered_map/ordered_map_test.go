package ordered_map

import (
	"reflect"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		want *OrderedMap
		name string
	}{
		{
			name: "TestNew",
			want: &OrderedMap{
				keys:   []string{},
				values: map[string]interface{}{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrderedMap_Set(t *testing.T) {
	type fields struct {
		values map[string]interface{}
		keys   []string
	}
	type args struct {
		value interface{}
		key   string
	}
	tests := []struct {
		args   args
		name   string
		fields fields
	}{
		{
			name: "TestOrderedMap_Set",
			fields: fields{
				values: map[string]interface{}{},
				keys:   []string{},
			},
			args: args{},
		},
		{
			name: "TestOrderedMap_Set2",
			fields: fields{
				values: map[string]interface{}{
					"hello": "world",
				},
				keys: []string{"hello"},
			},
			args: args{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &OrderedMap{
				values: tt.fields.values,
				keys:   tt.fields.keys,
			}
			o.Set(tt.args.key, tt.args.value)
			for _, key := range o.keys {
				if _, ok := o.values[key]; !ok {
					t.Errorf("OrderedMap.Set() key not found in values map")
				}
			}
		})
	}
}

func TestOrderedMap_MarshalJSON(t *testing.T) {
	type fields struct {
		values map[string]interface{}
		keys   []string
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr bool
	}{
		{
			name: "TestOrderedMap_MarshalJSON",
			fields: fields{
				values: map[string]interface{}{
					"hello": "world",
				},
				keys: []string{"hello"},
			},
			want:    []byte(`{"hello":"world"}`),
			wantErr: false,
		},
		{
			name: "TestOrderedMap_MarshalJSON2",
			fields: fields{
				values: map[string]interface{}{
					"hello": "world",
					"foo":   "bar",
				},
				keys: []string{"hello", "foo"},
			},
			want:    []byte(`{"hello":"world","foo":"bar"}`),
			wantErr: false,
		},
		{
			name: "TestOrderedMap_MarshalJSON_with_error",
			fields: fields{
				values: map[string]interface{}{
					"hello": "world",
					"foo":   make(chan int),
				},
				keys: []string{"hello", "foo"},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := OrderedMap{
				values: tt.fields.values,
				keys:   tt.fields.keys,
			}
			got, err := o.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("OrderedMap.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OrderedMap.MarshalJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkOrderedMapTest(b *testing.B) {
	m := New()
	for i := 0; i < b.N; i++ {
		m.Set("hello", "world")
	}
}

func BenchmarkOrderedMarshalJSON(b *testing.B) {
	m := New()
	for i := 0; i < b.N; i++ {
		m.Set("hello", "world")
		m.MarshalJSON()
	}
}

func BenchmarkMapTest(b *testing.B) {
	m := make(map[string]interface{})
	for i := 0; i < b.N; i++ {
		m["hello"] = "world"
	}
}
