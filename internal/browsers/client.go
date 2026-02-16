package browsers

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

type ManagerClient interface {
	Spawn(ctx context.Context, request SpawnRequest) (SpawnResponse, error)
	List(ctx context.Context) ([]Browser, error)
	Get(ctx context.Context, browserID string) (Browser, error)
	KeepAlive(ctx context.Context, browserID string) (Browser, error)
	Close(ctx context.Context, browserID string) error
}

type UpstreamError struct {
	StatusCode int
	Message    string
}

func (e *UpstreamError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("upstream returned status %d", e.StatusCode)
	}

	return fmt.Sprintf("upstream returned status %d: %s", e.StatusCode, e.Message)
}

type HTTPManagerClient struct {
	baseURL    *url.URL
	httpClient *http.Client
}

func NewHTTPManagerClient(baseURL string, httpClient *http.Client) (*HTTPManagerClient, error) {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		return nil, fmt.Errorf("base URL is required")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("base URL must include scheme and host")
	}

	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}

	return &HTTPManagerClient{
		baseURL:    parsed,
		httpClient: httpClient,
	}, nil
}

func (c *HTTPManagerClient) Spawn(ctx context.Context, request SpawnRequest) (SpawnResponse, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		return SpawnResponse{}, fmt.Errorf("marshal spawn request: %w", err)
	}

	httpRequest, err := c.newRequest(ctx, http.MethodPost, "/api/v1/browsers", bytes.NewReader(payload))
	if err != nil {
		return SpawnResponse{}, err
	}
	httpRequest.Header.Set("Content-Type", "application/json")

	var response SpawnResponse
	if err := c.do(httpRequest, http.StatusCreated, &response); err != nil {
		return SpawnResponse{}, err
	}

	return response, nil
}

func (c *HTTPManagerClient) List(ctx context.Context) ([]Browser, error) {
	httpRequest, err := c.newRequest(ctx, http.MethodGet, "/api/v1/browsers", nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRaw(httpRequest, http.StatusOK)
	if err != nil {
		return nil, err
	}

	browsers, err := decodeBrowserList(body)
	if err != nil {
		return nil, err
	}

	return browsers, nil
}

func (c *HTTPManagerClient) Get(ctx context.Context, browserID string) (Browser, error) {
	httpRequest, err := c.newRequest(ctx, http.MethodGet, path.Join("/api/v1/browsers", browserID), nil)
	if err != nil {
		return Browser{}, err
	}

	body, err := c.doRaw(httpRequest, http.StatusOK)
	if err != nil {
		return Browser{}, err
	}

	browser, err := decodeBrowser(body)
	if err != nil {
		return Browser{}, err
	}

	return browser, nil
}

func (c *HTTPManagerClient) KeepAlive(ctx context.Context, browserID string) (Browser, error) {
	httpRequest, err := c.newRequest(ctx, http.MethodPost, path.Join("/api/v1/browsers", browserID, "keepalive"), nil)
	if err != nil {
		return Browser{}, err
	}

	body, err := c.doRaw(httpRequest, http.StatusOK)
	if err != nil {
		return Browser{}, err
	}

	browser, err := decodeBrowser(body)
	if err != nil {
		return Browser{}, err
	}

	return browser, nil
}

func (c *HTTPManagerClient) Close(ctx context.Context, browserID string) error {
	httpRequest, err := c.newRequest(ctx, http.MethodDelete, path.Join("/api/v1/browsers", browserID), nil)
	if err != nil {
		return err
	}

	_, err = c.doRaw(httpRequest, http.StatusNoContent)
	if err != nil {
		return err
	}

	return nil
}

func (c *HTTPManagerClient) newRequest(ctx context.Context, method string, resourcePath string, body io.Reader) (*http.Request, error) {
	requestURL := *c.baseURL
	requestURL.Path = path.Join(c.baseURL.Path, resourcePath)

	httpRequest, err := http.NewRequestWithContext(ctx, method, requestURL.String(), body)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	return httpRequest, nil
}

func (c *HTTPManagerClient) do(request *http.Request, expectedStatus int, output any) error {
	body, err := c.doRaw(request, expectedStatus)
	if err != nil {
		return err
	}

	if len(body) == 0 || output == nil {
		return nil
	}

	if err := json.Unmarshal(body, output); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}

func (c *HTTPManagerClient) doRaw(request *http.Request, expectedStatus int) ([]byte, error) {
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("call CDP manager API: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if response.StatusCode != expectedStatus {
		return nil, parseUpstreamError(response.StatusCode, body)
	}

	return body, nil
}

func parseUpstreamError(statusCode int, body []byte) error {
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

	return &UpstreamError{
		StatusCode: statusCode,
		Message:    message,
	}
}

func decodeBrowserList(body []byte) ([]Browser, error) {
	var browsers []Browser
	if err := json.Unmarshal(body, &browsers); err == nil {
		return browsers, nil
	}

	var wrapped struct {
		Browsers []Browser `json:"browsers"`
	}
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, fmt.Errorf("decode list response: %w", err)
	}

	return wrapped.Browsers, nil
}

func decodeBrowser(body []byte) (Browser, error) {
	var browser Browser
	if err := json.Unmarshal(body, &browser); err == nil && browser.ID != "" {
		return browser, nil
	}

	var wrapped struct {
		Browser Browser `json:"browser"`
	}
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return Browser{}, fmt.Errorf("decode browser response: %w", err)
	}

	if wrapped.Browser.ID == "" {
		return Browser{}, fmt.Errorf("decode browser response: browser.id is required")
	}

	return wrapped.Browser, nil
}
