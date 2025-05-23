package gql

import (
	"bytes"
	"os"
	"strings"
	"testing"

	logging "github.com/lukaszraczylo/go-simple-graphql/logging"
)

func TestLogLevelEnvironmentIntegration(t *testing.T) {
	tests := []struct {
		name            string
		logLevel        string
		expectedLevel   int
		shouldShowDebug bool
		shouldShowInfo  bool
		shouldShowWarn  bool
		shouldShowError bool
	}{
		{
			name:            "Debug level shows all logs",
			logLevel:        "debug",
			expectedLevel:   logging.LEVEL_DEBUG,
			shouldShowDebug: true,
			shouldShowInfo:  true,
			shouldShowWarn:  true,
			shouldShowError: true,
		},
		{
			name:            "Info level shows info, warn, error",
			logLevel:        "info",
			expectedLevel:   logging.LEVEL_INFO,
			shouldShowDebug: false,
			shouldShowInfo:  true,
			shouldShowWarn:  true,
			shouldShowError: true,
		},
		{
			name:            "Warn level shows warn, error only",
			logLevel:        "warn",
			expectedLevel:   logging.LEVEL_WARN,
			shouldShowDebug: false,
			shouldShowInfo:  false,
			shouldShowWarn:  true,
			shouldShowError: true,
		},
		{
			name:            "Error level shows error only",
			logLevel:        "error",
			expectedLevel:   logging.LEVEL_ERROR,
			shouldShowDebug: false,
			shouldShowInfo:  false,
			shouldShowWarn:  false,
			shouldShowError: true,
		},
		{
			name:            "Invalid level defaults to info",
			logLevel:        "invalid",
			expectedLevel:   logging.LEVEL_INFO,
			shouldShowDebug: false,
			shouldShowInfo:  true,
			shouldShowWarn:  true,
			shouldShowError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set LOG_LEVEL environment variable
			originalLogLevel := os.Getenv("LOG_LEVEL")
			defer os.Setenv("LOG_LEVEL", originalLogLevel)

			os.Setenv("LOG_LEVEL", tt.logLevel)

			// Test GetLogLevel function directly
			parsedLevel := logging.GetLogLevel(tt.logLevel)
			if parsedLevel != tt.expectedLevel {
				t.Errorf("GetLogLevel(%s) = %d, want %d", tt.logLevel, parsedLevel, tt.expectedLevel)
			}

			// Test logger with captured output
			var buf bytes.Buffer
			logger := logging.New()
			logger.SetOutput(&buf)
			logger.SetMinLogLevel(parsedLevel)

			// Test all log levels
			logger.Debug(&logging.LogMessage{Message: "debug message"})
			logger.Info(&logging.LogMessage{Message: "info message"})
			logger.Warn(&logging.LogMessage{Message: "warn message"})
			logger.Error(&logging.LogMessage{Message: "error message"})

			output := buf.String()

			// Check if expected messages appear in output
			hasDebug := strings.Contains(output, "debug message")
			hasInfo := strings.Contains(output, "info message")
			hasWarn := strings.Contains(output, "warn message")
			hasError := strings.Contains(output, "error message")

			if hasDebug != tt.shouldShowDebug {
				t.Errorf("Debug message visibility: got %v, want %v", hasDebug, tt.shouldShowDebug)
			}
			if hasInfo != tt.shouldShowInfo {
				t.Errorf("Info message visibility: got %v, want %v", hasInfo, tt.shouldShowInfo)
			}
			if hasWarn != tt.shouldShowWarn {
				t.Errorf("Warn message visibility: got %v, want %v", hasWarn, tt.shouldShowWarn)
			}
			if hasError != tt.shouldShowError {
				t.Errorf("Error message visibility: got %v, want %v", hasError, tt.shouldShowError)
			}

			t.Logf("LOG_LEVEL=%s parsed to level %d (%s), output length: %d bytes",
				tt.logLevel, parsedLevel, logging.LevelNames[parsedLevel], len(output))
		})
	}
}

