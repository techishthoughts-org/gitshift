package observability

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
)

// Logger provides structured logging following 2025 best practices
type Logger struct {
	*slog.Logger
	level slog.Level
}

// LogLevel represents available log levels
type LogLevel string

const (
	LevelDebug LogLevel = "debug"
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
)

// NewLogger creates a new structured logger with beautiful terminal output
func NewLogger(level LogLevel) *Logger {
	var slogLevel slog.Level
	switch level {
	case LevelDebug:
		slogLevel = slog.LevelDebug
	case LevelWarn:
		slogLevel = slog.LevelWarn
	case LevelError:
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	// Use tint for beautiful terminal logging (2025 standard)
	handler := tint.NewHandler(os.Stderr, &tint.Options{
		Level:      slogLevel,
		TimeFormat: time.Kitchen,
		NoColor:    os.Getenv("NO_COLOR") != "",
	})

	logger := slog.New(handler)

	return &Logger{
		Logger: logger,
		level:  slogLevel,
	}
}

// WithContext adds context to the logger
func (l *Logger) WithContext(ctx context.Context) *Logger {
	return &Logger{
		Logger: l.Logger.With("trace_id", getTraceID(ctx)),
		level:  l.level,
	}
}

// AccountEvent logs account-related operations
func (l *Logger) AccountEvent(action string, account string, details map[string]any) {
	l.Info("Account operation",
		"action", action,
		"account", account,
		"timestamp", time.Now().UTC(),
		"details", details,
	)
}

// SecurityEvent logs security-related events
func (l *Logger) SecurityEvent(event string, account string, severity string) {
	l.Warn("Security event",
		"event", event,
		"account", account,
		"severity", severity,
		"timestamp", time.Now().UTC(),
	)
}

// PerformanceEvent logs performance metrics
func (l *Logger) PerformanceEvent(operation string, duration time.Duration, success bool) {
	l.Info("Performance metric",
		"operation", operation,
		"duration_ms", duration.Milliseconds(),
		"success", success,
		"timestamp", time.Now().UTC(),
	)
}

// GitHubAPIEvent logs GitHub API interactions
func (l *Logger) GitHubAPIEvent(endpoint string, method string, statusCode int, duration time.Duration) {
	l.Info("GitHub API call",
		"endpoint", endpoint,
		"method", method,
		"status_code", statusCode,
		"duration_ms", duration.Milliseconds(),
		"timestamp", time.Now().UTC(),
	)
}

// ErrorEvent logs errors with context
func (l *Logger) ErrorEvent(err error, operation string, account string) {
	l.Error("Operation failed",
		"error", err.Error(),
		"operation", operation,
		"account", account,
		"timestamp", time.Now().UTC(),
	)
}

// getTraceID extracts trace ID from context (for distributed tracing)
func getTraceID(ctx context.Context) string {
	// In a real implementation, this would extract the trace ID
	// from OpenTelemetry or similar tracing system
	return "trace-123456"
}

// Structured event types for consistent logging
type AccountSwitchEvent struct {
	FromAccount string        `json:"from_account"`
	ToAccount   string        `json:"to_account"`
	Duration    time.Duration `json:"duration"`
	Success     bool          `json:"success"`
}

type SSHKeyEvent struct {
	Account string `json:"account"`
	KeyPath string `json:"key_path"`
	Action  string `json:"action"` // "generated", "uploaded", "validated"
	Success bool   `json:"success"`
}

type GitConfigEvent struct {
	Account string            `json:"account"`
	Changes map[string]string `json:"changes"`
	Global  bool              `json:"global"`
	Success bool              `json:"success"`
}

// LogAccountSwitch logs account switching events
func (l *Logger) LogAccountSwitch(event AccountSwitchEvent) {
	l.AccountEvent("switch", event.ToAccount, map[string]any{
		"from_account": event.FromAccount,
		"duration_ms":  event.Duration.Milliseconds(),
		"success":      event.Success,
	})
}

// LogSSHKey logs SSH key operations
func (l *Logger) LogSSHKey(event SSHKeyEvent) {
	l.SecurityEvent("ssh_key_operation", event.Account, "info")
	l.Info("SSH key event",
		"account", event.Account,
		"key_path", event.KeyPath,
		"action", event.Action,
		"success", event.Success,
	)
}

// LogGitConfig logs Git configuration changes
func (l *Logger) LogGitConfig(event GitConfigEvent) {
	l.AccountEvent("git_config", event.Account, map[string]any{
		"changes": event.Changes,
		"global":  event.Global,
		"success": event.Success,
	})
}
