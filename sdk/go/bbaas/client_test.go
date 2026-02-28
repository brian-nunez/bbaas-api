package bbaas

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestClientSpawn(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method == http.MethodPost && request.URL.Path == "/api/v1/browsers" {
			if request.Header.Get("X-API-Key") != "bbaas_token" {
				return jsonResponse(http.StatusUnauthorized, `{"error":{"error_message":"unauthorized"}}`), nil
			}

			return jsonResponse(http.StatusCreated, `{"browser":{"id":"brw_1"}}`), nil
		}

		return jsonResponse(http.StatusNotFound, `{}`), nil
	})}

	client, err := NewClient("http://bbaas.local", WithHTTPClient(httpClient), WithAPIToken("bbaas_token"))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	spawned, err := client.SpawnBrowser(context.Background(), SpawnBrowserRequest{})
	if err != nil {
		t.Fatalf("spawn browser: %v", err)
	}
	if spawned.Browser.ID != "brw_1" {
		t.Fatalf("expected browser id brw_1, got %q", spawned.Browser.ID)
	}
}

func TestClientRequiresTokenForProtectedEndpoints(t *testing.T) {
	t.Parallel()

	client, err := NewClient("http://localhost:8080")
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = client.SpawnBrowser(context.Background(), SpawnBrowserRequest{})
	if err == nil || err.Error() != "API token is required for this endpoint" {
		t.Fatalf("expected missing token error, got %v", err)
	}
}

type roundTripFunc func(request *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func jsonResponse(statusCode int, payload string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(payload)),
	}
}
