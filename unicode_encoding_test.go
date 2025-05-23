package gql

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	assertions "github.com/stretchr/testify/assert"
)

// TestUnicodeEncodingIssue reproduces the "trailing garbage" error with emojis and special characters
func TestUnicodeEncodingIssue(t *testing.T) {
	assert := assertions.New(t)

	t.Run("should handle emoji and Chinese characters in GraphQL variables", func(t *testing.T) {
		// Create a test server that simulates the problematic scenario
		var receivedBody []byte
		var receivedContentType string
		var receivedContentLength int64

		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Capture the request details for analysis
			receivedContentType = r.Header.Get("Content-Type")
			receivedContentLength = r.ContentLength

			// Read the request body to analyze the encoding
			body := make([]byte, r.ContentLength)
			r.Body.Read(body)
			receivedBody = body

			// Log the received data for debugging
			t.Logf("Received Content-Type: %s", receivedContentType)
			t.Logf("Received Content-Length: %d", receivedContentLength)
			t.Logf("Received Body (first 200 chars): %s", string(body[:min(200, len(body))]))
			t.Logf("Received Body (hex): %x", body[:min(50, len(body))])

			// Check if the JSON is valid
			var parsedRequest map[string]interface{}
			err := json.Unmarshal(body, &parsedRequest)
			if err != nil {
				t.Logf("JSON parsing error: %s", err.Error())
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":"trailing garbage after JSON","code":"invalid-json"}`))
				return
			}

			// Successful response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data":{"result":"success"}}`))
		}))
		defer testServer.Close()

		client := CreateTestClient()

		// Test data that reproduces the issue
		variables := map[string]interface{}{
			"group_description": "This a group for serious buyers and sellers\nOf uber eats and Door dash account \nRippers are not allowed in this group😈",
			"user_name":         "2eba巴结",
		}

		query := "mutation storeMessage($group_description: String!, $user_name: String!) { insert_message(group_description: $group_description, user_name: $user_name) { id } }"

		// Compile the query to see how it's encoded
		compiledQuery := client.compileQuery(query, variables)
		assert.NotNil(compiledQuery)
		assert.NotNil(compiledQuery.JsonQuery)

		t.Logf("Compiled Query JSON: %s", string(compiledQuery.JsonQuery))
		t.Logf("Compiled Query JSON (hex): %x", compiledQuery.JsonQuery)

		// Check if the JSON is valid UTF-8
		assert.True(json.Valid(compiledQuery.JsonQuery), "Compiled JSON should be valid")

		// Verify the JSON contains the Unicode characters correctly
		var parsedQuery map[string]interface{}
		err := json.Unmarshal(compiledQuery.JsonQuery, &parsedQuery)
		assert.NoError(err)

		variables_parsed := parsedQuery["variables"].(map[string]interface{})
		assert.Equal("This a group for serious buyers and sellers\nOf uber eats and Door dash account \nRippers are not allowed in this group😈", variables_parsed["group_description"])
		assert.Equal("2eba巴结", variables_parsed["user_name"])

		// Execute the query
		qe := &QueryExecutor{
			BaseClient: client,
			Query:      compiledQuery.JsonQuery,
			Headers:    map[string]interface{}{"Content-Type": "application/json; charset=utf-8"},
			CacheKey:   "no-cache",
			Retries:    false,
		}
		qe.endpoint = testServer.URL
		qe.client = testServer.Client()
		qe.cache = client.cache

		result, err := qe.executeQuery()

		// Analyze the request that was sent
		t.Logf("Request body size: %d bytes", len(receivedBody))
		t.Logf("Content-Type sent: %s", receivedContentType)

		// Check if the request was properly encoded
		if len(receivedBody) > 0 {
			var receivedRequest map[string]interface{}
			parseErr := json.Unmarshal(receivedBody, &receivedRequest)
			if parseErr != nil {
				t.Logf("Server couldn't parse JSON: %s", parseErr.Error())
				t.Logf("Raw body: %s", string(receivedBody))
			} else {
				t.Logf("Server successfully parsed JSON")
				if vars, ok := receivedRequest["variables"].(map[string]interface{}); ok {
					t.Logf("Received group_description: %s", vars["group_description"])
					t.Logf("Received user_name: %s", vars["user_name"])
				}
			}
		}

		// The test should pass - no trailing garbage errors
		assert.NoError(err, "Should not get trailing garbage errors with Unicode characters")
		assert.NotNil(result)
	})

	t.Run("should properly encode various Unicode characters", func(t *testing.T) {
		client := CreateTestClient()

		testCases := []struct {
			name        string
			description string
			variables   map[string]interface{}
		}{
			{
				name:        "emoji_only",
				description: "Test with various emojis",
				variables: map[string]interface{}{
					"message": "Hello 😀😃😄😁😆😅😂🤣😊😇🙂🙃😉😌😍🥰😘😗😙😚😋😛😝😜🤪🤨🧐🤓😎🤩🥳😏😒😞😔😟😕🙁☹️😣😖😫😩🥺😢😭😤😠😡🤬🤯😳🥵🥶😱😨😰😥😓🤗🤔🤭🤫🤥😶😐😑😬🙄😯😦😧😮😲🥱😴🤤😪😵🤐🥴🤢🤮🤧😷🤒🤕🤑🤠😈👿👹👺🤡💩👻💀☠️👽👾🤖🎃😺😸😹😻😼😽🙀😿😾",
				},
			},
			{
				name:        "chinese_characters",
				description: "Test with Chinese characters",
				variables: map[string]interface{}{
					"name":    "张三李四王五赵六",
					"address": "北京市朝阳区建国门外大街1号",
					"company": "中国国际贸易中心有限公司",
				},
			},
			{
				name:        "japanese_characters",
				description: "Test with Japanese characters",
				variables: map[string]interface{}{
					"name":    "田中太郎",
					"address": "東京都渋谷区渋谷1-1-1",
					"message": "こんにちは、世界！",
				},
			},
			{
				name:        "korean_characters",
				description: "Test with Korean characters",
				variables: map[string]interface{}{
					"name":    "김철수",
					"address": "서울특별시 강남구 테헤란로 123",
					"message": "안녕하세요, 세계!",
				},
			},
			{
				name:        "arabic_characters",
				description: "Test with Arabic characters",
				variables: map[string]interface{}{
					"name":    "محمد أحمد",
					"address": "الرياض، المملكة العربية السعودية",
					"message": "مرحبا بالعالم!",
				},
			},
			{
				name:        "mixed_unicode",
				description: "Test with mixed Unicode characters",
				variables: map[string]interface{}{
					"message": "Hello 世界 🌍 مرحبا こんにちは 안녕하세요 Здравствуй Bonjour ¡Hola! Γεια σας",
					"symbols": "©®™€£¥₹₽¢₩₪₫₱₡₦₨₹₽₴₸₼₾₿",
					"math":    "∑∏∫∂∇∆∞≠≤≥±×÷√∛∜∝∞",
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				query := "mutation testUnicode($message: String, $name: String, $address: String, $company: String, $symbols: String, $math: String) { test }"

				// Compile the query
				compiledQuery := client.compileQuery(query, tc.variables)
				assert.NotNil(compiledQuery, "Query should compile successfully for %s", tc.description)
				assert.NotNil(compiledQuery.JsonQuery, "JSON query should be generated for %s", tc.description)

				// Verify the JSON is valid
				assert.True(json.Valid(compiledQuery.JsonQuery), "JSON should be valid for %s", tc.description)

				// Verify we can parse it back
				var parsedQuery map[string]interface{}
				err := json.Unmarshal(compiledQuery.JsonQuery, &parsedQuery)
				assert.NoError(err, "Should be able to parse JSON for %s", tc.description)

				// Verify the variables are preserved correctly
				if parsedVars, ok := parsedQuery["variables"].(map[string]interface{}); ok {
					for key, expectedValue := range tc.variables {
						actualValue := parsedVars[key]
						assert.Equal(expectedValue, actualValue, "Variable %s should be preserved correctly for %s", key, tc.description)
					}
				}

				t.Logf("✓ %s: JSON size: %d bytes", tc.description, len(compiledQuery.JsonQuery))
			})
		}
	})

	t.Run("should handle Content-Type with charset specification", func(t *testing.T) {
		var receivedContentType string
		var receivedBody []byte

		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedContentType = r.Header.Get("Content-Type")
			body := make([]byte, r.ContentLength)
			r.Body.Read(body)
			receivedBody = body

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data":{"result":"success"}}`))
		}))
		defer testServer.Close()

		client := CreateTestClient()

		variables := map[string]interface{}{
			"emoji_text": "Test with emoji 😈 and Chinese 巴结",
		}

		query := "mutation test($emoji_text: String!) { test(text: $emoji_text) }"
		compiledQuery := client.compileQuery(query, variables)

		// Test different Content-Type headers
		contentTypes := []string{
			"application/json",
			"application/json; charset=utf-8",
			"application/json; charset=UTF-8",
		}

		for _, contentType := range contentTypes {
			t.Run("content_type_"+strings.ReplaceAll(contentType, " ", "_"), func(t *testing.T) {
				qe := &QueryExecutor{
					BaseClient: client,
					Query:      compiledQuery.JsonQuery,
					Headers:    map[string]interface{}{"Content-Type": contentType},
					CacheKey:   "no-cache",
					Retries:    false,
				}
				qe.endpoint = testServer.URL
				qe.client = testServer.Client()
				qe.cache = client.cache

				result, err := qe.executeQuery()

				assert.NoError(err, "Should handle Content-Type: %s", contentType)
				assert.NotNil(result)
				assert.Equal(contentType, receivedContentType, "Content-Type should be preserved")

				// Verify the body is valid JSON with Unicode characters
				var parsedBody map[string]interface{}
				parseErr := json.Unmarshal(receivedBody, &parsedBody)
				assert.NoError(parseErr, "Request body should be valid JSON")

				if vars, ok := parsedBody["variables"].(map[string]interface{}); ok {
					assert.Equal("Test with emoji 😈 and Chinese 巴结", vars["emoji_text"])
				}
			})
		}
	})
}

