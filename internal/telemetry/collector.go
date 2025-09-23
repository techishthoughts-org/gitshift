package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/techishthoughts/GitPersona/internal/observability"
)

// TelemetryCollector collects and manages telemetry data
type TelemetryCollector struct {
	logger      observability.Logger
	enabled     bool
	anonymized  bool
	dataDir     string
	sessionID   string
	startTime   time.Time
	events      []TelemetryEvent
	metrics     map[string]*Metric
	mutex       sync.RWMutex
	flushTicker *time.Ticker
	stopChan    chan struct{}
}

// TelemetryEvent represents a telemetry event
type TelemetryEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Category  string                 `json:"category"`
	Action    string                 `json:"action"`
	Label     string                 `json:"label"`
	Value     float64                `json:"value"`
	Timestamp time.Time              `json:"timestamp"`
	SessionID string                 `json:"session_id"`
	UserID    string                 `json:"user_id,omitempty"`
	Metadata  map[string]interface{} `json:"metadata"`
	Duration  time.Duration          `json:"duration,omitempty"`
}

// Metric represents a performance or usage metric
type Metric struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"` // counter, gauge, histogram
	Value       float64           `json:"value"`
	Count       int64             `json:"count"`
	Sum         float64           `json:"sum"`
	Min         float64           `json:"min"`
	Max         float64           `json:"max"`
	LastUpdated time.Time         `json:"last_updated"`
	Tags        map[string]string `json:"tags"`
}

// SystemInfo contains system information for telemetry
type SystemInfo struct {
	OS           string `json:"os"`
	Arch         string `json:"arch"`
	GoVersion    string `json:"go_version"`
	GitPersona   string `json:"gitpersona_version"`
	GitVersion   string `json:"git_version"`
	ShellType    string `json:"shell_type"`
	TerminalType string `json:"terminal_type"`
	Locale       string `json:"locale"`
}

// UsageStats contains usage statistics
type UsageStats struct {
	SessionDuration     time.Duration      `json:"session_duration"`
	CommandsExecuted    int64              `json:"commands_executed"`
	AccountSwitches     int64              `json:"account_switches"`
	ErrorsEncountered   int64              `json:"errors_encountered"`
	FeaturesUsed        []string           `json:"features_used"`
	PerformanceMetrics  map[string]float64 `json:"performance_metrics"`
	MostUsedCommands    map[string]int64   `json:"most_used_commands"`
	AverageResponseTime time.Duration      `json:"average_response_time"`
	SystemHealth        float64            `json:"system_health"`
}

// NewTelemetryCollector creates a new telemetry collector
func NewTelemetryCollector(logger observability.Logger, dataDir string, enabled, anonymized bool) *TelemetryCollector {
	if dataDir == "" {
		homeDir, _ := os.UserHomeDir()
		dataDir = filepath.Join(homeDir, ".config", "gitpersona", "telemetry")
	}

	// Ensure data directory exists
	_ = os.MkdirAll(dataDir, 0755)

	sessionID := fmt.Sprintf("session-%d", time.Now().UnixNano())

	tc := &TelemetryCollector{
		logger:     logger,
		enabled:    enabled,
		anonymized: anonymized,
		dataDir:    dataDir,
		sessionID:  sessionID,
		startTime:  time.Now(),
		events:     make([]TelemetryEvent, 0),
		metrics:    make(map[string]*Metric),
		stopChan:   make(chan struct{}),
	}

	if enabled {
		// Start background flush routine
		tc.flushTicker = time.NewTicker(5 * time.Minute)
		go tc.flushRoutine()

		// Record session start
		tc.TrackEvent("session", "start", "user_session", "", 0, nil)
	}

	return tc
}

// Start starts the telemetry collector
func (tc *TelemetryCollector) Start(ctx context.Context) error {
	if !tc.enabled {
		return nil
	}

	tc.logger.Info(ctx, "starting_telemetry_collector",
		observability.F("session_id", tc.sessionID),
		observability.F("anonymized", tc.anonymized),
	)

	// Collect system information
	systemInfo := tc.collectSystemInfo()
	tc.TrackEvent("system", "info", "system_information", "", 0, map[string]interface{}{
		"system_info": systemInfo,
	})

	return nil
}

