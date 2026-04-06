package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	DefaultBaseURL = "https://www.mdpush.io"
	userAgent      = "mdpush-cli/0.1"
)

// Client is the HTTP client for the mdpush API.
type Client struct {
	BaseURL    string
	Token      string // Bearer token for authenticated requests
	HTTPClient *http.Client
}

// NewClient creates a new API client with the default base URL.
// Override with MDPUSH_API_URL env var for development.
func NewClient() *Client {
	base := DefaultBaseURL
	if env := os.Getenv("MDPUSH_API_URL"); env != "" {
		base = env
	}
	return &Client{
		BaseURL: base,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// WithToken returns a copy of the client with the given auth token.
func (c *Client) WithToken(token string) *Client {
	return &Client{
		BaseURL:    c.BaseURL,
		Token:      token,
		HTTPClient: c.HTTPClient,
	}
}

// APIError represents a structured error from the API.
type APIError struct {
	StatusCode        int    `json:"-"`
	ErrorCode         string `json:"error"`
	Message           string `json:"message,omitempty"`
	LockType          string `json:"lockType,omitempty"`
	RetryAfterSeconds int    `json:"retryAfterSeconds,omitempty"`
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s: %s (HTTP %d)", e.ErrorCode, e.Message, e.StatusCode)
	}
	return fmt.Sprintf("%s (HTTP %d)", e.ErrorCode, e.StatusCode)
}

// IsUnauthorized returns true if the error is a 401.
func (e *APIError) IsUnauthorized() bool {
	return e.StatusCode == 401
}

// IsNotFound returns true if the error is a 404.
func (e *APIError) IsNotFound() bool {
	return e.StatusCode == 404
}

// IsRateLimited returns true if the error is a 429.
func (e *APIError) IsRateLimited() bool {
	return e.StatusCode == 429
}

// doJSON performs a JSON request and decodes the response into result.
// If the response is not 2xx, it returns an *APIError.
func (c *Client) doJSON(method, path string, body any, result any) error {
	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBytes)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		// Try to parse as JSON error
		if json.Unmarshal(respBody, apiErr) != nil {
			apiErr.ErrorCode = "unknown"
			apiErr.Message = string(respBody)
		}
		if apiErr.ErrorCode == "" {
			apiErr.ErrorCode = "unknown"
		}
		return apiErr
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}

	return nil
}

// doRaw performs a request and returns the raw response (for /raw endpoint).
func (c *Client) doRaw(method, path string, headers map[string]string) ([]byte, http.Header, error) {
	req, err := http.NewRequest(method, c.BaseURL+path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		if json.Unmarshal(body, apiErr) != nil {
			apiErr.ErrorCode = "unknown"
			apiErr.Message = string(body)
		}
		if apiErr.ErrorCode == "" {
			apiErr.ErrorCode = "unknown"
		}
		return nil, nil, apiErr
	}

	return body, resp.Header, nil
}
