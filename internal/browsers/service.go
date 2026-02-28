package browsers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/brian-nunez/bbaas-api/internal/applications"
	"github.com/brian-nunez/bbaas-api/internal/authorization"
	"github.com/brian-nunez/bbaas-api/internal/data"
	"github.com/brian-nunez/bbaas-api/internal/security"
)

var (
	ErrBrowserNotFound = errors.New("browser not found")
	ErrForbidden       = errors.New("forbidden")
)

type Service struct {
	client        ManagerClient
	store         *data.Store
	authorization *authorization.APIAuthorizer
	publicCDPBase string
	now           func() time.Time
}

func NewService(client ManagerClient, store *data.Store, authorizer *authorization.APIAuthorizer, publicCDPBase string) *Service {
	return &Service{
		client:        client,
		store:         store,
		authorization: authorizer,
		publicCDPBase: strings.TrimSpace(publicCDPBase),
		now:           time.Now,
	}
}

func (s *Service) SpawnForAPIKey(ctx context.Context, principal applications.APIKeyPrincipal, request SpawnRequest) (SpawnResponse, error) {
	if !s.can(principal, "browsers.write") {
		return SpawnResponse{}, ErrForbidden
	}

	spawnedBrowser, err := s.client.Spawn(ctx, request)
	if err != nil {
		return SpawnResponse{}, err
	}

	recordID, err := security.GeneratePrefixedToken("bsn", 14)
	if err != nil {
		return SpawnResponse{}, fmt.Errorf("generate browser session id: %w", err)
	}

	record := data.BrowserSessionRecord{
		ID:                recordID,
		ApplicationID:     principal.ApplicationID,
		ExternalBrowserID: spawnedBrowser.Browser.ID,
		Status:            "RUNNING",
		CDPURL:            spawnedBrowser.Browser.CDPURL,
		CDPHTTPURL:        spawnedBrowser.Browser.CDPHTTPURL,
		Headless:          spawnedBrowser.Browser.Headless,
		SpawnTaskProcess:  strings.TrimSpace(spawnedBrowser.SpawnTaskProcessID),
		CreatedAt:         spawnedBrowser.Browser.CreatedAt,
		LastActiveAt:      spawnedBrowser.Browser.LastActiveAt,
		IdleTimeout:       spawnedBrowser.Browser.IdleTimeoutSeconds,
		ExpiresAt:         spawnedBrowser.Browser.ExpiresAt,
	}
	if spawnedBrowser.SpawnedByWorkerID != 0 {
		workerID := spawnedBrowser.SpawnedByWorkerID
		record.SpawnedByWorkerID = &workerID
	}

	if err := s.store.CreateBrowserSession(ctx, record); err != nil {
		return SpawnResponse{}, fmt.Errorf("persist browser session: %w", err)
	}

	spawnedBrowser.Browser = RewriteBrowserForPublicGateway(spawnedBrowser.Browser, s.publicCDPBase)
	return spawnedBrowser, nil
}

func (s *Service) ListForAPIKey(ctx context.Context, principal applications.APIKeyPrincipal) ([]Browser, error) {
	if !s.can(principal, "browsers.read") {
		return nil, ErrForbidden
	}

	recordedSessions, err := s.store.ListBrowserSessionsByApplicationID(ctx, principal.ApplicationID)
	if err != nil {
		return nil, fmt.Errorf("list tracked browser sessions: %w", err)
	}

	activeFromUpstream, err := s.client.List(ctx)
	if err != nil {
		return nil, err
	}

	upstreamByID := make(map[string]Browser, len(activeFromUpstream))
	for _, browser := range activeFromUpstream {
		upstreamByID[browser.ID] = browser
	}

	ownedBrowsers := make([]Browser, 0)
	now := s.now().UTC()
	for _, session := range recordedSessions {
		if session.Status == "COMPLETED" {
			continue
		}

		upstreamBrowser, found := upstreamByID[session.ExternalBrowserID]
		if !found {
			_ = s.store.MarkBrowserSessionCompleted(ctx, principal.ApplicationID, session.ExternalBrowserID, now)
			continue
		}

		upstreamBrowser = RewriteBrowserForPublicGateway(upstreamBrowser, s.publicCDPBase)
		ownedBrowsers = append(ownedBrowsers, upstreamBrowser)
	}

	sort.Slice(ownedBrowsers, func(i int, j int) bool {
		return ownedBrowsers[i].CreatedAt.After(ownedBrowsers[j].CreatedAt)
	})

	return ownedBrowsers, nil
}

