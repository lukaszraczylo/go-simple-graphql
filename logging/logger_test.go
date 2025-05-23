package libpack_logger

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/goccy/go-reflect"
)

func (suite *LoggerTestSuite) Test_LogMessageString() {
	msg := &LogMessage{
		Message: "test message",
	}

	assert.Equal("test message", msg.String())
}

func callLoggerMethod(logger *Logger, methodName string, message *LogMessage) {
	// Get the method by name using reflection
	method := reflect.ValueOf(logger).MethodByName(methodName)
	if method.IsValid() {
		// Call the method with the message as an argument
		method.Call([]reflect.Value{reflect.ValueOf(message)})
	} else {
		fmt.Printf("Method %s does not exist on Logger\n", methodName)
	}
}

func (suite *LoggerTestSuite) Test_LogsLevelsPrint() {
	output := &bytes.Buffer{}
	logger := New().SetOutput(output)

	tests := []struct {
		pairs           map[string]any
		name            string
		method          string
		message         string
		loggerMinLevel  int
		messageLogLevel int
		wantOutput      bool
	}{
		{
			name:            "Log: Debug, Level: Debug - no pairs",
			method:          "Debug",
			loggerMinLevel:  LEVEL_DEBUG,
			messageLogLevel: LEVEL_DEBUG,
			message:         "debug message",
			wantOutput:      true,
		},
		{
			name:            "Log: Info, Level: Info - one pair",
			method:          "Info",
			loggerMinLevel:  LEVEL_INFO,
			messageLogLevel: LEVEL_INFO,
			message:         "info message",
			pairs: map[string]any{
				"key": "value",
			},
			wantOutput: true,
		},
		{
			name:            "Log: Info, Level: Warn - with pairs",
			method:          "Info",
			loggerMinLevel:  LEVEL_WARN,
			messageLogLevel: LEVEL_INFO,
			message:         "warn message",
			pairs: map[string]any{
				"key1": "value1",
				"key2": "value2",
			},
			wantOutput: false,
		},
		{
			name:            "Log: Warn, Level: Info - with 500 pairs",
			method:          "Warn",
			loggerMinLevel:  LEVEL_INFO,
			messageLogLevel: LEVEL_WARN,
			message:         "warn message with 500 pairs",
			pairs: func() map[string]any {
				pairs := make(map[string]any)
				for i := 0; i < 500; i++ {
					pairs[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
				}
				return pairs
			}(),
			wantOutput: true,
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			msg := &LogMessage{
				Message: tt.message,
				Pairs:   tt.pairs,
			}
			output.Reset()

			// Set logger's minimum log level
			logger.SetMinLogLevel(tt.loggerMinLevel)
			fmt.Println("Logger min log level:", LevelNames[logger.minLogLevel])

			// Call the logging method
			callLoggerMethod(logger, tt.method, msg)

			logOutput := output.String()
			fmt.Println("Output:", logOutput)

			if tt.wantOutput {
				var loggedMessage map[string]any
				err := json.Unmarshal([]byte(logOutput), &loggedMessage)
				if err != nil {
					t.Fatalf("Error unmarshalling log message: %v\nLog output: %s", err, logOutput)
				}

				if !containsLogMessage(logOutput, tt.message) {
					t.Errorf("Expected log message %q, but got %q", tt.message, logOutput)
				}
				assert.Equal(LevelNames[tt.messageLogLevel], loggedMessage["level"])
				if tt.pairs != nil {
					for k, v := range tt.pairs {
						assert.Equal(v, loggedMessage[k])
					}
				}
			} else {
				assert.Equal("", logOutput)
			}
		})
	}
}

func containsLogMessage(logOutput, expectedMessage string) bool {
	return bytes.Contains([]byte(logOutput), []byte(expectedMessage))
}

func (suite *LoggerTestSuite) Test_SetFormat() {
	logger := New().SetFormat(time.RFC3339Nano)

	assert.Equal(time.RFC3339Nano, logger.format)
}

func (suite *LoggerTestSuite) Test_SetMinLogLevel() {
	logger := New().SetMinLogLevel(LEVEL_DEBUG)

	assert.Equal(LEVEL_DEBUG, logger.minLogLevel)
}