// Stop stops the telemetry collector
func (tc *TelemetryCollector) Stop(ctx context.Context) error {
	if !tc.enabled {
		return nil
	}

	tc.logger.Info(ctx, "stopping_telemetry_collector")

	// Record session end
	sessionDuration := time.Since(tc.startTime)
	tc.TrackEvent("session", "end", "user_session", "", float64(sessionDuration.Seconds()), map[string]interface{}{
		"duration_seconds": sessionDuration.Seconds(),
	})

	// Stop flush routine
	if tc.flushTicker != nil {
		tc.flushTicker.Stop()
	}
	close(tc.stopChan)

	// Final flush
	tc.flush(ctx)

	return nil
}

// TrackEvent tracks a telemetry event
func (tc *TelemetryCollector) TrackEvent(category, action, eventType, label string, value float64, metadata map[string]interface{}) {
	if !tc.enabled {
		return
	}

	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	event := TelemetryEvent{
		ID:        fmt.Sprintf("event-%d", time.Now().UnixNano()),
		Type:      eventType,
		Category:  category,
		Action:    action,
		Label:     label,
		Value:     value,
		Timestamp: time.Now(),
		SessionID: tc.sessionID,
		Metadata:  metadata,
	}

	// Add user ID if not anonymized
	if !tc.anonymized {
		event.UserID = tc.getUserID()
	}

	tc.events = append(tc.events, event)

	tc.logger.Debug(context.Background(), "telemetry_event_tracked",
		observability.F("category", category),
		observability.F("action", action),
		observability.F("type", eventType),
	)
}

// TrackCommand tracks command execution
func (tc *TelemetryCollector) TrackCommand(command string, duration time.Duration, success bool, errorType string) {
	metadata := map[string]interface{}{
		"duration_ms": duration.Milliseconds(),
		"success":     success,
	}

	if errorType != "" {
		metadata["error_type"] = errorType
	}

	tc.TrackEvent("command", "execute", "command_execution", command, float64(duration.Milliseconds()), metadata)

	// Update command usage metric
	tc.IncrementMetric(fmt.Sprintf("command.%s.count", command), 1.0, nil)
	tc.UpdateMetric(fmt.Sprintf("command.%s.duration", command), float64(duration.Milliseconds()), map[string]string{
		"unit": "milliseconds",
	})
}

// TrackAccountSwitch tracks account switching operations
func (tc *TelemetryCollector) TrackAccountSwitch(fromAccount, toAccount string, duration time.Duration, success bool, errorType string) {
	metadata := map[string]interface{}{
		"duration_ms": duration.Milliseconds(),
		"success":     success,
	}

	if !tc.anonymized {
		metadata["from_account"] = fromAccount
		metadata["to_account"] = toAccount
	}

	if errorType != "" {
		metadata["error_type"] = errorType
	}

	tc.TrackEvent("account", "switch", "account_switch", "", float64(duration.Milliseconds()), metadata)

	// Update metrics
	tc.IncrementMetric("account.switches.total", 1.0, nil)
	if success {
		tc.IncrementMetric("account.switches.success", 1.0, nil)
	} else {
		tc.IncrementMetric("account.switches.failed", 1.0, nil)
	}
}

// TrackError tracks error occurrences
func (tc *TelemetryCollector) TrackError(errorType, errorMessage, context string, severity string) {
	metadata := map[string]interface{}{
		"error_type": errorType,
		"context":    context,
		"severity":   severity,
	}

	if !tc.anonymized {
		metadata["error_message"] = errorMessage
	}

	tc.TrackEvent("error", "occurred", "error_event", errorType, 0, metadata)

	// Update error metrics
	tc.IncrementMetric("errors.total", 1.0, nil)
	tc.IncrementMetric(fmt.Sprintf("errors.%s", errorType), 1.0, map[string]string{
		"severity": severity,
	})
}

// TrackPerformance tracks performance metrics
func (tc *TelemetryCollector) TrackPerformance(operation string, duration time.Duration, metadata map[string]interface{}) {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["duration_ms"] = duration.Milliseconds()

	tc.TrackEvent("performance", "measure", "performance_metric", operation, float64(duration.Milliseconds()), metadata)

	// Update performance metrics
	tc.UpdateMetric(fmt.Sprintf("performance.%s.duration", operation), float64(duration.Milliseconds()), map[string]string{
		"unit": "milliseconds",
	})
}