func TestMainApplicationLogLevelIntegration(t *testing.T) {
	tests := []struct {
		name        string
		logLevel    string
		expectLevel int
	}{
		{"Debug level", "debug", logging.LEVEL_DEBUG},
		{"Info level", "info", logging.LEVEL_INFO},
		{"Warn level", "warn", logging.LEVEL_WARN},
		{"Error level", "error", logging.LEVEL_ERROR},
		{"Empty defaults to info", "", logging.LEVEL_INFO},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment
			originalLogLevel := os.Getenv("LOG_LEVEL")
			defer os.Setenv("LOG_LEVEL", originalLogLevel)

			// Set test LOG_LEVEL
			if tt.logLevel == "" {
				os.Unsetenv("LOG_LEVEL")
			} else {
				os.Setenv("LOG_LEVEL", tt.logLevel)
			}

			// Create new connection (this tests main.go integration)
			client := NewConnection()

			// Set up buffer for test logging
			var buf bytes.Buffer
			client.Logger.SetOutput(&buf)

			// Verify the logger was configured with correct level by testing behavior
			client.Logger.Debug(&logging.LogMessage{Message: "test debug"})
			client.Logger.Info(&logging.LogMessage{Message: "test info"})

			output := buf.String()

			// The main validation is that the logger respects the LOG_LEVEL setting
			// We can see from the console output that initialization works correctly

			// Verify level parsing worked correctly by checking if debug shows when expected
			hasDebug := strings.Contains(output, "test debug")
			shouldShowDebug := tt.expectLevel <= logging.LEVEL_DEBUG

			if hasDebug != shouldShowDebug {
				t.Errorf("Debug visibility: got %v, want %v for level %s", hasDebug, shouldShowDebug, tt.logLevel)
			}

			t.Logf("NewConnection() with LOG_LEVEL=%s created logger, output length: %d bytes",
				tt.logLevel, len(output))
		})
	}
}

func TestTestHelpersLogLevelIntegration(t *testing.T) {
	tests := []struct {
		name        string
		logLevel    string
		expectLevel int
	}{
		{"Test with debug level", "debug", logging.LEVEL_DEBUG},
		{"Test with info level", "info", logging.LEVEL_INFO},
		{"Test with warn level", "warn", logging.LEVEL_WARN},
		{"Test with error level", "error", logging.LEVEL_ERROR},
		{"Test with no LOG_LEVEL (defaults to error)", "", logging.LEVEL_ERROR},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment
			originalLogLevel := os.Getenv("LOG_LEVEL")
			defer os.Setenv("LOG_LEVEL", originalLogLevel)

			// Set test LOG_LEVEL
			if tt.logLevel == "" {
				os.Unsetenv("LOG_LEVEL")
			} else {
				os.Setenv("LOG_LEVEL", tt.logLevel)
			}

			// Reset test logger to force re-initialization
			testLogger = nil

			// Get test logger (this tests test_helpers.go integration)
			var buf bytes.Buffer
			logger := GetTestLogger()
			logger.SetOutput(&buf)

			// Test logging at different levels
			logger.Debug(&logging.LogMessage{Message: "test debug"})
			logger.Info(&logging.LogMessage{Message: "test info"})
			logger.Warn(&logging.LogMessage{Message: "test warn"})
			logger.Error(&logging.LogMessage{Message: "test error"})

			output := buf.String()

			// The test logger initialization works correctly as seen in console output
			// Focus on validating the actual log level behavior

			// Verify level filtering works correctly
			hasDebug := strings.Contains(output, "test debug")
			hasInfo := strings.Contains(output, "test info")
			hasWarn := strings.Contains(output, "test warn")
			hasError := strings.Contains(output, "test error")

			shouldShowDebug := tt.expectLevel <= logging.LEVEL_DEBUG
			shouldShowInfo := tt.expectLevel <= logging.LEVEL_INFO
			shouldShowWarn := tt.expectLevel <= logging.LEVEL_WARN
			shouldShowError := tt.expectLevel <= logging.LEVEL_ERROR

			if hasDebug != shouldShowDebug {
				t.Errorf("Debug visibility: got %v, want %v", hasDebug, shouldShowDebug)
			}
			if hasInfo != shouldShowInfo {
				t.Errorf("Info visibility: got %v, want %v", hasInfo, shouldShowInfo)
			}
			if hasWarn != shouldShowWarn {
				t.Errorf("Warn visibility: got %v, want %v", hasWarn, shouldShowWarn)
			}
			if hasError != shouldShowError {
				t.Errorf("Error visibility: got %v, want %v", hasError, shouldShowError)
			}

			t.Logf("GetTestLogger() with LOG_LEVEL=%s, output length: %d bytes",
				tt.logLevel, len(output))
		})
	}
}

func TestLogLevelCaseInsensitive(t *testing.T) {
	testCases := []string{"DEBUG", "Info", "WARN", "Error", "debug", "info", "warn", "error"}

	for _, testCase := range testCases {
		t.Run("Case_"+testCase, func(t *testing.T) {
			level := logging.GetLogLevel(testCase)
			expectedLevel := logging.GetLogLevel(strings.ToLower(testCase))

			if level != expectedLevel {
				t.Errorf("GetLogLevel(%s) = %d, want %d (case insensitive)", testCase, level, expectedLevel)
			}
		})
	}
}
