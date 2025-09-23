package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/techishthoughts/GitPersona/internal/observability"
)

// WebhookServer provides real-time synchronization via webhooks
type WebhookServer struct {
	logger     observability.Logger
	server     *http.Server
	handlers   map[string]WebhookHandler
	middleware []MiddlewareFunc
	secret     string
	mutex      sync.RWMutex
	running    bool
}

// WebhookHandler processes webhook events
type WebhookHandler interface {
	HandleEvent(ctx context.Context, event *WebhookEvent) error
	GetEventTypes() []string
}

// MiddlewareFunc represents middleware for webhook processing
type MiddlewareFunc func(next http.HandlerFunc) http.HandlerFunc

// WebhookEvent represents an incoming webhook event
type WebhookEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	Signature string                 `json:"-"`
	Headers   map[string]string      `json:"-"`
}

// GitHubWebhookHandler handles GitHub webhook events
type GitHubWebhookHandler struct {
	logger      observability.Logger
	syncManager *SyncManager
}

// SyncManager manages real-time synchronization
type SyncManager struct {
	logger    observability.Logger
	syncQueue chan *SyncTask
	workers   int
	stopChan  chan struct{}
	waitGroup sync.WaitGroup
}

// SyncTask represents a synchronization task
type SyncTask struct {
	Type      string
	AccountID string
	Data      map[string]interface{}
	Priority  int
	CreatedAt time.Time
}

// NewWebhookServer creates a new webhook server
func NewWebhookServer(logger observability.Logger, addr, secret string) *WebhookServer {
	ws := &WebhookServer{
		logger:   logger,
		handlers: make(map[string]WebhookHandler),
		secret:   secret,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", ws.handleWebhook)
	mux.HandleFunc("/health", ws.handleHealth)

	ws.server = &http.Server{
		Addr:         addr,
		Handler:      ws.applyMiddleware(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Add default middleware
	ws.AddMiddleware(ws.loggingMiddleware)
	ws.AddMiddleware(ws.recoveryMiddleware)
	ws.AddMiddleware(ws.rateLimitMiddleware)

	return ws
}

// Start starts the webhook server
func (ws *WebhookServer) Start(ctx context.Context) error {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	if ws.running {
		return fmt.Errorf("webhook server is already running")
	}

	ws.logger.Info(ctx, "starting_webhook_server",
		observability.F("addr", ws.server.Addr),
	)

	ws.running = true

	go func() {
		if err := ws.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			ws.logger.Error(ctx, "webhook_server_error",
				observability.F("error", err.Error()),
			)
		}
	}()

	ws.logger.Info(ctx, "webhook_server_started")
	return nil
}

// Stop stops the webhook server
func (ws *WebhookServer) Stop(ctx context.Context) error {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	if !ws.running {
		return nil
	}

	ws.logger.Info(ctx, "stopping_webhook_server")

	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := ws.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("failed to shutdown webhook server: %w", err)
	}

	ws.running = false
	ws.logger.Info(ctx, "webhook_server_stopped")
	return nil
}

// AddHandler adds a webhook handler
func (ws *WebhookServer) AddHandler(source string, handler WebhookHandler) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	ws.handlers[source] = handler

	ws.logger.Info(context.Background(), "webhook_handler_added",
		observability.F("source", source),
		observability.F("event_types", handler.GetEventTypes()),
	)
}

// AddMiddleware adds middleware to the webhook server
func (ws *WebhookServer) AddMiddleware(middleware MiddlewareFunc) {
	ws.middleware = append(ws.middleware, middleware)
}