// IncrementMetric increments a counter metric
func (tc *TelemetryCollector) IncrementMetric(name string, value float64, tags map[string]string) {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	metric, exists := tc.metrics[name]
	if !exists {
		metric = &Metric{
			Name:        name,
			Type:        "counter",
			Value:       0,
			Count:       0,
			Sum:         0,
			Min:         value,
			Max:         value,
			LastUpdated: time.Now(),
			Tags:        tags,
		}
		tc.metrics[name] = metric
	}

	metric.Value += value
	metric.Count++
	metric.Sum += value
	if value < metric.Min {
		metric.Min = value
	}
	if value > metric.Max {
		metric.Max = value
	}
	metric.LastUpdated = time.Now()
}

// UpdateMetric updates a gauge metric
func (tc *TelemetryCollector) UpdateMetric(name string, value float64, tags map[string]string) {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	metric, exists := tc.metrics[name]
	if !exists {
		metric = &Metric{
			Name:        name,
			Type:        "gauge",
			Count:       0,
			Sum:         0,
			Min:         value,
			Max:         value,
			LastUpdated: time.Now(),
			Tags:        tags,
		}
		tc.metrics[name] = metric
	}

	metric.Value = value
	metric.Count++
	metric.Sum += value
	if value < metric.Min {
		metric.Min = value
	}
	if value > metric.Max {
		metric.Max = value
	}
	metric.LastUpdated = time.Now()
}

// GetUsageStats returns current usage statistics
func (tc *TelemetryCollector) GetUsageStats() *UsageStats {
	tc.mutex.RLock()
	defer tc.mutex.RUnlock()

	stats := &UsageStats{
		SessionDuration:    time.Since(tc.startTime),
		FeaturesUsed:       make([]string, 0),
		PerformanceMetrics: make(map[string]float64),
		MostUsedCommands:   make(map[string]int64),
	}

	// Extract statistics from metrics
	for name, metric := range tc.metrics {
		switch {
		case name == "commands.total":
			stats.CommandsExecuted = int64(metric.Value)
		case name == "account.switches.total":
			stats.AccountSwitches = int64(metric.Value)
		case name == "errors.total":
			stats.ErrorsEncountered = int64(metric.Value)
		case strings.HasPrefix(name, "command.") && strings.HasSuffix(name, ".count"):
			commandName := strings.TrimSuffix(strings.TrimPrefix(name, "command."), ".count")
			stats.MostUsedCommands[commandName] = int64(metric.Value)
		case strings.HasPrefix(name, "performance."):
			stats.PerformanceMetrics[name] = metric.Value
		}
	}

	// Calculate average response time
	if totalCommands := stats.CommandsExecuted; totalCommands > 0 {
		totalDuration := float64(0)
		count := int64(0)
		for name, metric := range tc.metrics {
			if strings.HasPrefix(name, "command.") && strings.HasSuffix(name, ".duration") {
				totalDuration += metric.Sum
				count += metric.Count
			}
		}
		if count > 0 {
			stats.AverageResponseTime = time.Duration(totalDuration/float64(count)) * time.Millisecond
		}
	}

	return stats
}

// collectSystemInfo collects system information
func (tc *TelemetryCollector) collectSystemInfo() *SystemInfo {
	info := &SystemInfo{
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		GoVersion:  runtime.Version(),
		GitPersona: "1.0.0", // This would be read from version info
	}

	// Get Git version
	if gitVersion := tc.getGitVersion(); gitVersion != "" {
		info.GitVersion = gitVersion
	}

	// Get shell type
	if shell := os.Getenv("SHELL"); shell != "" {
		info.ShellType = filepath.Base(shell)
	}

	// Get terminal type
	if term := os.Getenv("TERM"); term != "" {
		info.TerminalType = term
	}

	// Get locale
	if locale := os.Getenv("LANG"); locale != "" {
		info.Locale = locale
	}

	return info
}

// getGitVersion gets the Git version
func (tc *TelemetryCollector) getGitVersion() string {
	// Implementation would execute `git --version` and parse output
	return "2.39.0" // Placeholder
}

// getUserID gets a user identifier (anonymized if needed)
func (tc *TelemetryCollector) getUserID() string {
	if tc.anonymized {
		return ""
	}

	// Implementation would generate or retrieve a stable user ID
	return "user-12345" // Placeholder
}

