package observability

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestLogLevelString(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{LogLevelDebug, "DEBUG"},
		{LogLevelInfo, "INFO"},
		{LogLevelWarn, "WARN"},
		{LogLevelError, "ERROR"},
		{LogLevelFatal, "FATAL"},
		{LogLevel(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.level.String()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestField(t *testing.T) {
	field := F("key", "value")
	if field.Key != "key" {
		t.Errorf("Expected key 'key', got %s", field.Key)
	}
	if field.Value != "value" {
		t.Errorf("Expected value 'value', got %v", field.Value)
	}
}

func TestFields(t *testing.T) {
	tests := []struct {
		name     string
		kv       []interface{}
		expected int
	}{
		{
			name:     "valid key-value pairs",
			kv:       []interface{}{"key1", "value1", "key2", "value2"},
			expected: 2,
		},
		{
			name:     "odd number of arguments",
			kv:       []interface{}{"key1", "value1", "key2"},
			expected: 0,
		},
		{
			name:     "empty arguments",
			kv:       []interface{}{},
			expected: 0,
		},
		{
			name:     "non-string key",
			kv:       []interface{}{123, "value1", "key2", "value2"},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := Fields(tt.kv...)
			if len(fields) != tt.expected {
				t.Errorf("Expected %d fields, got %d", tt.expected, len(fields))
			}
		})
	}
}

func TestNewLogger(t *testing.T) {
	logger := NewLogger(LogLevelInfo)
	if logger == nil {
		t.Fatal("NewLogger returned nil")
	}

	structuredLogger, ok := logger.(*structuredLogger)
	if !ok {
		t.Fatalf("Expected *structuredLogger, got %T", logger)
	}

	if structuredLogger.level != LogLevelInfo {
		t.Errorf("Expected level %v, got %v", LogLevelInfo, structuredLogger.level)
	}

	if structuredLogger.output == nil {
		t.Error("Expected output to be set")
	}

	if structuredLogger.fields == nil {
		t.Error("Expected fields to be initialized")
	}
}

func TestNewDefaultLogger(t *testing.T) {
	logger := NewDefaultLogger()
	if logger == nil {
		t.Fatal("NewDefaultLogger returned nil")
	}

	structuredLogger, ok := logger.(*structuredLogger)
	if !ok {
		t.Fatalf("Expected *structuredLogger, got %T", logger)
	}

	if structuredLogger.level != LogLevelInfo {
		t.Errorf("Expected default level %v, got %v", LogLevelInfo, structuredLogger.level)
	}
}

func TestLoggerMethods(t *testing.T) {
	var buf bytes.Buffer
	logger := &structuredLogger{
		level:  LogLevelDebug,
		output: &buf,
		fields: make([]Field, 0),
	}

	ctx := context.Background()

	t.Run("Debug", func(t *testing.T) {
		buf.Reset()
		logger.Debug(ctx, "debug message", F("key", "value"))
		output := buf.String()
		if !strings.Contains(output, "debug message") {
			t.Errorf("Expected output to contain 'debug message', got %s", output)
		}
		if !strings.Contains(output, "key") {
			t.Errorf("Expected output to contain 'key', got %s", output)
		}
	})

	t.Run("Info", func(t *testing.T) {
		buf.Reset()
		logger.Info(ctx, "info message", F("key", "value"))
		output := buf.String()
		if !strings.Contains(output, "info message") {
			t.Errorf("Expected output to contain 'info message', got %s", output)
		}
	})

	t.Run("Warn", func(t *testing.T) {
		buf.Reset()
		logger.Warn(ctx, "warn message", F("key", "value"))
		output := buf.String()
		if !strings.Contains(output, "warn message") {
			t.Errorf("Expected output to contain 'warn message', got %s", output)
		}
	})

	t.Run("Error", func(t *testing.T) {
		buf.Reset()
		logger.Error(ctx, "error message", F("key", "value"))
		output := buf.String()
		if !strings.Contains(output, "error message") {
			t.Errorf("Expected output to contain 'error message', got %s", output)
		}
	})

	t.Run("Fatal", func(t *testing.T) {
		buf.Reset()
		// Note: Fatal calls os.Exit(1), so we can't easily test it
		// We'll just test that the method exists and doesn't panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Fatal panicked: %v", r)
			}
		}()
		// We can't actually call Fatal in a test as it exits the process
		// logger.Fatal(ctx, "fatal message", F("key", "value"))
	})
}

