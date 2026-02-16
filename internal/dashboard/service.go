package dashboard

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/brian-nunez/bbaas-api/internal/applications"
	"github.com/brian-nunez/bbaas-api/internal/data"
	"github.com/brian-nunez/bbaas-api/internal/users"
)

type BrowserSession struct {
	ApplicationName   string
	ExternalBrowserID string
	Status            string
	CDPURL            string
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
}

func NewService(store *data.Store, usersService *users.Service, applicationsService *applications.Service) *Service {
	return &Service{
		store:               store,
		usersService:        usersService,
		applicationsService: applicationsService,
	}
}

func (s *Service) BuildViewData(ctx context.Context, viewer users.User) (ViewData, error) {
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
			CreatedAt:         browserRecord.CreatedAt,
			LastActiveAt:      browserRecord.LastActiveAt,
			ClosedAt:          browserRecord.ClosedAt,
			ExpiresAt:         browserRecord.ExpiresAt,
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