// flushRoutine runs the background flush routine
func (tc *TelemetryCollector) flushRoutine() {
	for {
		select {
		case <-tc.flushTicker.C:
			tc.flush(context.Background())
		case <-tc.stopChan:
			return
		}
	}
}

// flush saves telemetry data to disk
func (tc *TelemetryCollector) flush(ctx context.Context) {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	if len(tc.events) == 0 {
		return
	}

	tc.logger.Debug(ctx, "flushing_telemetry_data",
		observability.F("event_count", len(tc.events)),
		observability.F("metric_count", len(tc.metrics)),
	)

	// Create telemetry data structure
	telemetryData := map[string]interface{}{
		"session_id": tc.sessionID,
		"timestamp":  time.Now(),
		"events":     tc.events,
		"metrics":    tc.metrics,
		"system":     tc.collectSystemInfo(),
		"stats":      tc.GetUsageStats(),
	}

	// Save to file
	filename := fmt.Sprintf("telemetry-%s.json", time.Now().Format("20060102-150405"))
	filePath := filepath.Join(tc.dataDir, filename)

	data, err := json.MarshalIndent(telemetryData, "", "  ")
	if err != nil {
		tc.logger.Error(ctx, "failed_to_marshal_telemetry_data",
			observability.F("error", err.Error()),
		)
		return
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		tc.logger.Error(ctx, "failed_to_write_telemetry_file",
			observability.F("file", filePath),
			observability.F("error", err.Error()),
		)
		return
	}

	// Clear events after successful flush
	tc.events = make([]TelemetryEvent, 0)

	tc.logger.Debug(ctx, "telemetry_data_flushed",
		observability.F("file", filePath),
	)
}

// ExportData exports telemetry data for analysis
func (tc *TelemetryCollector) ExportData(ctx context.Context, outputPath string) error {
	tc.mutex.RLock()
	defer tc.mutex.RUnlock()

	tc.logger.Info(ctx, "exporting_telemetry_data",
		observability.F("output_path", outputPath),
	)

	exportData := map[string]interface{}{
		"session_id":  tc.sessionID,
		"export_time": time.Now(),
		"events":      tc.events,
		"metrics":     tc.metrics,
		"system_info": tc.collectSystemInfo(),
		"usage_stats": tc.GetUsageStats(),
		"anonymized":  tc.anonymized,
	}

	data, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal export data: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}

	tc.logger.Info(ctx, "telemetry_data_exported",
		observability.F("output_path", outputPath),
	)

	return nil
}

// ClearData clears all telemetry data
func (tc *TelemetryCollector) ClearData(ctx context.Context) error {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	tc.logger.Info(ctx, "clearing_telemetry_data")

	tc.events = make([]TelemetryEvent, 0)
	tc.metrics = make(map[string]*Metric)

	// Remove telemetry files
	files, err := os.ReadDir(tc.dataDir)
	if err != nil {
		return fmt.Errorf("failed to read telemetry directory: %w", err)
	}

	for _, file := range files {
		if strings.HasPrefix(file.Name(), "telemetry-") && strings.HasSuffix(file.Name(), ".json") {
			filePath := filepath.Join(tc.dataDir, file.Name())
			if err := os.Remove(filePath); err != nil {
				tc.logger.Warn(ctx, "failed_to_remove_telemetry_file",
					observability.F("file", filePath),
					observability.F("error", err.Error()),
				)
			}
		}
	}

	tc.logger.Info(ctx, "telemetry_data_cleared")
	return nil
}

// SetEnabled enables or disables telemetry collection
func (tc *TelemetryCollector) SetEnabled(enabled bool) {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	tc.enabled = enabled

	if enabled && tc.flushTicker == nil {
		tc.flushTicker = time.NewTicker(5 * time.Minute)
		go tc.flushRoutine()
	} else if !enabled && tc.flushTicker != nil {
		tc.flushTicker.Stop()
		tc.flushTicker = nil
	}
}

// IsEnabled returns whether telemetry is enabled
func (tc *TelemetryCollector) IsEnabled() bool {
	tc.mutex.RLock()
	defer tc.mutex.RUnlock()
	return tc.enabled
}
