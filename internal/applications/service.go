package applications

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/brian-nunez/bbaas-api/internal/authorization"
	"github.com/brian-nunez/bbaas-api/internal/data"
	"github.com/brian-nunez/bbaas-api/internal/security"
	"github.com/brian-nunez/bbaas-api/internal/users"
)

var (
	ErrApplicationNameRequired   = errors.New("application name is required")
	ErrApplicationNameTooLong    = errors.New("application name cannot be longer than 80 characters")
	ErrDescriptionTooLong        = errors.New("description cannot be longer than 500 characters")
	ErrGitHubLinkRequired        = errors.New("github link is required")
	ErrInvalidGitHubLink         = errors.New("github link must be a valid github.com URL")
	ErrDomainRequired            = errors.New("domain is required")
	ErrInvalidDomain             = errors.New("domain must be a valid host name")
	ErrAPIKeyNameRequired        = errors.New("API key name is required")
	ErrAPIKeyNameTooLong         = errors.New("API key name cannot be longer than 80 characters")
	ErrAPIKeyPermissionsRequired = errors.New("at least one API key permission must be enabled")
	ErrApplicationNotFound       = errors.New("application not found")
	ErrAPIKeyNotFound            = errors.New("API key not found")
	ErrInvalidAPIKey             = errors.New("invalid API key")
	ErrForbidden                 = errors.New("forbidden")
)

type RegisterApplicationInput struct {
	Name        string
	Description string
	GitHubLink  string
	Domain      string
}

type CreateAPIKeyInput struct {
	Name        string
	Permissions APIKeyPermissions
}

type CreateAPIKeyResult struct {
	APIKey APIKey
	Token  string
}

type Service struct {
	store         *data.Store
	webAuthorizer *authorization.WebAuthorizer
	now           func() time.Time
}

func NewService(store *data.Store, webAuthorizer *authorization.WebAuthorizer) *Service {
	return &Service{
		store:         store,
		webAuthorizer: webAuthorizer,
		now:           time.Now,
	}
}