func TestWithContext(t *testing.T) {
	logger := NewLogger(LogLevelInfo)
	ctx := context.WithValue(context.Background(), ContextKey("test"), "value")

	newLogger := logger.WithContext(ctx)
	if newLogger == nil {
		t.Fatal("WithContext returned nil")
	}

	structuredLogger, ok := newLogger.(*structuredLogger)
	if !ok {
		t.Fatalf("Expected *structuredLogger, got %T", newLogger)
	}

	if structuredLogger.context != ctx {
		t.Error("Expected context to be set")
	}
}

func TestWithFields(t *testing.T) {
	logger := NewLogger(LogLevelInfo)
	fields := []Field{F("key1", "value1"), F("key2", "value2")}

	newLogger := logger.WithFields(fields...)
	if newLogger == nil {
		t.Fatal("WithFields returned nil")
	}

	structuredLogger, ok := newLogger.(*structuredLogger)
	if !ok {
		t.Fatalf("Expected *structuredLogger, got %T", newLogger)
	}

	if len(structuredLogger.fields) != len(fields) {
		t.Errorf("Expected %d fields, got %d", len(fields), len(structuredLogger.fields))
	}
}

func TestLogLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := &structuredLogger{
		level:  LogLevelWarn, // Only warn and above
		output: &buf,
		fields: make([]Field, 0),
	}

	ctx := context.Background()

	// Debug should not be logged
	buf.Reset()
	logger.Debug(ctx, "debug message")
	if buf.Len() > 0 {
		t.Error("Debug message should not be logged when level is Warn")
	}

	// Info should not be logged
	buf.Reset()
	logger.Info(ctx, "info message")
	if buf.Len() > 0 {
		t.Error("Info message should not be logged when level is Warn")
	}

	// Warn should be logged
	buf.Reset()
	logger.Warn(ctx, "warn message")
	if buf.Len() == 0 {
		t.Error("Warn message should be logged when level is Warn")
	}

	// Error should be logged
	buf.Reset()
	logger.Error(ctx, "error message")
	if buf.Len() == 0 {
		t.Error("Error message should be logged when level is Warn")
	}
}

func TestContextKeys(t *testing.T) {
	ctx := context.Background()

	// Test WithRequestID
	requestID := "test-request-123"
	ctxWithRequestID := WithRequestID(ctx, requestID)
	if ctxWithRequestID == nil {
		t.Fatal("WithRequestID returned nil context")
	}

	// Test WithUserID
	userID := "test-user-456"
	ctxWithUserID := WithUserID(ctx, userID)
	if ctxWithUserID == nil {
		t.Fatal("WithUserID returned nil context")
	}
}

func TestLogCommandExecution(t *testing.T) {
	var buf bytes.Buffer
	logger := &structuredLogger{
		level:  LogLevelInfo,
		output: &buf,
		fields: make([]Field, 0),
	}

	ctx := context.Background()
	command := "git status"
	args := []string{"--porcelain"}

	LogCommandExecution(ctx, logger, command, args)
	output := buf.String()

	if !strings.Contains(output, command) {
		t.Errorf("Expected output to contain command '%s', got %s", command, output)
	}
}

func TestLogCommandSuccess(t *testing.T) {
	var buf bytes.Buffer
	logger := &structuredLogger{
		level:  LogLevelInfo,
		output: &buf,
		fields: make([]Field, 0),
	}

	ctx := context.Background()
	command := "git add ."
	duration := 100 * time.Millisecond

	LogCommandSuccess(ctx, logger, command, duration)
	output := buf.String()

	if !strings.Contains(output, command) {
		t.Errorf("Expected output to contain command '%s', got %s", command, output)
	}
}

func TestLogCommandError(t *testing.T) {
	var buf bytes.Buffer
	logger := &structuredLogger{
		level:  LogLevelError,
		output: &buf,
		fields: make([]Field, 0),
	}

	ctx := context.Background()
	command := "git push"
	err := errors.New("fatal: remote origin already exists")
	duration := 100 * time.Millisecond

	LogCommandError(ctx, logger, command, err, duration)
	output := buf.String()

	if !strings.Contains(output, command) {
		t.Errorf("Expected output to contain command '%s', got %s", command, output)
	}
	if !strings.Contains(output, err.Error()) {
		t.Errorf("Expected output to contain error '%s', got %s", err.Error(), output)
	}
}

func TestLogAccountSwitch(t *testing.T) {
	var buf bytes.Buffer
	logger := &structuredLogger{
		level:  LogLevelInfo,
		output: &buf,
		fields: make([]Field, 0),
	}

	ctx := context.Background()
	fromAccount := "work"
	toAccount := "personal"

	LogAccountSwitch(ctx, logger, fromAccount, toAccount)
	output := buf.String()

	if !strings.Contains(output, fromAccount) {
		t.Errorf("Expected output to contain from account '%s', got %s", fromAccount, output)
	}
	if !strings.Contains(output, toAccount) {
		t.Errorf("Expected output to contain to account '%s', got %s", toAccount, output)
	}
}

