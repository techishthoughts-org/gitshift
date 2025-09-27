// Package gh provides GitHub API and authentication functionality for gitshift.
package gh

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	ghapi "github.com/cli/go-gh/v2/pkg/api"
)

// Client wraps the GitHub API client with additional functionality.
type Client struct {
	REST      *ghapi.RESTClient
	logger    *slog.Logger
	mu        sync.Mutex
	rateLimit struct {
		Limit     int
		Remaining int
		Reset     time.Time
	}
}

// ClientOption is a function that configures a Client.
type ClientOption func(*Client)

// WithLogger sets the logger for the client.
func WithLogger(logger *slog.Logger) ClientOption {
	return func(c *Client) {
		c.logger = logger
	}
}

// NewClient creates a new GitHub API client.
func NewClient(opts ...ClientOption) (*Client, error) {
	restClient, err := ghapi.DefaultRESTClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub REST client: %w", err)
	}

	client := &Client{
		REST:   restClient,
		logger: slog.Default(),
	}

	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

// WithToken creates a new client with the specified token.
func WithToken(token string, opts ...ClientOption) (*Client, error) {
	client, err := ghapi.NewRESTClient(ghapi.ClientOptions{
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("token %s", token),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create authenticated client: %w", err)
	}

	c := &Client{
		REST:   client,
		logger: slog.Default(),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// CheckRateLimit checks the current rate limit status.
func (c *Client) CheckRateLimit() (*RateLimit, error) {
	var rateLimit struct {
		Resources struct {
			Core struct {
				Limit     int `json:"limit"`
				Remaining int `json:"remaining"`
				Reset     int `json:"reset"`
			} `json:"core"`
		} `json:"resources"`
	}

	err := c.REST.Get("rate_limit", &rateLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to get rate limit: %w", err)
	}

	return &RateLimit{
		Limit:     rateLimit.Resources.Core.Limit,
		Remaining: rateLimit.Resources.Core.Remaining,
		Reset:     rateLimit.Resources.Core.Reset,
	}, nil
}

// RateLimit represents GitHub API rate limit information.
type RateLimit struct {
	Limit     int
	Remaining int
	Reset     int
}

// IsAuthenticated checks if the client is properly authenticated.
func (c *Client) IsAuthenticated() (bool, error) {
	// Try to access a protected endpoint
	var user struct {
		Login string `json:"login"`
	}

	err := c.REST.Get("user", &user)
	if err != nil {
		if apiErr, ok := err.(*ghapi.HTTPError); ok && apiErr.StatusCode == http.StatusUnauthorized {
			return false, nil
		}
	}

	return user.Login != "", nil
}

// GetAuthenticatedUser gets the username of the currently authenticated user.
func (c *Client) GetAuthenticatedUser(ctx context.Context) (string, error) {
	var user struct {
		Login string `json:"login"`
	}

	if err := c.doWithRetry(ctx, "GET", "user", nil, &user); err != nil {
		return "", fmt.Errorf("failed to get authenticated user: %w", err)
	}

	if user.Login == "" {
		return "", fmt.Errorf("no authenticated user found")
	}

	return user.Login, nil
}

// doWithRetry executes a request with retry logic and rate limiting.
func (c *Client) doWithRetry(ctx context.Context, method, path string, body, result interface{}) error {
	const maxRetries = 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Second * time.Duration(1<<uint(attempt-1))): // Exponential backoff
			}
		}

		// Check rate limits before making the request
		if err := c.checkRateLimit(ctx); err != nil {
			c.logger.WarnContext(ctx, "Rate limit check failed", "error", err)
		}

		// Make the request
		var err error
		switch method {
		case "GET":
			err = c.REST.Get(path, result)
		case "POST", "PUT":
			// Convert body to JSON if it's not nil
			var bodyReader io.Reader
			if body != nil {
				jsonBody, jsonErr := json.Marshal(body)
				if jsonErr != nil {
					return fmt.Errorf("failed to marshal request body: %w", jsonErr)
				}
				bodyReader = bytes.NewReader(jsonBody)
			}
			err = c.REST.DoWithContext(ctx, method, path, bodyReader, result)
		case "DELETE":
			err = c.REST.Delete(path, result)
		default:
			err = fmt.Errorf("unsupported HTTP method: %s", method)
		}

		if err == nil {
			return nil
		}

		// Check if error is retryable
		if !isRetryableError(err) {
			return err
		}

		lastErr = err
		c.logger.WarnContext(ctx, "Request failed, retrying...",
			"attempt", attempt+1,
			"error", err,
			"path", path,
		)
	}

	return fmt.Errorf("after %d attempts, last error: %w", maxRetries, lastErr)
}

// checkRateLimit checks the current rate limit status and sleeps if needed.
func (c *Client) checkRateLimit(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	if now.Before(c.rateLimit.Reset) && c.rateLimit.Remaining <= 0 {
		sleepTime := time.Until(c.rateLimit.Reset)
		c.logger.InfoContext(ctx, "Rate limit reached, sleeping until reset",
			"reset_in", sleepTime,
		)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(sleepTime):
			// Continue after rate limit reset
		}
	}

	// Update rate limit status from headers if available
	// Note: This is a simplified example. In a real implementation,
	// you would parse the rate limit headers from the response.
	return nil
}

// isRetryableError checks if an error is retryable.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for network errors or rate limiting
	if errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, context.Canceled) ||
		strings.Contains(err.Error(), "rate limit") ||
		strings.Contains(err.Error(), "connection reset") ||
		strings.Contains(err.Error(), "timeout") {
		return true
	}

	// Check for HTTP status codes that are safe to retry
	if apiErr, ok := err.(interface{ StatusCode() int }); ok {
		statusCode := apiErr.StatusCode()
		return statusCode >= 500 || statusCode == 429 // 5xx or Too Many Requests
	}

	return false
}

// VerifySSHKey verifies if an SSH key is added to the authenticated user's account.
func (c *Client) VerifySSHKey(ctx context.Context, publicKey string) (bool, error) {
	var keys []struct {
		Key string `json:"key"`
	}

	if err := c.doWithRetry(ctx, "GET", "user/keys", nil, &keys); err != nil {
		return false, fmt.Errorf("failed to get SSH keys: %w", err)
	}

	// Normalize the public key for comparison
	trimmedPublicKey := strings.TrimSpace(publicKey)
	for _, key := range keys {
		if strings.TrimSpace(key.Key) == trimmedPublicKey {
			return true, nil
		}
	}

	return false, nil
}