func (suite *LoggerTestSuite) Test_ShouldLog() {
	logger := New().SetMinLogLevel(LEVEL_WARN)

	assert.True(logger.shouldLog(LEVEL_WARN))
	assert.True(logger.shouldLog(LEVEL_ERROR))
	assert.False(logger.shouldLog(LEVEL_INFO))
	assert.False(logger.shouldLog(LEVEL_DEBUG))
}

func (suite *LoggerTestSuite) Test_GetLogLevel() {
	tests := []struct {
		name     string
		level    string
		expected int
	}{
		{"debug level", "debug", LEVEL_DEBUG},
		{"info level", "info", LEVEL_INFO},
		{"warn level", "warn", LEVEL_WARN},
		{"error level", "error", LEVEL_ERROR},
		{"fatal level", "fatal", LEVEL_FATAL},
		{"uppercase debug", "DEBUG", LEVEL_DEBUG},
		{"mixed case warn", "WaRn", LEVEL_WARN},
		{"unknown level", "unknown", defaultMinLevel},
		{"empty string", "", defaultMinLevel},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			result := GetLogLevel(tt.level)
			assert.Equal(tt.expected, result)
		})
	}
}

func (suite *LoggerTestSuite) Test_Warning() {
	output := &bytes.Buffer{}
	logger := New().SetOutput(output).SetMinLogLevel(LEVEL_WARN)

	msg := &LogMessage{
		Message: "warning message",
		Pairs:   map[string]any{"key": "value"},
	}

	logger.Warning(msg)

	logOutput := output.String()
	fmt.Printf("DEBUG: logOutput = %q\n", logOutput)
	fmt.Printf("DEBUG: logOutput length = %d\n", len(logOutput))
	assert.NotEmpty(logOutput)

	var loggedMessage map[string]any
	err := json.Unmarshal([]byte(logOutput), &loggedMessage)
	fmt.Printf("DEBUG: unmarshal error = %v\n", err)
	fmt.Printf("DEBUG: loggedMessage = %+v\n", loggedMessage)
	assert.NoError(err)
	assert.Equal("warn", loggedMessage["level"])
	assert.Equal("warning message", loggedMessage["message"])
}

func (suite *LoggerTestSuite) Test_Error() {
	output := &bytes.Buffer{}
	logger := New().SetOutput(output).SetMinLogLevel(LEVEL_ERROR)

	msg := &LogMessage{
		Message: "error message",
		Pairs:   map[string]any{"error": "test error"},
	}

	logger.Error(msg)

	logOutput := output.String()
	assert.NotEmpty(logOutput)

	var loggedMessage map[string]any
	err := json.Unmarshal([]byte(logOutput), &loggedMessage)
	assert.NoError(err)
	assert.Equal("error", loggedMessage["level"])
	assert.Equal("error message", loggedMessage["message"])
	assert.Equal("test error", loggedMessage["error"])
}

func (suite *LoggerTestSuite) Test_Fatal() {
	output := &bytes.Buffer{}
	logger := New().SetOutput(output).SetMinLogLevel(LEVEL_FATAL)

	msg := &LogMessage{
		Message: "fatal message",
		Pairs:   map[string]any{"fatal": "test fatal"},
	}

	logger.Fatal(msg)

	logOutput := output.String()
	assert.NotEmpty(logOutput)

	var loggedMessage map[string]any
	err := json.Unmarshal([]byte(logOutput), &loggedMessage)
	assert.NoError(err)
	assert.Equal("fatal", loggedMessage["level"])
	assert.Equal("fatal message", loggedMessage["message"])
	assert.Equal("test fatal", loggedMessage["fatal"])
}

