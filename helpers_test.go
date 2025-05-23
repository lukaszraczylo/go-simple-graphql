package gql

import (
	"testing"
	"time"

	"github.com/goccy/go-reflect"
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
		want     any
		name     string
		setType  string
		wantType string
		args     args
		wantErr  bool
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

func (suite *Tests) Test_searchForKeysInMapStringInterface_nil() {
	suite.T().Run("should handle nil map", func(t *testing.T) {
		result := searchForKeysInMapStringInterface(nil, "test")
		assert.Nil(result)
	})
}

func (suite *Tests) Test_calculateHash() {
	suite.T().Run("should calculate consistent hash", func(t *testing.T) {
		query1 := &Query{JsonQuery: []byte(`{"query": "{ user { name } }"}`)}
		query2 := &Query{JsonQuery: []byte(`{"query": "{ user { name } }"}`)}
		query3 := &Query{JsonQuery: []byte(`{"query": "{ user { email } }"}`)}

		hash1 := calculateHash(query1)
		hash2 := calculateHash(query2)
		hash3 := calculateHash(query3)

		// Same queries should produce same hash
		assert.Equal(hash1, hash2)
		// Different queries should produce different hashes
		assert.NotEqual(hash1, hash3)
		// Hash should be non-empty
		assert.NotEmpty(hash1)
	})
}

func (suite *Tests) TestBaseClient_cacheLookup() {
	suite.T().Run("should lookup cache entries", func(t *testing.T) {
		client := CreateTestClient()

		// Test cache miss
		result := client.cacheLookup("nonexistent")
		assert.Nil(result)

		// Test cache hit
		testData := []byte("test data")
		client.cache.Set("test_key", testData, 5*time.Second)
		result = client.cacheLookup("test_key")
		assert.Equal(testData, result)
	})
}

func (suite *Tests) TestBaseClient_decodeResponse_errors() {
	suite.T().Run("should handle invalid JSON for mapstring", func(t *testing.T) {
		client := NewConnection()
		client.SetOutput("mapstring")

		_, err := client.decodeResponse([]byte(`invalid json`))
		assert.Error(err)
	})

	suite.T().Run("should handle unknown response type", func(t *testing.T) {
		client := NewConnection()
		client.responseType = "unknown"

		_, err := client.decodeResponse([]byte(`{"test": "data"}`))
		assert.Error(err)
		assert.Contains(err.Error(), "unknown response type")
	})
}
