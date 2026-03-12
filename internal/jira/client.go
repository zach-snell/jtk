package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Client is the Jira Cloud REST API HTTP client.
type Client struct {
	http     *http.Client
	baseURL  string // https://{domain}.atlassian.net/rest/api/3
	agileURL string // https://{domain}.atlassian.net/rest/agile/1.0
	domain   string
	email    string
	token    string // API token for Basic Auth

	rateLimiter *RateLimiter
	mu          sync.Mutex
}

// RateLimiter implements a token bucket rate limiter.
type RateLimiter struct {
	tokens     int
	maxTokens  int
	refillRate time.Duration
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a rate limiter with the specified max tokens and refill rate.
func NewRateLimiter(maxTokens int, refillRate time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is allowed and consumes a token.
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.refill()

	if rl.tokens > 0 {
		rl.tokens--
		return true
	}
	return false
}

func (rl *RateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)
	tokensToAdd := int(elapsed / rl.refillRate)
	if tokensToAdd > 0 {
		rl.tokens += tokensToAdd
		if rl.tokens > rl.maxTokens {
			rl.tokens = rl.maxTokens
		}
		rl.lastRefill = now
	}
}

// NewClient creates a Jira API client with Basic Auth.
func NewClient(domain, email, token string) *Client {
	return &Client{
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:     fmt.Sprintf("https://%s.atlassian.net/rest/api/3", domain),
		agileURL:    fmt.Sprintf("https://%s.atlassian.net/rest/agile/1.0", domain),
		domain:      domain,
		email:       email,
		token:       token,
		rateLimiter: NewRateLimiter(20, 3*time.Second), // 20 req/min ≈ 1 token per 3s
	}
}

// Domain returns the Jira domain this client is configured for.
func (c *Client) Domain() string {
	return c.domain
}

// do executes an HTTP request with auth headers and rate limiting.
func (c *Client) do(method, url string, bodyData []byte, contentType string) (*http.Response, error) {
	if !c.rateLimiter.Allow() {
		return nil, fmt.Errorf("rate limit exceeded: max 20 requests per minute. Please wait and retry")
	}

	var bodyReader io.Reader
	if bodyData != nil {
		bodyReader = bytes.NewReader(bodyData)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.SetBasicAuth(c.email, c.token)

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}

	return resp, nil
}

// Get performs a GET request to the REST API v3 and returns the response body.
func (c *Client) Get(path string) ([]byte, error) {
	resp, err := c.do(http.MethodGet, c.baseURL+path, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, parseAPIError(resp.StatusCode, data)
	}

	return data, nil
}

// GetAgile performs a GET request to the Agile REST API and returns the response body.
func (c *Client) GetAgile(path string) ([]byte, error) {
	resp, err := c.do(http.MethodGet, c.agileURL+path, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, parseAPIError(resp.StatusCode, data)
	}

	return data, nil
}

// Post performs a POST request with a JSON body.
func (c *Client) Post(path string, body interface{}) ([]byte, error) {
	var bodyData []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling body: %w", err)
		}
		bodyData = b
	}

	resp, err := c.do(http.MethodPost, c.baseURL+path, bodyData, "application/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respData))
	}

	return respData, nil
}

// Put performs a PUT request with a JSON body.
func (c *Client) Put(path string, body interface{}) ([]byte, error) {
	var bodyData []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling body: %w", err)
		}
		bodyData = b
	}

	resp, err := c.do(http.MethodPut, c.baseURL+path, bodyData, "application/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respData))
	}

	return respData, nil
}

// Delete performs a DELETE request.
func (c *Client) Delete(path string) error {
	resp, err := c.do(http.MethodDelete, c.baseURL+path, nil, "")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(resp.Body)
		return parseAPIError(resp.StatusCode, data)
	}

	return nil
}

// GetDevStatus performs a GET request to the dev-status API (not under /rest/api/3/).
// The path should start with /rest/dev-status/... and will be appended to the site root.
func (c *Client) GetDevStatus(path string) ([]byte, error) {
	devURL := fmt.Sprintf("https://%s.atlassian.net%s", c.domain, path)
	resp, err := c.do(http.MethodGet, devURL, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, parseAPIError(resp.StatusCode, data)
	}

	return data, nil
}

// GetJSON performs a GET and unmarshals the JSON response into a typed result.
func GetJSON[T any](c *Client, path string) (*T, error) {
	data, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}

	return &result, nil
}

// GetAgileJSON performs a GET on the agile API and unmarshals the response.
func GetAgileJSON[T any](c *Client, path string) (*T, error) {
	data, err := c.GetAgile(path)
	if err != nil {
		return nil, err
	}

	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}

	return &result, nil
}

// PostAgile performs a POST request with a JSON body to the Agile REST API.
func (c *Client) PostAgile(path string, body interface{}) ([]byte, error) {
	var bodyData []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling body: %w", err)
		}
		bodyData = b
	}

	resp, err := c.do(http.MethodPost, c.agileURL+path, bodyData, "application/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respData))
	}

	return respData, nil
}

// GetAbsolute performs a GET request to an absolute URL (not relative to baseURL).
// Useful for downloading attachment content where the URL is already fully qualified.
func (c *Client) GetAbsolute(absoluteURL string) (*http.Response, error) {
	return c.do(http.MethodGet, absoluteURL, nil, "")
}

// PostMultipart performs a POST request with multipart/form-data body.
// Used for uploading attachments to Jira issues.
func (c *Client) PostMultipart(path string, bodyData []byte, contentType string) ([]byte, error) {
	if !c.rateLimiter.Allow() {
		return nil, fmt.Errorf("rate limit exceeded: max 20 requests per minute. Please wait and retry")
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(bodyData))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.SetBasicAuth(c.email, c.token)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Atlassian-Token", "no-check")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respData))
	}

	return respData, nil
}

func parseAPIError(statusCode int, body []byte) error {
	if statusCode == http.StatusForbidden {
		return fmt.Errorf("403 Forbidden: Permission denied. Ensure your Jira API token has the required permissions. Details: %s", string(body))
	}
	if statusCode == http.StatusUnauthorized {
		return fmt.Errorf("401 Unauthorized: Authentication failed. Check your email and API token. Details: %s", string(body))
	}
	return fmt.Errorf("API error %d: %s", statusCode, string(body))
}