func (suite *LoggerTestSuite) Test_SetFieldName() {
	// Save original field names
	originalFieldNames := make(map[string]string)
	for k, v := range fieldNames {
		originalFieldNames[k] = v
	}

	// Restore original field names after test
	defer func() {
		for k, v := range originalFieldNames {
			fieldNames[k] = v
		}
	}()

	logger := New()

	// Test setting custom field names
	logger.SetFieldName("timestamp", "time")
	logger.SetFieldName("level", "severity")
	logger.SetFieldName("message", "msg")

	output := &bytes.Buffer{}
	logger.SetOutput(output)

	msg := &LogMessage{
		Message: "test message",
	}

	logger.Info(msg)

	logOutput := output.String()
	assert.NotEmpty(logOutput)

	var loggedMessage map[string]any
	err := json.Unmarshal([]byte(logOutput), &loggedMessage)
	assert.NoError(err)

	// Check that custom field names are used
	assert.Contains(loggedMessage, "time")
	assert.Contains(loggedMessage, "severity")
	assert.Contains(loggedMessage, "msg")
	assert.Equal("info", loggedMessage["severity"])
	assert.Equal("test message", loggedMessage["msg"])
}

func (suite *LoggerTestSuite) Test_SetShowCaller() {
	output := &bytes.Buffer{}
	logger := New().SetOutput(output).SetShowCaller(true)

	msg := &LogMessage{
		Message: "test message with caller",
	}

	logger.Info(msg)

	logOutput := output.String()
	assert.NotEmpty(logOutput)

	var loggedMessage map[string]any
	err := json.Unmarshal([]byte(logOutput), &loggedMessage)
	assert.NoError(err)

	// Check that caller information is included
	assert.Contains(loggedMessage, "caller")
	caller, ok := loggedMessage["caller"].(string)
	assert.True(ok)
	assert.Contains(caller, "logger_test.go:")
}

func (suite *LoggerTestSuite) Test_GetCaller() {
	// Test getCaller function indirectly through SetShowCaller
	output := &bytes.Buffer{}
	logger := New().SetOutput(output).SetShowCaller(true).SetMinLogLevel(LEVEL_DEBUG)

	msg := &LogMessage{
		Message: "caller test",
	}

	logger.Debug(msg)

	logOutput := output.String()
	assert.NotEmpty(logOutput)

	var loggedMessage map[string]any
	err := json.Unmarshal([]byte(logOutput), &loggedMessage)
	assert.NoError(err)

	caller, exists := loggedMessage["caller"]
	assert.True(exists)
	assert.NotEmpty(caller)

	// Caller should be in format "file:line"
	callerStr, ok := caller.(string)
	assert.True(ok)
	assert.Contains(callerStr, ":")
}

func (suite *LoggerTestSuite) Test_LogWithNilPairs() {
	output := &bytes.Buffer{}
	logger := New().SetOutput(output)

	msg := &LogMessage{
		Message: "message with nil pairs",
		Pairs:   nil,
	}

	logger.Info(msg)

	logOutput := output.String()
	assert.NotEmpty(logOutput)

	var loggedMessage map[string]any
	err := json.Unmarshal([]byte(logOutput), &loggedMessage)
	assert.NoError(err)
	assert.Equal("info", loggedMessage["level"])
	assert.Equal("message with nil pairs", loggedMessage["message"])
}

func (suite *LoggerTestSuite) Test_LogLevelFiltering() {
	output := &bytes.Buffer{}
	logger := New().SetOutput(output).SetMinLogLevel(LEVEL_WARN)

	// These should not produce output (below minimum level)
	logger.Debug(&LogMessage{Message: "debug message"})
	logger.Info(&LogMessage{Message: "info message"})

	debugInfoOutput := output.String()
	assert.Empty(debugInfoOutput)

	// These should produce output (at or above minimum level)
	logger.Warn(&LogMessage{Message: "warn message"})
	warnOutput := output.String()
	assert.NotEmpty(warnOutput)
	assert.Contains(warnOutput, "warn message")

	output.Reset()
	logger.Error(&LogMessage{Message: "error message"})
	errorOutput := output.String()
	assert.NotEmpty(errorOutput)
	assert.Contains(errorOutput, "error message")

	output.Reset()
	logger.Fatal(&LogMessage{Message: "fatal message"})
	fatalOutput := output.String()
	assert.NotEmpty(fatalOutput)
	assert.Contains(fatalOutput, "fatal message")
}