// TestUnicodeEncodingEdgeCases tests edge cases that might cause encoding issues
func TestUnicodeEncodingEdgeCases(t *testing.T) {
	assert := assertions.New(t)
	client := CreateTestClient()

	t.Run("should handle null bytes and control characters", func(t *testing.T) {
		// Test with potentially problematic characters
		variables := map[string]interface{}{
			"text_with_nulls":    "Text\x00with\x00null\x00bytes",
			"text_with_controls": "Text\x01\x02\x03\x04\x05with\x06\x07\x08controls",
			"text_with_tabs":     "Text\twith\ttabs\tand\nnewlines\r\n",
		}

		query := "mutation test($text_with_nulls: String, $text_with_controls: String, $text_with_tabs: String) { test }"
		compiledQuery := client.compileQuery(query, variables)

		assert.NotNil(compiledQuery)
		assert.NotNil(compiledQuery.JsonQuery)

		// The JSON should be valid (Go's json package handles escaping)
		assert.True(json.Valid(compiledQuery.JsonQuery))

		// Verify we can parse it back
		var parsedQuery map[string]interface{}
		err := json.Unmarshal(compiledQuery.JsonQuery, &parsedQuery)
		assert.NoError(err)

		t.Logf("JSON with control characters: %s", string(compiledQuery.JsonQuery))
	})

	t.Run("should handle very long Unicode strings", func(t *testing.T) {
		// Create a very long string with Unicode characters
		longUnicodeString := strings.Repeat("Hello 世界 😀 مرحبا ", 1000)

		variables := map[string]interface{}{
			"long_unicode": longUnicodeString,
		}

		query := "mutation test($long_unicode: String!) { test }"
		compiledQuery := client.compileQuery(query, variables)

		assert.NotNil(compiledQuery)
		assert.NotNil(compiledQuery.JsonQuery)
		assert.True(json.Valid(compiledQuery.JsonQuery))

		t.Logf("Long Unicode JSON size: %d bytes", len(compiledQuery.JsonQuery))

		// Verify the content is preserved
		var parsedQuery map[string]interface{}
		err := json.Unmarshal(compiledQuery.JsonQuery, &parsedQuery)
		assert.NoError(err)

		if vars, ok := parsedQuery["variables"].(map[string]interface{}); ok {
			assert.Equal(longUnicodeString, vars["long_unicode"])
		}
	})

	t.Run("should handle Unicode normalization edge cases", func(t *testing.T) {
		// Test different Unicode normalization forms
		variables := map[string]interface{}{
			"composed":   "é",       // U+00E9 (composed)
			"decomposed": "e\u0301", // U+0065 U+0301 (decomposed)
			"ligature":   "ﬁ",       // U+FB01 (ligature)
			"surrogate":  "𝕳𝖊𝖑𝖑𝖔",   // Mathematical script letters (surrogate pairs)
		}

		query := "mutation test($composed: String, $decomposed: String, $ligature: String, $surrogate: String) { test }"
		compiledQuery := client.compileQuery(query, variables)

		assert.NotNil(compiledQuery)
		assert.NotNil(compiledQuery.JsonQuery)
		assert.True(json.Valid(compiledQuery.JsonQuery))

		// Verify all forms are preserved
		var parsedQuery map[string]interface{}
		err := json.Unmarshal(compiledQuery.JsonQuery, &parsedQuery)
		assert.NoError(err)

		if vars, ok := parsedQuery["variables"].(map[string]interface{}); ok {
			for key, expectedValue := range variables {
				assert.Equal(expectedValue, vars[key], "Unicode form should be preserved for %s", key)
			}
		}

		t.Logf("Unicode normalization test JSON: %s", string(compiledQuery.JsonQuery))
	})
}
