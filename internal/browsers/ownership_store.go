package browsers

import (
	"context"
	"sync"
)

type OwnershipStore interface {
	SetOwner(ctx context.Context, applicationID string, browserID string) error
	IsOwnedBy(ctx context.Context, applicationID string, browserID string) (bool, error)
	Remove(ctx context.Context, browserID string) error
}

type InMemoryOwnershipStore struct {
	mu     sync.RWMutex
	owners map[string]string
}

func NewInMemoryOwnershipStore() *InMemoryOwnershipStore {
	return &InMemoryOwnershipStore{
		owners: make(map[string]string),
	}
}

func (s *InMemoryOwnershipStore) SetOwner(_ context.Context, applicationID string, browserID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.owners[browserID] = applicationID
	return nil
}

func (s *InMemoryOwnershipStore) IsOwnedBy(_ context.Context, applicationID string, browserID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	owner, ok := s.owners[browserID]
	if !ok {
		return false, nil
	}

	return owner == applicationID, nil
}

func (s *InMemoryOwnershipStore) Remove(_ context.Context, browserID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.owners, browserID)
	return nil
}
