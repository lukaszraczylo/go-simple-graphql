package gql

import (
	"reflect"
	"testing"
)

func (suite *Tests) Test_searchForKeysInMapStringInterface() {
	type args struct {
		msi map[string]interface{}
		key string
	}
	tests := []struct {
		wantValue any
		args      args
		name      string
	}{
		{
			name: "Test_searchForKeysInMapStringInterface_exists",
			args: args{
				msi: map[string]interface{}{
					"gqlcache": true,
					"test":     "test2",
				},
				key: "test",
			},
			wantValue: "test2",
		},
		{
			name: "Test_searchForKeysInMapStringInterface_not_exists",
			args: args{
				msi: map[string]interface{}{
					"gqlcache": true,
					"test":     "test2",
				},
				key: "test-new",
			},
			wantValue: nil,
		},
	}
	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			assert.Equal(tt.wantValue, searchForKeysInMapStringInterface(tt.args.msi, tt.args.key))
		})
	}
}

func (suite *Tests) TestBaseClient_decodeResponse() {
	type args struct {
		response []byte
	}
	tests := []struct {
		name     string
		args     args
		want     any
		wantErr  bool
		setType  string
		wantType string
	}{
		{
			name: "TestBaseClient_decodeResponse_mapstring",
			args: args{
				response: []byte(`{"data":{"viewer":{"login":"lukaszraczylo"}}}`),
			},
			want: map[string]interface{}{
				"data": map[string]interface{}{
					"viewer": map[string]interface{}{
						"login": "lukaszraczylo",
					},
				},
			},
			setType:  "mapstring",
			wantType: "map[string]interface {}",
		},
		{
			name: "TestBaseClient_decodeResponse_string",
			args: args{
				response: []byte(`{"data":{"viewer":{"login":"lukaszraczylo"}}}`),
			},
			want:     `{"data":{"viewer":{"login":"lukaszraczylo"}}}`,
			setType:  "string",
			wantType: "string",
		},
		{
			name: "TestBaseClient_decodeResponse_byte",
			args: args{
				response: []byte(`{"data":{"viewer":{"login":"lukaszraczylo"}}}`),
			},
			want:     []byte(`{"data":{"viewer":{"login":"lukaszraczylo"}}}`),
			setType:  "byte",
			wantType: "[]uint8",
		},
	}
	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			b := NewConnection()
			b.SetOutput(tt.setType)
			got, err := b.decodeResponse(tt.args.response)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(tt.wantType, reflect.TypeOf(got).String())
		})
	}
}