func (s *Service) RegisterApplication(ctx context.Context, actor users.User, input RegisterApplicationInput) (Application, error) {
	if !s.webAuthorizer.Can(toWebSubject(actor), authorization.OwnedResource{OwnerUserID: actor.ID}, "applications.create") {
		return Application{}, ErrForbidden
	}

	normalizedInput, err := normalizeApplicationInput(input)
	if err != nil {
		return Application{}, err
	}

	applicationID, err := security.GeneratePrefixedToken("app", 18)
	if err != nil {
		return Application{}, fmt.Errorf("generate application id: %w", err)
	}

	now := s.now().UTC()
	record := data.ApplicationRecord{
		ID:          applicationID,
		OwnerUserID: actor.ID,
		Name:        normalizedInput.Name,
		Description: normalizedInput.Description,
		GitHubLink:  normalizedInput.GitHubLink,
		Domain:      normalizedInput.Domain,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.store.CreateApplication(ctx, record); err != nil {
		return Application{}, fmt.Errorf("create application: %w", err)
	}

	return mapApplicationRecord(record), nil
}

func (s *Service) ListApplicationsForViewer(ctx context.Context, actor users.User) ([]Application, error) {
	applicationRecords, err := s.store.ListApplicationsByUserID(ctx, actor.ID)
	if err != nil {
		return nil, fmt.Errorf("list applications for viewer: %w", err)
	}

	applications := make([]Application, 0, len(applicationRecords))
	for _, record := range applicationRecords {
		applications = append(applications, mapApplicationRecord(record))
	}

	return applications, nil
}

func (s *Service) ListAPIKeysForApplication(ctx context.Context, actor users.User, applicationID string) ([]APIKey, error) {
	applicationRecord, err := s.getOwnedApplication(ctx, actor, applicationID, "applications.read")
	if err != nil {
		return nil, err
	}
	if applicationRecord.ID == "" {
		return nil, ErrApplicationNotFound
	}

	keyRecords, err := s.store.ListAPIKeysByApplicationID(ctx, applicationRecord.ID)
	if err != nil {
		return nil, fmt.Errorf("list API keys by application: %w", err)
	}

	keys := make([]APIKey, 0, len(keyRecords))
	for _, record := range keyRecords {
		keys = append(keys, mapAPIKeyRecord(record))
	}

	return keys, nil
}

func (s *Service) CreateAPIKey(ctx context.Context, actor users.User, applicationID string, input CreateAPIKeyInput) (CreateAPIKeyResult, error) {
	applicationRecord, err := s.getOwnedApplication(ctx, actor, applicationID, "api_keys.create")
	if err != nil {
		return CreateAPIKeyResult{}, err
	}
	if applicationRecord.ID == "" {
		return CreateAPIKeyResult{}, ErrApplicationNotFound
	}

	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return CreateAPIKeyResult{}, ErrAPIKeyNameRequired
	}
	if len(input.Name) > 80 {
		return CreateAPIKeyResult{}, ErrAPIKeyNameTooLong
	}
	if !input.Permissions.CanRead && !input.Permissions.CanWrite && !input.Permissions.CanDelete {
		return CreateAPIKeyResult{}, ErrAPIKeyPermissionsRequired
	}

	rawToken, err := security.GeneratePrefixedToken("bka", 24)
	if err != nil {
		return CreateAPIKeyResult{}, fmt.Errorf("generate API key token: %w", err)
	}

	keyID, err := security.GeneratePrefixedToken("key", 14)
	if err != nil {
		return CreateAPIKeyResult{}, fmt.Errorf("generate API key id: %w", err)
	}

	keyPrefix := rawToken
	if len(rawToken) > 12 {
		keyPrefix = rawToken[:12]
	}

	now := s.now().UTC()
	record := data.APIKeyRecord{
		ID:            keyID,
		ApplicationID: applicationRecord.ID,
		Name:          input.Name,
		KeyPrefix:     keyPrefix,
		KeyHash:       security.DigestSHA256(rawToken),
		CanRead:       input.Permissions.CanRead,
		CanWrite:      input.Permissions.CanWrite,
		CanDelete:     input.Permissions.CanDelete,
		CreatedAt:     now,
	}

	if err := s.store.CreateAPIKey(ctx, record); err != nil {
		return CreateAPIKeyResult{}, fmt.Errorf("create API key: %w", err)
	}

	return CreateAPIKeyResult{
		APIKey: mapAPIKeyRecord(record),
		Token:  rawToken,
	}, nil
}

func (s *Service) RevokeAPIKey(ctx context.Context, actor users.User, applicationID string, keyID string) error {
	applicationRecord, err := s.getOwnedApplication(ctx, actor, applicationID, "api_keys.delete")
	if err != nil {
		return err
	}
	if applicationRecord.ID == "" {
		return ErrApplicationNotFound
	}

	revoked, err := s.store.RevokeAPIKey(ctx, applicationRecord.ID, keyID, s.now().UTC())
	if err != nil {
		return fmt.Errorf("revoke API key: %w", err)
	}
	if !revoked {
		return ErrAPIKeyNotFound
	}

	return nil
}

func (s *Service) AuthenticateAPIKey(ctx context.Context, rawToken string) (APIKeyPrincipal, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return APIKeyPrincipal{}, ErrInvalidAPIKey
	}

	authRecord, found, err := s.store.GetActiveAPIKeyAuthByHash(ctx, security.DigestSHA256(rawToken))
	if err != nil {
		return APIKeyPrincipal{}, fmt.Errorf("lookup API key by hash: %w", err)
	}
	if !found {
		return APIKeyPrincipal{}, ErrInvalidAPIKey
	}

	_ = s.store.TouchAPIKeyLastUsed(ctx, authRecord.Key.ID, s.now().UTC())

	return APIKeyPrincipal{
		KeyID:         authRecord.Key.ID,
		ApplicationID: authRecord.Application.ID,
		Permissions: APIKeyPermissions{
			CanRead:   authRecord.Key.CanRead,
			CanWrite:  authRecord.Key.CanWrite,
			CanDelete: authRecord.Key.CanDelete,
		},
	}, nil
}

