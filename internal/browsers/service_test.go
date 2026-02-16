package browsers

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeManagerClient struct {
	browsers map[string]Browser
}

func newFakeManagerClient() *fakeManagerClient {
	return &fakeManagerClient{
		browsers: make(map[string]Browser),
	}
}

func (f *fakeManagerClient) Spawn(_ context.Context, _ SpawnRequest) (SpawnResponse, error) {
	id := "brw_test"
	browser := Browser{
		ID:                 id,
		CDPURL:             "ws://localhost:1234/devtools/browser/test",
		CDPHTTPURL:         "http://localhost:1234",
		CreatedAt:          time.Now(),
		LastActiveAt:       time.Now(),
		IdleTimeoutSeconds: 60,
		ExpiresAt:          time.Now().Add(time.Minute),
	}
	f.browsers[id] = browser
	return SpawnResponse{Browser: browser}, nil
}

func (f *fakeManagerClient) List(_ context.Context) ([]Browser, error) {
	list := make([]Browser, 0, len(f.browsers))
	for _, browser := range f.browsers {
		list = append(list, browser)
	}
	return list, nil
}

func (f *fakeManagerClient) Get(_ context.Context, browserID string) (Browser, error) {
	browser, ok := f.browsers[browserID]
	if !ok {
		return Browser{}, &UpstreamError{StatusCode: 404, Message: "not found"}
	}
	return browser, nil
}

func (f *fakeManagerClient) KeepAlive(_ context.Context, browserID string) (Browser, error) {
	browser, ok := f.browsers[browserID]
	if !ok {
		return Browser{}, &UpstreamError{StatusCode: 404, Message: "not found"}
	}
	browser.LastActiveAt = time.Now()
	f.browsers[browserID] = browser
	return browser, nil
}

func (f *fakeManagerClient) Close(_ context.Context, browserID string) error {
	if _, ok := f.browsers[browserID]; !ok {
		return &UpstreamError{StatusCode: 404, Message: "not found"}
	}
	delete(f.browsers, browserID)
	return nil
}

func TestServiceOwnership(t *testing.T) {
	t.Parallel()

	client := newFakeManagerClient()
	service := NewService(client, NewInMemoryOwnershipStore())

	spawned, err := service.SpawnForApplication(context.Background(), "app_1", SpawnRequest{})
	if err != nil {
		t.Fatalf("expected spawn success, got error: %v", err)
	}

	_, err = service.GetForApplication(context.Background(), "app_2", spawned.Browser.ID)
	if !errors.Is(err, ErrBrowserNotFound) {
		t.Fatalf("expected ErrBrowserNotFound for non-owner access, got %v", err)
	}

	_, err = service.GetForApplication(context.Background(), "app_1", spawned.Browser.ID)
	if err != nil {
		t.Fatalf("expected owner access, got error: %v", err)
	}
}

func TestListForApplicationFiltersByOwnership(t *testing.T) {
	t.Parallel()

	client := newFakeManagerClient()
	service := NewService(client, NewInMemoryOwnershipStore())

	_, err := service.SpawnForApplication(context.Background(), "app_1", SpawnRequest{})
	if err != nil {
		t.Fatalf("spawn app_1: %v", err)
	}

	client.browsers["brw_external"] = Browser{ID: "brw_external"}

	list, err := service.ListForApplication(context.Background(), "app_1")
	if err != nil {
		t.Fatalf("list app_1: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 owned browser, got %d", len(list))
	}
}

func TestCloseRemovesOwnership(t *testing.T) {
	t.Parallel()

	client := newFakeManagerClient()
	service := NewService(client, NewInMemoryOwnershipStore())

	spawned, err := service.SpawnForApplication(context.Background(), "app_1", SpawnRequest{})
	if err != nil {
		t.Fatalf("spawn app_1: %v", err)
	}

	err = service.CloseForApplication(context.Background(), "app_1", spawned.Browser.ID)
	if err != nil {
		t.Fatalf("close browser: %v", err)
	}

	_, err = service.GetForApplication(context.Background(), "app_1", spawned.Browser.ID)
	if !errors.Is(err, ErrBrowserNotFound) {
		t.Fatalf("expected ErrBrowserNotFound after close, got %v", err)
	}
}