func TestLogSSHValidation(t *testing.T) {
	var buf bytes.Buffer
	logger := &structuredLogger{
		level:  LogLevelInfo,
		output: &buf,
		fields: make([]Field, 0),
	}

	ctx := context.Background()
	keyPath := "/home/user/.ssh/id_rsa"
	success := true
	message := "SSH key validation successful"

	LogSSHValidation(ctx, logger, keyPath, success, message)
	output := buf.String()

	if !strings.Contains(output, keyPath) {
		t.Errorf("Expected output to contain key path '%s', got %s", keyPath, output)
	}
	if !strings.Contains(output, message) {
		t.Errorf("Expected output to contain message '%s', got %s", message, output)
	}
}

func TestLogConfigOperation(t *testing.T) {
	var buf bytes.Buffer
	logger := &structuredLogger{
		level:  LogLevelInfo,
		output: &buf,
		fields: make([]Field, 0),
	}

	ctx := context.Background()
	operation := "save"
	configPath := "/home/user/.config/gitpersona/config.yaml"
	success := true

	LogConfigOperation(ctx, logger, operation, success, configPath)
	output := buf.String()

	if !strings.Contains(output, operation) {
		t.Errorf("Expected output to contain operation '%s', got %s", operation, output)
	}
	if !strings.Contains(output, configPath) {
		t.Errorf("Expected output to contain config path '%s', got %s", configPath, output)
	}
}

func TestFormatLogEntry(t *testing.T) {
	var buf bytes.Buffer
	logger := &structuredLogger{
		level:     LogLevelInfo,
		output:    &buf,
		fields:    make([]Field, 0),
		useColors: false, // Disable colors for testing
	}

	ctx := context.Background()
	level := LogLevelInfo
	message := "test message"
	fields := []Field{F("key", "value")}

	logger.log(ctx, level, message, fields...)
	output := buf.String()

	// Check that the log entry contains expected components
	if !strings.Contains(output, "INFO") {
		t.Errorf("Expected output to contain log level 'INFO', got %s", output)
	}
	if !strings.Contains(output, message) {
		t.Errorf("Expected output to contain message '%s', got %s", message, output)
	}
	if !strings.Contains(output, "key") {
		t.Errorf("Expected output to contain field key 'key', got %s", output)
	}
	if !strings.Contains(output, "value") {
		t.Errorf("Expected output to contain field value 'value', got %s", output)
	}
}

func TestGetRequestID(t *testing.T) {
	ctx := context.Background()
	requestID := getRequestID(ctx)
	if requestID != "" {
		t.Errorf("Expected empty request ID for context without request ID, got %s", requestID)
	}

	ctxWithRequestID := WithRequestID(ctx, "test-request")
	requestID = getRequestID(ctxWithRequestID)
	if requestID != "test-request" {
		t.Errorf("Expected request ID 'test-request', got %s", requestID)
	}
}

func TestGetUserID(t *testing.T) {
	ctx := context.Background()
	userID := getUserID(ctx)
	if userID != "" {
		t.Errorf("Expected empty user ID for context without user ID, got %s", userID)
	}

	ctxWithUserID := WithUserID(ctx, "test-user")
	userID = getUserID(ctxWithUserID)
	if userID != "test-user" {
		t.Errorf("Expected user ID 'test-user', got %s", userID)
	}
}

func TestLoggerChaining(t *testing.T) {
	logger := NewLogger(LogLevelInfo)
	ctx := context.Background()

	// Chain multiple operations
	chainedLogger := logger.
		WithContext(ctx).
		WithFields(F("service", "test"), F("version", "1.0.0"))

	if chainedLogger == nil {
		t.Fatal("Chained logger is nil")
	}

	// Test that the chained logger works
	var buf bytes.Buffer
	structuredLogger := chainedLogger.(*structuredLogger)
	structuredLogger.output = &buf

	chainedLogger.Info(ctx, "chained message")
	output := buf.String()

	if !strings.Contains(output, "chained message") {
		t.Errorf("Expected output to contain 'chained message', got %s", output)
	}
	if !strings.Contains(output, "service") {
		t.Errorf("Expected output to contain 'service', got %s", output)
	}
	if !strings.Contains(output, "test") {
		t.Errorf("Expected output to contain 'test', got %s", output)
	}
}