func (s *Service) GetForAPIKey(ctx context.Context, principal applications.APIKeyPrincipal, browserID string) (Browser, error) {
	if !s.can(principal, "browsers.read") {
		return Browser{}, ErrForbidden
	}

	if _, err := s.getTrackedSession(ctx, principal.ApplicationID, browserID); err != nil {
		return Browser{}, err
	}

	browser, err := s.client.Get(ctx, browserID)
	if err != nil {
		if isNotFoundError(err) {
			_ = s.store.MarkBrowserSessionCompleted(ctx, principal.ApplicationID, browserID, s.now().UTC())
			return Browser{}, ErrBrowserNotFound
		}
		return Browser{}, err
	}

	if err := s.store.UpdateBrowserSessionHeartbeat(ctx, principal.ApplicationID, browser.ID, mapBrowserToSessionRecord(principal.ApplicationID, browser)); err != nil {
		return Browser{}, fmt.Errorf("update browser heartbeat: %w", err)
	}

	browser = RewriteBrowserForPublicGateway(browser, s.publicCDPBase)
	return browser, nil
}

func (s *Service) KeepAliveForAPIKey(ctx context.Context, principal applications.APIKeyPrincipal, browserID string) (Browser, error) {
	if !s.can(principal, "browsers.write") {
		return Browser{}, ErrForbidden
	}

	if _, err := s.getTrackedSession(ctx, principal.ApplicationID, browserID); err != nil {
		return Browser{}, err
	}

	browser, err := s.client.KeepAlive(ctx, browserID)
	if err != nil {
		if isNotFoundError(err) {
			_ = s.store.MarkBrowserSessionCompleted(ctx, principal.ApplicationID, browserID, s.now().UTC())
			return Browser{}, ErrBrowserNotFound
		}
		return Browser{}, err
	}

	if err := s.store.UpdateBrowserSessionHeartbeat(ctx, principal.ApplicationID, browser.ID, mapBrowserToSessionRecord(principal.ApplicationID, browser)); err != nil {
		return Browser{}, fmt.Errorf("update browser heartbeat: %w", err)
	}

	browser = RewriteBrowserForPublicGateway(browser, s.publicCDPBase)
	return browser, nil
}

func (s *Service) CloseForAPIKey(ctx context.Context, principal applications.APIKeyPrincipal, browserID string) error {
	if !s.can(principal, "browsers.delete") {
		return ErrForbidden
	}

	if _, err := s.getTrackedSession(ctx, principal.ApplicationID, browserID); err != nil {
		return err
	}

	if err := s.client.Close(ctx, browserID); err != nil {
		if isNotFoundError(err) {
			_ = s.store.MarkBrowserSessionCompleted(ctx, principal.ApplicationID, browserID, s.now().UTC())
			return ErrBrowserNotFound
		}
		return err
	}

	if err := s.store.MarkBrowserSessionCompleted(ctx, principal.ApplicationID, browserID, s.now().UTC()); err != nil {
		return fmt.Errorf("mark browser session completed: %w", err)
	}

	return nil
}

func (s *Service) can(principal applications.APIKeyPrincipal, action string) bool {
	return s.authorization.Can(
		authorization.APIKeySubject{
			AppID:     principal.ApplicationID,
			Roles:     []string{"api_key"},
			CanRead:   principal.Permissions.CanRead,
			CanWrite:  principal.Permissions.CanWrite,
			CanDelete: principal.Permissions.CanDelete,
		},
		authorization.BrowserResource{AppID: principal.ApplicationID},
		action,
	)
}

func (s *Service) getTrackedSession(ctx context.Context, applicationID string, browserID string) (data.BrowserSessionRecord, error) {
	session, found, err := s.store.GetBrowserSessionByExternalID(ctx, applicationID, browserID)
	if err != nil {
		return data.BrowserSessionRecord{}, fmt.Errorf("lookup tracked browser session: %w", err)
	}
	if !found || session.Status == "COMPLETED" {
		return data.BrowserSessionRecord{}, ErrBrowserNotFound
	}

	return session, nil
}

func mapBrowserToSessionRecord(applicationID string, browser Browser) data.BrowserSessionRecord {
	return data.BrowserSessionRecord{
		ApplicationID:     applicationID,
		ExternalBrowserID: browser.ID,
		CDPURL:            browser.CDPURL,
		CDPHTTPURL:        browser.CDPHTTPURL,
		Headless:          browser.Headless,
		LastActiveAt:      browser.LastActiveAt,
		IdleTimeout:       browser.IdleTimeoutSeconds,
		ExpiresAt:         browser.ExpiresAt,
	}
}

func isNotFoundError(err error) bool {
	var upstreamError *UpstreamError
	if errors.As(err, &upstreamError) {
		return upstreamError.StatusCode == http.StatusNotFound
	}

	return false
}
