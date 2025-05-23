package gql

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"unicode/utf8"

	assertions "github.com/stretchr/testify/assert"
)

// TestUnicodeDebugging creates specific scenarios that might cause "trailing garbage" errors
func TestUnicodeDebugging(t *testing.T) {
	assert := assertions.New(t)

	t.Run("should debug exact scenario from user report", func(t *testing.T) {
		// Recreate the exact scenario from the user's log
		client := CreateTestClient()

		// Exact data from the user's report
		variables := map[string]interface{}{
			"group_description": "This a group for serious buyers and sellers\nOf uber eats and Door dash account \nRippers are not allowed in this group😈",
			"user_name":         "2eba巴结",
		}

		query := "mutation storeMessage($group_description: String!, $user_name: String!) { insert_message(group_description: $group_description, user_name: $user_name) { id } }"

		// Compile the query
		compiledQuery := client.compileQuery(query, variables)
		assert.NotNil(compiledQuery)

		jsonBytes := compiledQuery.JsonQuery
		t.Logf("JSON size: %d bytes", len(jsonBytes))
		t.Logf("JSON content: %s", string(jsonBytes))

		// Check for potential issues
		t.Logf("Is valid UTF-8: %v", utf8.Valid(jsonBytes))
		t.Logf("Is valid JSON: %v", json.Valid(jsonBytes))

		// Check for any trailing bytes or null terminators
		if len(jsonBytes) > 0 {
			t.Logf("Last byte: 0x%02x (%c)", jsonBytes[len(jsonBytes)-1], jsonBytes[len(jsonBytes)-1])
			if len(jsonBytes) > 1 {
				t.Logf("Second to last byte: 0x%02x (%c)", jsonBytes[len(jsonBytes)-2], jsonBytes[len(jsonBytes)-2])
			}
		}

		// Check for any BOM (Byte Order Mark)
		if len(jsonBytes) >= 3 && jsonBytes[0] == 0xEF && jsonBytes[1] == 0xBB && jsonBytes[2] == 0xBF {
			t.Logf("WARNING: UTF-8 BOM detected at start of JSON")
		}

		// Try to parse with different JSON parsers to see if there are differences
		var parsed1 map[string]interface{}
		err1 := json.Unmarshal(jsonBytes, &parsed1)
		t.Logf("Standard json.Unmarshal error: %v", err1)

		// Check if there are any non-printable characters at the end
		for i := len(jsonBytes) - 1; i >= 0 && i >= len(jsonBytes)-10; i-- {
			b := jsonBytes[i]
			if b < 32 && b != '\n' && b != '\r' && b != '\t' {
				t.Logf("Non-printable character at position %d: 0x%02x", i, b)
			}
		}

		// Simulate the exact request size mentioned in the user's log (1093 bytes)
		t.Logf("Expected size from user log: 1093 bytes, actual size: %d bytes", len(jsonBytes))
		if len(jsonBytes) != 1093 {
			t.Logf("Size mismatch - this might indicate a different encoding or additional data")
		}
	})

	t.Run("should test potential causes of trailing garbage", func(t *testing.T) {
		client := CreateTestClient()

		// Test scenarios that might cause trailing garbage
		testCases := []struct {
			name        string
			variables   map[string]interface{}
			expectIssue bool
		}{
			{
				name: "double_encoding_scenario",
				variables: map[string]interface{}{
					"json_string": `{"nested": "json with emoji 😈"}`,
				},
				expectIssue: false,
			},
			{
				name: "escaped_unicode_scenario",
				variables: map[string]interface{}{
					"escaped": "\\u1F608", // Escaped Unicode
				},
				expectIssue: false,
			},
			{
				name: "mixed_encoding_scenario",
				variables: map[string]interface{}{
					"mixed": "ASCII text with 😈 emoji and 巴结 Chinese",
				},
				expectIssue: false,
			},
			{
				name: "large_unicode_scenario",
				variables: map[string]interface{}{
					"large": "This a group for serious buyers and sellers\nOf uber eats and Door dash account \nRippers are not allowed in this group😈",
					"user":  "2eba巴结",
					"extra": "Additional data to increase size",
				},
				expectIssue: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				query := "mutation test($json_string: String, $escaped: String, $mixed: String, $large: String, $user: String, $extra: String) { test }"
				compiledQuery := client.compileQuery(query, tc.variables)

				assert.NotNil(compiledQuery)
				jsonBytes := compiledQuery.JsonQuery

				t.Logf("%s - JSON size: %d bytes", tc.name, len(jsonBytes))
				t.Logf("%s - Is valid UTF-8: %v", tc.name, utf8.Valid(jsonBytes))
				t.Logf("%s - Is valid JSON: %v", tc.name, json.Valid(jsonBytes))

				// Try to unmarshal
				var parsed map[string]interface{}
				err := json.Unmarshal(jsonBytes, &parsed)
				if tc.expectIssue {
					assert.Error(err, "Expected error for %s", tc.name)
				} else {
					assert.NoError(err, "Should not error for %s", tc.name)
				}
			})
		}
	})

	t.Run("should test server response to problematic requests", func(t *testing.T) {
		// Create a server that's very strict about JSON parsing
		strictServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body := make([]byte, r.ContentLength)
			n, _ := r.Body.Read(body)
			actualBody := body[:n]

			t.Logf("Server received %d bytes", len(actualBody))
			t.Logf("Server received content: %s", string(actualBody))

			// Try to parse with strict JSON decoder
			decoder := json.NewDecoder(bytes.NewReader(actualBody))
			decoder.DisallowUnknownFields()

			var request map[string]interface{}
			err := decoder.Decode(&request)
			if err != nil {
				t.Logf("Server JSON decode error: %s", err.Error())
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(fmt.Sprintf(`{"error":"trailing garbage after JSON","code":"invalid-json","details":"%s"}`, err.Error())))
				return
			}

			// Check if there's more data after the JSON
			var extra interface{}
			err = decoder.Decode(&extra)
			if err == nil {
				t.Logf("Server found trailing data after JSON")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":"trailing garbage after JSON","code":"invalid-json"}`))
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data":{"result":"success"}}`))
		}))
		defer strictServer.Close()

		client := CreateTestClient()

		variables := map[string]interface{}{
			"group_description": "This a group for serious buyers and sellers\nOf uber eats and Door dash account \nRippers are not allowed in this group😈",
			"user_name":         "2eba巴结",
		}

		query := "mutation storeMessage($group_description: String!, $user_name: String!) { insert_message(group_description: $group_description, user_name: $user_name) { id } }"
		compiledQuery := client.compileQuery(query, variables)

		qe := &QueryExecutor{
			BaseClient: client,
			Query:      compiledQuery.JsonQuery,
			Headers:    map[string]interface{}{"Content-Type": "application/json; charset=utf-8"},
			CacheKey:   "no-cache",
			Retries:    false,
		}
		qe.endpoint = strictServer.URL
		qe.client = strictServer.Client()
		qe.cache = client.cache

		result, err := qe.executeQuery()
		if err != nil {
			t.Logf("Error from strict server: %s", err.Error())
		} else {
			t.Logf("Strict server accepted the request successfully")
		}

		assert.NoError(err, "Strict server should accept properly encoded JSON")
		assert.NotNil(result)
	})

	t.Run("should analyze JSON encoder behavior", func(t *testing.T) {
		client := CreateTestClient()

		// Test the exact data structure
		query := &Query{
			Query: "mutation storeMessage($group_description: String!, $user_name: String!) { insert_message(group_description: $group_description, user_name: $user_name) { id } }",
			Variables: map[string]interface{}{
				"group_description": "This a group for serious buyers and sellers\nOf uber eats and Door dash account \nRippers are not allowed in this group😈",
				"user_name":         "2eba巴结",
			},
		}

		// Test different encoding methods
		t.Run("using_json_Marshal", func(t *testing.T) {
			jsonBytes, err := json.Marshal(query)
			assert.NoError(err)
			t.Logf("json.Marshal size: %d bytes", len(jsonBytes))
			t.Logf("json.Marshal valid: %v", json.Valid(jsonBytes))
		})

		t.Run("using_convertToJSON", func(t *testing.T) {
			jsonBytes := client.convertToJSON(query)
			assert.NotNil(jsonBytes)
			t.Logf("convertToJSON size: %d bytes", len(jsonBytes))
			t.Logf("convertToJSON valid: %v", json.Valid(jsonBytes))
		})

		t.Run("using_json_Encoder", func(t *testing.T) {
			var buf bytes.Buffer
			encoder := json.NewEncoder(&buf)
			encoder.SetEscapeHTML(false)
			err := encoder.Encode(query)
			assert.NoError(err)

			jsonBytes := buf.Bytes()
			// Remove trailing newline that Encoder adds
			if len(jsonBytes) > 0 && jsonBytes[len(jsonBytes)-1] == '\n' {
				jsonBytes = jsonBytes[:len(jsonBytes)-1]
			}

			t.Logf("json.Encoder size: %d bytes", len(jsonBytes))
			t.Logf("json.Encoder valid: %v", json.Valid(jsonBytes))
		})
	})
}

