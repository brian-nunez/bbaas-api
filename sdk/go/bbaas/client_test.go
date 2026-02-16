package bbaas

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientRegisterAndSpawn(t *testing.T) {
	t.Parallel()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/applications":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(RegisterApplicationResponse{
				Application: Application{ID: "app_1", Name: "app"},
				APIToken:    "bbaas_token",
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/browsers":
			if r.Header.Get("Authorization") != "Bearer bbaas_token" {
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]string{"error_message": "unauthorized"},
				})
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(SpawnBrowserResponse{Browser: Browser{ID: "brw_1"}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer apiServer.Close()

	client, err := NewClient(apiServer.URL)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	registered, err := client.RegisterApplication(context.Background(), RegisterApplicationRequest{
		Name:              "app",
		GitHubProfileLink: "https://github.com/example",
	})
	if err != nil {
		t.Fatalf("register app: %v", err)
	}

	client.SetAPIToken(registered.APIToken)
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
