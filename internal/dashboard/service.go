package dashboard

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/brian-nunez/bbaas-api/internal/applications"
	"github.com/brian-nunez/bbaas-api/internal/browsers"
	"github.com/brian-nunez/bbaas-api/internal/data"
	"github.com/brian-nunez/bbaas-api/internal/users"
)

type BrowserSession struct {
	ApplicationName   string
	ExternalBrowserID string
	Status            string
	CDPURL            string
	CDPHTTPURL        string
	CreatedAt         time.Time
	LastActiveAt      time.Time
	ClosedAt          *time.Time
	ExpiresAt         time.Time
}

type ApplicationWithKeys struct {
	Application applications.Application
	APIKeys     []applications.APIKey
}

type ViewData struct {
	CurrentUser       users.User
	VisibleUsers      []users.User
	Applications      []ApplicationWithKeys
	RunningBrowsers   []BrowserSession
	CompletedBrowsers []BrowserSession
}

type Service struct {
	store               *data.Store
	usersService        *users.Service
	applicationsService *applications.Service
	browserManager      browsers.ManagerClient
	publicCDPBase       string
	now                 func() time.Time
}

func NewService(store *data.Store, usersService *users.Service, applicationsService *applications.Service, browserManager browsers.ManagerClient, publicCDPBase string) *Service {
	return &Service{
		store:               store,
		usersService:        usersService,
		applicationsService: applicationsService,
		browserManager:      browserManager,
		publicCDPBase:       strings.TrimSpace(publicCDPBase),
		now:                 time.Now,
	}
}

func (s *Service) BuildViewData(ctx context.Context, viewer users.User) (ViewData, error) {
	if err := s.reconcileRunningSessions(ctx, viewer.ID); err != nil {
		return ViewData{}, fmt.Errorf("reconcile browser sessions: %w", err)
	}

	visibleUsers, err := s.usersService.ListUsersForViewer(ctx, viewer)
	if err != nil {
		return ViewData{}, fmt.Errorf("list visible users: %w", err)
	}

	ownedApplications, err := s.applicationsService.ListApplicationsForViewer(ctx, viewer)
	if err != nil {
		return ViewData{}, fmt.Errorf("list applications: %w", err)
	}

	applicationsWithKeys := make([]ApplicationWithKeys, 0, len(ownedApplications))
	appNameByID := make(map[string]string, len(ownedApplications))
	for _, application := range ownedApplications {
		keys, err := s.applicationsService.ListAPIKeysForApplication(ctx, viewer, application.ID)
		if err != nil {
			return ViewData{}, fmt.Errorf("list API keys for application %s: %w", application.ID, err)
		}

		applicationsWithKeys = append(applicationsWithKeys, ApplicationWithKeys{
			Application: application,
			APIKeys:     keys,
		})
		appNameByID[application.ID] = application.Name
	}

	browserRecords, err := s.store.ListBrowserSessionsByUserID(ctx, viewer.ID, 250)
	if err != nil {
		return ViewData{}, fmt.Errorf("list browser sessions: %w", err)
	}

	runningBrowsers := make([]BrowserSession, 0)
	completedBrowsers := make([]BrowserSession, 0)
	for _, browserRecord := range browserRecords {
		browser := BrowserSession{
			ApplicationName:   appNameByID[browserRecord.ApplicationID],
			ExternalBrowserID: browserRecord.ExternalBrowserID,
			Status:            browserRecord.Status,
			CDPURL:            browserRecord.CDPURL,
			CDPHTTPURL:        browserRecord.CDPHTTPURL,
			CreatedAt:         browserRecord.CreatedAt,
			LastActiveAt:      browserRecord.LastActiveAt,
			ClosedAt:          browserRecord.ClosedAt,
			ExpiresAt:         browserRecord.ExpiresAt,
		}
		publicBrowser := browsers.RewriteBrowserForPublicGateway(
			browsers.Browser{
				CDPHTTPURL: browserRecord.CDPHTTPURL,
				CDPURL:     browserRecord.CDPURL,
			},
			s.publicCDPBase,
		)
		if publicBrowser.CDPHTTPURL != "" {
			browser.CDPHTTPURL = publicBrowser.CDPHTTPURL
		}
		if publicBrowser.CDPURL != "" {
			browser.CDPURL = publicBrowser.CDPURL
		}
		if browser.Status == "RUNNING" {
			runningBrowsers = append(runningBrowsers, browser)
		} else {
			completedBrowsers = append(completedBrowsers, browser)
		}
	}

	sort.Slice(runningBrowsers, func(i int, j int) bool {
		return runningBrowsers[i].CreatedAt.After(runningBrowsers[j].CreatedAt)
	})
	sort.Slice(completedBrowsers, func(i int, j int) bool {
		return completedBrowsers[i].CreatedAt.After(completedBrowsers[j].CreatedAt)
	})

	return ViewData{
		CurrentUser:       viewer,
		VisibleUsers:      visibleUsers,
		Applications:      applicationsWithKeys,
		RunningBrowsers:   runningBrowsers,
		CompletedBrowsers: completedBrowsers,
	}, nil
}

func (s *Service) reconcileRunningSessions(ctx context.Context, viewerUserID string) error {
	if s.browserManager == nil {
		return nil
	}

	recordedSessions, err := s.store.ListBrowserSessionsByUserID(ctx, viewerUserID, 250)
	if err != nil {
		return fmt.Errorf("list user browser sessions for reconciliation: %w", err)
	}

	activeBrowsers, err := s.browserManager.List(ctx)
	if err != nil {
		// Keep dashboard available even if the browser manager is temporarily unavailable.
		return nil
	}

	activeBrowsersByExternalID := make(map[string]browsers.Browser, len(activeBrowsers))
	for _, browser := range activeBrowsers {
		activeBrowsersByExternalID[browser.ID] = browser
	}

	now := s.now().UTC()
	for _, session := range recordedSessions {
		if session.Status != "RUNNING" {
			continue
		}

		activeBrowser, found := activeBrowsersByExternalID[session.ExternalBrowserID]
		if !found {
			if err := s.store.MarkBrowserSessionCompleted(ctx, session.ApplicationID, session.ExternalBrowserID, now); err != nil {
				return fmt.Errorf("mark session %s completed: %w", session.ExternalBrowserID, err)
			}
			continue
		}

		err := s.store.UpdateBrowserSessionHeartbeat(ctx, session.ApplicationID, session.ExternalBrowserID, data.BrowserSessionRecord{
			ApplicationID:     session.ApplicationID,
			ExternalBrowserID: session.ExternalBrowserID,
			CDPURL:            activeBrowser.CDPURL,
			CDPHTTPURL:        activeBrowser.CDPHTTPURL,
			Headless:          activeBrowser.Headless,
			LastActiveAt:      activeBrowser.LastActiveAt,
			IdleTimeout:       activeBrowser.IdleTimeoutSeconds,
			ExpiresAt:         activeBrowser.ExpiresAt,
		})
		if err != nil {
			return fmt.Errorf("update heartbeat for session %s: %w", session.ExternalBrowserID, err)
		}
	}

	return nil
}