// handleWebhook handles incoming webhook requests
func (ws *WebhookServer) handleWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		ws.logger.Error(ctx, "failed_to_read_webhook_body",
			observability.F("error", err.Error()),
		)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Verify signature if secret is configured
	if ws.secret != "" {
		signature := r.Header.Get("X-Hub-Signature-256")
		if !ws.verifySignature(body, signature) {
			ws.logger.Warn(ctx, "invalid_webhook_signature")
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}
	}

	// Parse webhook event
	event, err := ws.parseWebhookEvent(r, body)
	if err != nil {
		ws.logger.Error(ctx, "failed_to_parse_webhook_event",
			observability.F("error", err.Error()),
		)
		http.Error(w, "Failed to parse webhook event", http.StatusBadRequest)
		return
	}

	// Find appropriate handler
	handler, exists := ws.handlers[event.Source]
	if !exists {
		ws.logger.Warn(ctx, "no_handler_for_webhook_source",
			observability.F("source", event.Source),
		)
		http.Error(w, "No handler for webhook source", http.StatusNotFound)
		return
	}

	// Process event
	if err := handler.HandleEvent(ctx, event); err != nil {
		ws.logger.Error(ctx, "webhook_handler_error",
			observability.F("source", event.Source),
			observability.F("type", event.Type),
			observability.F("error", err.Error()),
		)
		http.Error(w, "Failed to process webhook event", http.StatusInternalServerError)
		return
	}

	ws.logger.Info(ctx, "webhook_event_processed",
		observability.F("source", event.Source),
		observability.F("type", event.Type),
		observability.F("id", event.ID),
	)

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

// handleHealth handles health check requests
func (ws *WebhookServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"handlers":  len(ws.handlers),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(health)
}

// parseWebhookEvent parses an incoming webhook event
func (ws *WebhookServer) parseWebhookEvent(r *http.Request, body []byte) (*WebhookEvent, error) {
	// Determine source from headers or path
	source := ws.determineSource(r)

	event := &WebhookEvent{
		Source:    source,
		Timestamp: time.Now(),
		Headers:   make(map[string]string),
	}

	// Copy relevant headers
	for key, values := range r.Header {
		if len(values) > 0 {
			event.Headers[key] = values[0]
		}
	}

	// Parse based on source
	switch source {
	case "github":
		return ws.parseGitHubEvent(event, r, body)
	case "gitlab":
		return ws.parseGitLabEvent(event, r, body)
	default:
		return ws.parseGenericEvent(event, body)
	}
}

// determineSource determines the webhook source
func (ws *WebhookServer) determineSource(r *http.Request) string {
	// Check User-Agent header
	userAgent := r.Header.Get("User-Agent")
	if strings.Contains(userAgent, "GitHub-Hookshot") {
		return "github"
	}
	if strings.Contains(userAgent, "GitLab") {
		return "gitlab"
	}

	// Check specific headers
	if r.Header.Get("X-GitHub-Event") != "" {
		return "github"
	}
	if r.Header.Get("X-Gitlab-Event") != "" {
		return "gitlab"
	}

	// Default to generic
	return "generic"
}

// parseGitHubEvent parses a GitHub webhook event
func (ws *WebhookServer) parseGitHubEvent(event *WebhookEvent, r *http.Request, body []byte) (*WebhookEvent, error) {
	event.Type = r.Header.Get("X-GitHub-Event")
	event.ID = r.Header.Get("X-GitHub-Delivery")

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub payload: %w", err)
	}

	event.Data = payload
	return event, nil
}

// parseGitLabEvent parses a GitLab webhook event
func (ws *WebhookServer) parseGitLabEvent(event *WebhookEvent, r *http.Request, body []byte) (*WebhookEvent, error) {
	event.Type = r.Header.Get("X-Gitlab-Event")
	event.ID = r.Header.Get("X-Gitlab-Event-UUID")

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse GitLab payload: %w", err)
	}

	event.Data = payload
	return event, nil
}

// parseGenericEvent parses a generic webhook event
func (ws *WebhookServer) parseGenericEvent(event *WebhookEvent, body []byte) (*WebhookEvent, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse generic payload: %w", err)
	}

	// Try to extract type and ID from payload
	if eventType, ok := payload["type"].(string); ok {
		event.Type = eventType
	}
	if id, ok := payload["id"].(string); ok {
		event.ID = id
	}

	event.Data = payload
	return event, nil
}

