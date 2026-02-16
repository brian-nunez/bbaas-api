package browsers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

var ErrBrowserNotFound = errors.New("browser not found")

type Service struct {
	client         ManagerClient
	ownershipStore OwnershipStore
}

func NewService(client ManagerClient, ownershipStore OwnershipStore) *Service {
	return &Service{
		client:         client,
		ownershipStore: ownershipStore,
	}
}

func (s *Service) SpawnForApplication(ctx context.Context, applicationID string, request SpawnRequest) (SpawnResponse, error) {
	spawnedBrowser, err := s.client.Spawn(ctx, request)
	if err != nil {
		return SpawnResponse{}, err
	}

	if err := s.ownershipStore.SetOwner(ctx, applicationID, spawnedBrowser.Browser.ID); err != nil {
		return SpawnResponse{}, fmt.Errorf("store browser ownership: %w", err)
	}

	return spawnedBrowser, nil
}

func (s *Service) ListForApplication(ctx context.Context, applicationID string) ([]Browser, error) {
	allBrowsers, err := s.client.List(ctx)
	if err != nil {
		return nil, err
	}

	ownedBrowsers := make([]Browser, 0, len(allBrowsers))
	for _, browser := range allBrowsers {
		ownedByApplication, err := s.ownershipStore.IsOwnedBy(ctx, applicationID, browser.ID)
		if err != nil {
			return nil, fmt.Errorf("lookup browser ownership: %w", err)
		}
		if ownedByApplication {
			ownedBrowsers = append(ownedBrowsers, browser)
		}
	}

	return ownedBrowsers, nil
}

func (s *Service) GetForApplication(ctx context.Context, applicationID string, browserID string) (Browser, error) {
	isOwnedByApplication, err := s.ownershipStore.IsOwnedBy(ctx, applicationID, browserID)
	if err != nil {
		return Browser{}, fmt.Errorf("lookup browser ownership: %w", err)
	}
	if !isOwnedByApplication {
		return Browser{}, ErrBrowserNotFound
	}

	browser, err := s.client.Get(ctx, browserID)
	if err != nil {
		if isNotFoundError(err) {
			_ = s.ownershipStore.Remove(ctx, browserID)
			return Browser{}, ErrBrowserNotFound
		}
		return Browser{}, err
	}

	return browser, nil
}

func (s *Service) KeepAliveForApplication(ctx context.Context, applicationID string, browserID string) (Browser, error) {
	isOwnedByApplication, err := s.ownershipStore.IsOwnedBy(ctx, applicationID, browserID)
	if err != nil {
		return Browser{}, fmt.Errorf("lookup browser ownership: %w", err)
	}
	if !isOwnedByApplication {
		return Browser{}, ErrBrowserNotFound
	}

	browser, err := s.client.KeepAlive(ctx, browserID)
	if err != nil {
		if isNotFoundError(err) {
			_ = s.ownershipStore.Remove(ctx, browserID)
			return Browser{}, ErrBrowserNotFound
		}
		return Browser{}, err
	}

	return browser, nil
}

func (s *Service) CloseForApplication(ctx context.Context, applicationID string, browserID string) error {
	isOwnedByApplication, err := s.ownershipStore.IsOwnedBy(ctx, applicationID, browserID)
	if err != nil {
		return fmt.Errorf("lookup browser ownership: %w", err)
	}
	if !isOwnedByApplication {
		return ErrBrowserNotFound
	}

	if err := s.client.Close(ctx, browserID); err != nil {
		if isNotFoundError(err) {
			_ = s.ownershipStore.Remove(ctx, browserID)
			return ErrBrowserNotFound
		}
		return err
	}

	if err := s.ownershipStore.Remove(ctx, browserID); err != nil {
		return fmt.Errorf("remove browser ownership: %w", err)
	}

	return nil
}

func isNotFoundError(err error) bool {
	var upstreamError *UpstreamError
	if errors.As(err, &upstreamError) {
		return upstreamError.StatusCode == http.StatusNotFound
	}

	return false
}
