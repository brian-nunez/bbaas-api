package applications

import (
	"context"
	"sync"
)

type applicationRecord struct {
	application Application
	tokenDigest string
}

type InMemoryRepository struct {
	mu           sync.RWMutex
	applications map[string]applicationRecord
	tokenToAppID map[string]string
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		applications: make(map[string]applicationRecord),
		tokenToAppID: make(map[string]string),
	}
}

func (r *InMemoryRepository) Create(_ context.Context, app Application, tokenDigest string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.applications[app.ID] = applicationRecord{
		application: app,
		tokenDigest: tokenDigest,
	}
	r.tokenToAppID[tokenDigest] = app.ID

	return nil
}

func (r *InMemoryRepository) GetByID(_ context.Context, id string) (Application, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	record, ok := r.applications[id]
	if !ok {
		return Application{}, false, nil
	}

	return record.application, true, nil
}

func (r *InMemoryRepository) GetByTokenDigest(_ context.Context, tokenDigest string) (Application, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	appID, ok := r.tokenToAppID[tokenDigest]
	if !ok {
		return Application{}, false, nil
	}

	record, ok := r.applications[appID]
	if !ok {
		return Application{}, false, nil
	}

	return record.application, true, nil
}
