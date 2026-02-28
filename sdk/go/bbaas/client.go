package bbaas

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

type Option func(*Client)

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	apiToken   string
}

type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("API returned status %d", e.StatusCode)
	}

	return fmt.Sprintf("API returned status %d: %s", e.StatusCode, e.Message)
}

func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

func WithAPIToken(apiToken string) Option {
	return func(c *Client) {
		c.apiToken = strings.TrimSpace(apiToken)
	}
}

func NewClient(baseURL string, options ...Option) (*Client, error) {
	trimmedBaseURL := strings.TrimSpace(baseURL)
	if trimmedBaseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}

	parsedURL, err := url.Parse(trimmedBaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, fmt.Errorf("base URL must include scheme and host")
	}

	client := &Client{
		baseURL: parsedURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}

	for _, option := range options {
		option(client)
	}

	return client, nil
}

func (c *Client) SetAPIToken(apiToken string) {
	c.apiToken = strings.TrimSpace(apiToken)
}

func (c *Client) SpawnBrowser(ctx context.Context, request SpawnBrowserRequest) (SpawnBrowserResponse, error) {
	var response SpawnBrowserResponse
	var requestBody any
	if !isEmptySpawnBrowserRequest(request) {
		requestBody = request
	}
	if err := c.do(ctx, http.MethodPost, "/api/v1/browsers", requestBody, true, http.StatusCreated, &response); err != nil {
		return SpawnBrowserResponse{}, err
	}

	return response, nil
}

func isEmptySpawnBrowserRequest(request SpawnBrowserRequest) bool {
	return request.Headless == nil && request.IdleTimeoutSeconds == nil
}

func (c *Client) ListBrowsers(ctx context.Context) ([]Browser, error) {
	var response struct {
		Browsers []Browser `json:"browsers"`
	}

	if err := c.do(ctx, http.MethodGet, "/api/v1/browsers", nil, true, http.StatusOK, &response); err != nil {
		return nil, err
	}

	return response.Browsers, nil
}

func (c *Client) GetBrowser(ctx context.Context, browserID string) (Browser, error) {
	var response struct {
		Browser Browser `json:"browser"`
	}

	if err := c.do(ctx, http.MethodGet, path.Join("/api/v1/browsers", browserID), nil, true, http.StatusOK, &response); err != nil {
		return Browser{}, err
	}

	return response.Browser, nil
}

func (c *Client) KeepAliveBrowser(ctx context.Context, browserID string) (Browser, error) {
	var response struct {
		Browser Browser `json:"browser"`
	}

	if err := c.do(ctx, http.MethodPost, path.Join("/api/v1/browsers", browserID, "keepalive"), nil, true, http.StatusOK, &response); err != nil {
		return Browser{}, err
	}

	return response.Browser, nil
}

func (c *Client) CloseBrowser(ctx context.Context, browserID string) error {
	return c.do(ctx, http.MethodDelete, path.Join("/api/v1/browsers", browserID), nil, true, http.StatusOK, nil)
}

func (c *Client) do(ctx context.Context, method string, resourcePath string, requestBody any, requiresAuth bool, expectedStatus int, output any) error {
	var body io.Reader
	if requestBody != nil {
		payload, err := json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("encode request body: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	requestURL := *c.baseURL
	requestURL.Path = path.Join(c.baseURL.Path, resourcePath)

	httpRequest, err := http.NewRequestWithContext(ctx, method, requestURL.String(), body)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	httpRequest.Header.Set("Accept", "application/json")
	if requestBody != nil {
		httpRequest.Header.Set("Content-Type", "application/json")
	}

	if requiresAuth {
		if strings.TrimSpace(c.apiToken) == "" {
			return fmt.Errorf("API token is required for this endpoint")
		}
		httpRequest.Header.Set("X-API-Key", c.apiToken)
	}

	httpResponse, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return fmt.Errorf("call API: %w", err)
	}
	defer httpResponse.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(httpResponse.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if httpResponse.StatusCode != expectedStatus {
		return parseAPIError(httpResponse.StatusCode, responseBody)
	}

	if output == nil || len(responseBody) == 0 {
		return nil
	}

	if err := json.Unmarshal(responseBody, output); err != nil {
		return fmt.Errorf("decode response body: %w", err)
	}

	return nil
}

func parseAPIError(statusCode int, body []byte) error {
	message := strings.TrimSpace(string(body))
	if message == "" {
		message = http.StatusText(statusCode)
	}

	var structured struct {
		Message string `json:"message"`
		Error   struct {
			ErrorMessage string `json:"error_message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &structured); err == nil {
		if structured.Error.ErrorMessage != "" {
			message = structured.Error.ErrorMessage
		} else if structured.Message != "" {
			message = structured.Message
		}
	}

	return &APIError{
		StatusCode: statusCode,
		Message:    message,
	}
}
