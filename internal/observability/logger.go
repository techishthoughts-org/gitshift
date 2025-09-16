package observability

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"golang.org/x/term"
)

// ContextKey represents a custom type for context keys
type ContextKey string

// LogLevel represents the logging level
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Logger interface for structured logging
type Logger interface {
	Debug(ctx context.Context, msg string, fields ...Field)
	Info(ctx context.Context, msg string, fields ...Field)
	Warn(ctx context.Context, msg string, fields ...Field)
	Error(ctx context.Context, msg string, fields ...Field)
	Fatal(ctx context.Context, msg string, fields ...Field)

	WithContext(ctx context.Context) Logger
	WithFields(fields ...Field) Logger
}

// Field represents a key-value pair for structured logging
type Field struct {
	Key   string
	Value interface{}
}

// F creates a new Field
func F(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// Fields creates multiple fields from key-value pairs
func Fields(kv ...interface{}) []Field {
	if len(kv)%2 != 0 {
		return nil
	}

	var fields []Field
	for i := 0; i < len(kv); i += 2 {
		if key, ok := kv[i].(string); ok {
			fields = append(fields, F(key, kv[i+1]))
		}
	}
	return fields
}

// structuredLogger implements the Logger interface
type structuredLogger struct {
	level     LogLevel
	output    io.Writer
	fields    []Field
	context   context.Context
	useColors bool
}

// NewLogger creates a new logger instance
func NewLogger(level LogLevel) Logger {
	useColors := term.IsTerminal(int(os.Stdout.Fd()))

	return &structuredLogger{
		level:     level,
		output:    os.Stdout,
		fields:    make([]Field, 0),
		useColors: useColors,
	}
}

// NewDefaultLogger creates a logger with default settings
func NewDefaultLogger() Logger {
	level := LogLevelInfo
	if os.Getenv("GITPERSONA_DEBUG") == "true" {
		level = LogLevelDebug
	}
	return NewLogger(level)
}

// Debug logs a debug message
func (l *structuredLogger) Debug(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, LogLevelDebug, msg, fields...)
}

// Info logs an info message
func (l *structuredLogger) Info(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, LogLevelInfo, msg, fields...)
}

// Warn logs a warning message
func (l *structuredLogger) Warn(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, LogLevelWarn, msg, fields...)
}

// Error logs an error message
func (l *structuredLogger) Error(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, LogLevelError, msg, fields...)
}

// Fatal logs a fatal message and exits
func (l *structuredLogger) Fatal(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, LogLevelFatal, msg, fields...)
	os.Exit(1)
}

// WithContext returns a logger with the given context
func (l *structuredLogger) WithContext(ctx context.Context) Logger {
	newLogger := *l
	newLogger.context = ctx
	return &newLogger
}

// WithFields returns a logger with additional fields
func (l *structuredLogger) WithFields(fields ...Field) Logger {
	newLogger := *l
	newLogger.fields = append(newLogger.fields, fields...)
	return &newLogger
}

// log writes a log entry
func (l *structuredLogger) log(ctx context.Context, level LogLevel, msg string, fields ...Field) {
	if level < l.level {
		return
	}

	// Combine all fields
	allFields := make([]Field, 0, len(l.fields)+len(fields))
	allFields = append(allFields, l.fields...)
	allFields = append(allFields, fields...)

	// Add context fields
	if ctx != nil {
		if requestID := getRequestID(ctx); requestID != "" {
			allFields = append(allFields, F("request_id", requestID))
		}
		if userID := getUserID(ctx); userID != "" {
			allFields = append(allFields, F("user_id", userID))
		}
	}

	// Format the log entry
	entry := l.formatLogEntry(level, msg, allFields)

	// Write to output
	_, _ = fmt.Fprintln(l.output, entry)
}

// formatLogEntry formats a log entry
func (l *structuredLogger) formatLogEntry(level LogLevel, msg string, fields []Field) string {
	timestamp := time.Now().Format(time.RFC3339)

	// Start with timestamp and level
	entry := fmt.Sprintf("%s [%s] %s", timestamp, level.String(), msg)

	// Add fields
	if len(fields) > 0 {
		entry += " | "
		for i, field := range fields {
			if i > 0 {
				entry += " "
			}
			entry += fmt.Sprintf("%s=%v", field.Key, field.Value)
		}
	}

	return entry
}

// getRequestID extracts request ID from context
func getRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(ContextKey("request_id")).(string); ok {
		return id
	}
	return ""
}

// getUserID extracts user ID from context
func getUserID(ctx context.Context) string {
	if id, ok := ctx.Value(ContextKey("user_id")).(string); ok {
		return id
	}
	return ""
}

// Context helpers for adding values to context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, ContextKey("request_id"), requestID)
}

func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ContextKey("user_id"), userID)
}

// Convenience functions for common logging patterns
func LogCommandExecution(ctx context.Context, logger Logger, command string, args []string) {
	logger.Info(ctx, "executing_command",
		F("command", command),
		F("args", args),
	)
}

func LogCommandSuccess(ctx context.Context, logger Logger, command string, duration time.Duration) {
	logger.Info(ctx, "command_completed",
		F("command", command),
		F("duration", duration),
		F("status", "success"),
	)
}

func LogCommandError(ctx context.Context, logger Logger, command string, err error, duration time.Duration) {
	logger.Error(ctx, "command_failed",
		F("command", command),
		F("error", err.Error()),
		F("duration", duration),
		F("status", "error"),
	)
}

func LogAccountSwitch(ctx context.Context, logger Logger, fromAccount, toAccount string) {
	logger.Info(ctx, "account_switched",
		F("from_account", fromAccount),
		F("to_account", toAccount),
	)
}

func LogSSHValidation(ctx context.Context, logger Logger, account string, success bool, details string) {
	if success {
		logger.Info(ctx, "ssh_validation",
			F("account", account),
			F("success", success),
			F("details", details),
		)
	} else {
		logger.Error(ctx, "ssh_validation",
			F("account", account),
			F("success", success),
			F("details", details),
		)
	}
}

func LogConfigOperation(ctx context.Context, logger Logger, operation string, success bool, details string) {
	if success {
		logger.Info(ctx, "config_operation",
			F("operation", operation),
			F("success", success),
			F("details", details),
		)
	} else {
		logger.Error(ctx, "config_operation",
			F("operation", operation),
			F("success", success),
			F("details", details),
		)
	}
}
