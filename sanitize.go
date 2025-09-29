package gql

import (
	"regexp"
	"strings"
)

// Sensitive data patterns to redact from logs
var (
	// Pattern for common sensitive field names in JSON/GraphQL
	sensitiveFieldPattern = regexp.MustCompile(`(?i)"?(password|passwd|pwd|token|apikey|api_key|api-key|secret|authorization|auth|bearer|credentials|private_key|private-key|access_token|refresh_token|client_secret|session|cookie)"?\s*:\s*"[^"]*"`)

	// Pattern for authorization headers
	authHeaderPattern = regexp.MustCompile(`(?i)(authorization|x-api-key|x-auth-token|bearer)\s*:\s*[^\s,}]+`)

	// Pattern for basic auth in URLs
	basicAuthURLPattern = regexp.MustCompile(`(https?://)([^:]+):([^@]+)@`)

	// Pattern for JWT tokens
	jwtPattern = regexp.MustCompile(`eyJ[A-Za-z0-9_-]+\.eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`)
)

const redactedText = "[REDACTED]"

// sanitizeForLogging removes sensitive data from strings before logging
// This includes passwords, tokens, API keys, authorization headers, etc.
func sanitizeForLogging(input string) string {
	if input == "" {
		return input
	}

	// Create a working copy
	sanitized := input

	// Redact sensitive fields in JSON/GraphQL
	sanitized = sensitiveFieldPattern.ReplaceAllStringFunc(sanitized, func(match string) string {
		// Preserve the field name but redact the value
		parts := strings.SplitN(match, ":", 2)
		if len(parts) == 2 {
			fieldName := parts[0]
			return fieldName + `: "` + redactedText + `"`
		}
		return redactedText
	})

	// Redact authorization headers
	sanitized = authHeaderPattern.ReplaceAllStringFunc(sanitized, func(match string) string {
		parts := strings.SplitN(match, ":", 2)
		if len(parts) == 2 {
			headerName := parts[0]
			return headerName + ": " + redactedText
		}
		return redactedText
	})

	// Redact basic auth in URLs
	sanitized = basicAuthURLPattern.ReplaceAllString(sanitized, "${1}"+redactedText+":"+redactedText+"@")

	// Redact JWT tokens
	sanitized = jwtPattern.ReplaceAllString(sanitized, redactedText)

	return sanitized
}