// verifySignature verifies the webhook signature
func (ws *WebhookServer) verifySignature(body []byte, signature string) bool {
	if signature == "" {
		return false
	}

	// Remove "sha256=" prefix if present
	signature = strings.TrimPrefix(signature, "sha256=")

	// Calculate expected signature
	mac := hmac.New(sha256.New, []byte(ws.secret))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	// Compare signatures
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// applyMiddleware applies all middleware to the handler
func (ws *WebhookServer) applyMiddleware(handler http.Handler) http.Handler {
	h := handler.ServeHTTP

	// Apply middleware in reverse order
	for i := len(ws.middleware) - 1; i >= 0; i-- {
		h = ws.middleware[i](h)
	}

	return http.HandlerFunc(h)
}

// Middleware implementations
func (ws *WebhookServer) loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next(w, r)

		ws.logger.Info(r.Context(), "webhook_request",
			observability.F("method", r.Method),
			observability.F("path", r.URL.Path),
			observability.F("duration", time.Since(start).String()),
			observability.F("user_agent", r.Header.Get("User-Agent")),
		)
	}
}

func (ws *WebhookServer) recoveryMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				ws.logger.Error(r.Context(), "webhook_panic_recovered",
					observability.F("error", fmt.Sprintf("%v", err)),
				)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next(w, r)
	}
}

func (ws *WebhookServer) rateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	// Simple rate limiting - in production, use a more sophisticated implementation
	return func(w http.ResponseWriter, r *http.Request) {
		// Allow request to proceed
		next(w, r)
	}
}

// NewGitHubWebhookHandler creates a new GitHub webhook handler
func NewGitHubWebhookHandler(logger observability.Logger, syncManager *SyncManager) *GitHubWebhookHandler {
	return &GitHubWebhookHandler{
		logger:      logger,
		syncManager: syncManager,
	}
}

// HandleEvent handles GitHub webhook events
func (gh *GitHubWebhookHandler) HandleEvent(ctx context.Context, event *WebhookEvent) error {
	gh.logger.Info(ctx, "handling_github_webhook_event",
		observability.F("type", event.Type),
		observability.F("id", event.ID),
	)

	switch event.Type {
	case "push":
		return gh.handlePushEvent(ctx, event)
	case "repository":
		return gh.handleRepositoryEvent(ctx, event)
	case "member":
		return gh.handleMemberEvent(ctx, event)
	case "organization":
		return gh.handleOrganizationEvent(ctx, event)
	default:
		gh.logger.Debug(ctx, "unhandled_github_event_type",
			observability.F("type", event.Type),
		)
		return nil
	}
}

// GetEventTypes returns the event types this handler supports
func (gh *GitHubWebhookHandler) GetEventTypes() []string {
	return []string{"push", "repository", "member", "organization"}
}

// handlePushEvent handles GitHub push events
func (gh *GitHubWebhookHandler) handlePushEvent(ctx context.Context, event *WebhookEvent) error {
	// Extract repository information
	repository, ok := event.Data["repository"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid repository data in push event")
	}

	repoName, _ := repository["name"].(string)
	repoOwner, _ := repository["owner"].(map[string]interface{})["login"].(string)

	// Create sync task for repository update
	task := &SyncTask{
		Type:      "repository_sync",
		AccountID: repoOwner,
		Data: map[string]interface{}{
			"repository": repoName,
			"action":     "push",
		},
		Priority:  1,
		CreatedAt: time.Now(),
	}

	return gh.syncManager.QueueTask(ctx, task)
}

// handleRepositoryEvent handles GitHub repository events
func (gh *GitHubWebhookHandler) handleRepositoryEvent(ctx context.Context, event *WebhookEvent) error {
	action, _ := event.Data["action"].(string)

	task := &SyncTask{
		Type: "account_sync",
		Data: map[string]interface{}{
			"action": action,
			"type":   "repository",
		},
		Priority:  2,
		CreatedAt: time.Now(),
	}

	return gh.syncManager.QueueTask(ctx, task)
}

// handleMemberEvent handles GitHub member events
func (gh *GitHubWebhookHandler) handleMemberEvent(ctx context.Context, event *WebhookEvent) error {
	// Handle member addition/removal events
	task := &SyncTask{
		Type:      "permissions_sync",
		Data:      event.Data,
		Priority:  3,
		CreatedAt: time.Now(),
	}

	return gh.syncManager.QueueTask(ctx, task)
}

// handleOrganizationEvent handles GitHub organization events
func (gh *GitHubWebhookHandler) handleOrganizationEvent(ctx context.Context, event *WebhookEvent) error {
	// Handle organization-level events
	task := &SyncTask{
		Type:      "organization_sync",
		Data:      event.Data,
		Priority:  2,
		CreatedAt: time.Now(),
	}

	return gh.syncManager.QueueTask(ctx, task)
}