// TestUnicodeEncodingComparison compares different encoding approaches
func TestUnicodeEncodingComparison(t *testing.T) {
	assert := assertions.New(t)

	// Test data that caused the original issue
	testData := map[string]interface{}{
		"group_description": "This a group for serious buyers and sellers\nOf uber eats and Door dash account \nRippers are not allowed in this group😈",
		"user_name":         "2eba巴结",
	}

	t.Run("compare_encoding_methods", func(t *testing.T) {
		// Method 1: Standard json.Marshal
		bytes1, err1 := json.Marshal(testData)
		assert.NoError(err1)

		// Method 2: json.Encoder with SetEscapeHTML(false)
		var buf2 bytes.Buffer
		enc2 := json.NewEncoder(&buf2)
		enc2.SetEscapeHTML(false)
		err2 := enc2.Encode(testData)
		assert.NoError(err2)
		bytes2 := buf2.Bytes()
		if len(bytes2) > 0 && bytes2[len(bytes2)-1] == '\n' {
			bytes2 = bytes2[:len(bytes2)-1]
		}

		// Method 3: json.Encoder with SetEscapeHTML(true) - default
		var buf3 bytes.Buffer
		enc3 := json.NewEncoder(&buf3)
		enc3.SetEscapeHTML(true)
		err3 := enc3.Encode(testData)
		assert.NoError(err3)
		bytes3 := buf3.Bytes()
		if len(bytes3) > 0 && bytes3[len(bytes3)-1] == '\n' {
			bytes3 = bytes3[:len(bytes3)-1]
		}

		t.Logf("Method 1 (json.Marshal) size: %d", len(bytes1))
		t.Logf("Method 2 (Encoder, EscapeHTML=false) size: %d", len(bytes2))
		t.Logf("Method 3 (Encoder, EscapeHTML=true) size: %d", len(bytes3))

		t.Logf("Method 1 content: %s", string(bytes1))
		t.Logf("Method 2 content: %s", string(bytes2))
		t.Logf("Method 3 content: %s", string(bytes3))

		// All should be valid
		assert.True(json.Valid(bytes1))
		assert.True(json.Valid(bytes2))
		assert.True(json.Valid(bytes3))

		// Check for differences
		if !bytes.Equal(bytes1, bytes2) {
			t.Logf("Difference between Marshal and Encoder(EscapeHTML=false)")
		}
		if !bytes.Equal(bytes2, bytes3) {
			t.Logf("Difference between EscapeHTML=false and EscapeHTML=true")
		}
	})
}