func normalizeApplicationInput(input RegisterApplicationInput) (RegisterApplicationInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	input.GitHubLink = strings.TrimSpace(input.GitHubLink)
	input.Domain = strings.TrimSpace(strings.ToLower(input.Domain))

	if input.Name == "" {
		return RegisterApplicationInput{}, ErrApplicationNameRequired
	}
	if len(input.Name) > 80 {
		return RegisterApplicationInput{}, ErrApplicationNameTooLong
	}
	if len(input.Description) > 500 {
		return RegisterApplicationInput{}, ErrDescriptionTooLong
	}
	if input.GitHubLink == "" {
		return RegisterApplicationInput{}, ErrGitHubLinkRequired
	}
	if !isValidGitHubURL(input.GitHubLink) {
		return RegisterApplicationInput{}, ErrInvalidGitHubLink
	}
	if input.Domain == "" {
		return RegisterApplicationInput{}, ErrDomainRequired
	}
	if !isValidDomain(input.Domain) {
		return RegisterApplicationInput{}, ErrInvalidDomain
	}

	return input, nil
}

func isValidGitHubURL(rawURL string) bool {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	if parsedURL.Scheme != "https" && parsedURL.Scheme != "http" {
		return false
	}

	host := strings.ToLower(parsedURL.Host)
	if host != "github.com" && host != "www.github.com" {
		return false
	}

	return strings.Trim(parsedURL.Path, "/") != ""
}

func isValidDomain(domain string) bool {
	if strings.Contains(domain, " ") {
		return false
	}
	if strings.Contains(domain, "://") {
		return false
	}
	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return false
	}

	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return false
	}

	for _, part := range parts {
		if part == "" {
			return false
		}
	}

	return true
}

func (s *Service) getOwnedApplication(ctx context.Context, actor users.User, applicationID string, action string) (data.ApplicationRecord, error) {
	applicationID = strings.TrimSpace(applicationID)
	if applicationID == "" {
		return data.ApplicationRecord{}, ErrApplicationNotFound
	}

	record, found, err := s.store.GetApplicationByID(ctx, applicationID)
	if err != nil {
		return data.ApplicationRecord{}, fmt.Errorf("lookup application by id: %w", err)
	}
	if !found {
		return data.ApplicationRecord{}, nil
	}

	isAllowed := s.webAuthorizer.Can(
		toWebSubject(actor),
		authorization.OwnedResource{OwnerUserID: record.OwnerUserID},
		action,
	)
	if !isAllowed {
		return data.ApplicationRecord{}, ErrForbidden
	}

	return record, nil
}

func mapApplicationRecord(record data.ApplicationRecord) Application {
	return Application{
		ID:          record.ID,
		OwnerUserID: record.OwnerUserID,
		Name:        record.Name,
		Description: record.Description,
		GitHubLink:  record.GitHubLink,
		Domain:      record.Domain,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
	}
}

func mapAPIKeyRecord(record data.APIKeyRecord) APIKey {
	return APIKey{
		ID:            record.ID,
		ApplicationID: record.ApplicationID,
		Name:          record.Name,
		KeyPrefix:     record.KeyPrefix,
		CanRead:       record.CanRead,
		CanWrite:      record.CanWrite,
		CanDelete:     record.CanDelete,
		CreatedAt:     record.CreatedAt,
		LastUsedAt:    record.LastUsedAt,
		RevokedAt:     record.RevokedAt,
	}
}

func toWebSubject(user users.User) authorization.WebSubject {
	roles := []string{"user"}
	if user.IsAdmin() {
		roles = append(roles, "admin")
	}

	return authorization.WebSubject{
		UserID: user.ID,
		Roles:  roles,
	}
}