// NewSyncManager creates a new sync manager
func NewSyncManager(logger observability.Logger, workers int) *SyncManager {
	if workers <= 0 {
		workers = 3
	}

	return &SyncManager{
		logger:    logger,
		syncQueue: make(chan *SyncTask, 100),
		workers:   workers,
		stopChan:  make(chan struct{}),
	}
}

// Start starts the sync manager
func (sm *SyncManager) Start(ctx context.Context) error {
	sm.logger.Info(ctx, "starting_sync_manager",
		observability.F("workers", sm.workers),
	)

	// Start worker goroutines
	for i := 0; i < sm.workers; i++ {
		sm.waitGroup.Add(1)
		go sm.worker(ctx, i)
	}

	return nil
}

// Stop stops the sync manager
func (sm *SyncManager) Stop(ctx context.Context) error {
	sm.logger.Info(ctx, "stopping_sync_manager")

	close(sm.stopChan)
	sm.waitGroup.Wait()

	sm.logger.Info(ctx, "sync_manager_stopped")
	return nil
}

// QueueTask queues a synchronization task
func (sm *SyncManager) QueueTask(ctx context.Context, task *SyncTask) error {
	select {
	case sm.syncQueue <- task:
		sm.logger.Debug(ctx, "sync_task_queued",
			observability.F("type", task.Type),
			observability.F("priority", task.Priority),
		)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("sync queue is full")
	}
}

// worker processes sync tasks
func (sm *SyncManager) worker(ctx context.Context, workerID int) {
	defer sm.waitGroup.Done()

	sm.logger.Info(ctx, "sync_worker_started",
		observability.F("worker_id", workerID),
	)

	for {
		select {
		case task := <-sm.syncQueue:
			sm.processTask(ctx, workerID, task)
		case <-sm.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// processTask processes a synchronization task
func (sm *SyncManager) processTask(ctx context.Context, workerID int, task *SyncTask) {
	start := time.Now()

	sm.logger.Info(ctx, "processing_sync_task",
		observability.F("worker_id", workerID),
		observability.F("task_type", task.Type),
		observability.F("account_id", task.AccountID),
	)

	// Process the task based on type
	var err error
	switch task.Type {
	case "repository_sync":
		err = sm.processRepositorySync(ctx, task)
	case "account_sync":
		err = sm.processAccountSync(ctx, task)
	case "permissions_sync":
		err = sm.processPermissionsSync(ctx, task)
	case "organization_sync":
		err = sm.processOrganizationSync(ctx, task)
	default:
		err = fmt.Errorf("unknown task type: %s", task.Type)
	}

	duration := time.Since(start)

	if err != nil {
		sm.logger.Error(ctx, "sync_task_failed",
			observability.F("worker_id", workerID),
			observability.F("task_type", task.Type),
			observability.F("duration", duration.String()),
			observability.F("error", err.Error()),
		)
	} else {
		sm.logger.Info(ctx, "sync_task_completed",
			observability.F("worker_id", workerID),
			observability.F("task_type", task.Type),
			observability.F("duration", duration.String()),
		)
	}
}

// Sync task processors
func (sm *SyncManager) processRepositorySync(ctx context.Context, task *SyncTask) error {
	// Implementation would sync repository-specific data
	sm.logger.Debug(ctx, "processing_repository_sync",
		observability.F("data", task.Data),
	)
	return nil
}

func (sm *SyncManager) processAccountSync(ctx context.Context, task *SyncTask) error {
	// Implementation would sync account-level data
	sm.logger.Debug(ctx, "processing_account_sync",
		observability.F("data", task.Data),
	)
	return nil
}

func (sm *SyncManager) processPermissionsSync(ctx context.Context, task *SyncTask) error {
	// Implementation would sync permission changes
	sm.logger.Debug(ctx, "processing_permissions_sync",
		observability.F("data", task.Data),
	)
	return nil
}

func (sm *SyncManager) processOrganizationSync(ctx context.Context, task *SyncTask) error {
	// Implementation would sync organization-level changes
	sm.logger.Debug(ctx, "processing_organization_sync",
		observability.F("data", task.Data),
	)
	return nil
}
