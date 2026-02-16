package applications

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

var (
	ErrNameRequired       = errors.New("application name is required")
	ErrNameTooLong        = errors.New("application name cannot be longer than 80 characters")
	ErrDescriptionTooLong = errors.New("description cannot be longer than 500 characters")
	ErrGitHubLinkRequired = errors.New("github profile link is required")
	ErrInvalidGitHubLink  = errors.New("github profile link must be a valid github.com URL")
	ErrInvalidAPIToken    = errors.New("invalid API token")
)

type Service struct {
	repository Repository
	now        func() time.Time
}

func NewService(repository Repository) *Service {
	return &Service{
		repository: repository,
		now:        time.Now,
	}
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (RegisterOutput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	input.GitHubProfileLink = strings.TrimSpace(input.GitHubProfileLink)

	if err := validateRegisterInput(input); err != nil {
		return RegisterOutput{}, err
	}

	apiToken, err := generateToken("bbaas")
	if err != nil {
		return RegisterOutput{}, fmt.Errorf("generate API token: %w", err)
	}

	appID, err := generateToken("app")
	if err != nil {
		return RegisterOutput{}, fmt.Errorf("generate application id: %w", err)
	}

	now := s.now().UTC()
	app := Application{
		ID:                appID,
		Name:              input.Name,
		Description:       input.Description,
		GitHubProfileLink: input.GitHubProfileLink,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := s.repository.Create(ctx, app, digestToken(apiToken)); err != nil {
		return RegisterOutput{}, fmt.Errorf("store application: %w", err)
	}

	return RegisterOutput{
		Application: app,
		APIToken:    apiToken,
	}, nil
}

func (s *Service) AuthenticateToken(ctx context.Context, token string) (Application, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return Application{}, ErrInvalidAPIToken
	}

	app, ok, err := s.repository.GetByTokenDigest(ctx, digestToken(token))
	if err != nil {
		return Application{}, fmt.Errorf("get application by token: %w", err)
	}
	if !ok {
		return Application{}, ErrInvalidAPIToken
	}

	return app, nil
}

func validateRegisterInput(input RegisterInput) error {
	if input.Name == "" {
		return ErrNameRequired
	}
	if len(input.Name) > 80 {
		return ErrNameTooLong
	}
	if len(input.Description) > 500 {
		return ErrDescriptionTooLong
	}
	if input.GitHubProfileLink == "" {
		return ErrGitHubLinkRequired
	}
	if !isValidGitHubProfileLink(input.GitHubProfileLink) {
		return ErrInvalidGitHubLink
	}

	return nil
}

func isValidGitHubProfileLink(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return false
	}

	host := strings.ToLower(parsed.Host)
	if host != "github.com" && host != "www.github.com" {
		return false
	}

	trimmedPath := strings.Trim(parsed.Path, "/")
	return trimmedPath != ""
}

func generateToken(prefix string) (string, error) {
	buffer := make([]byte, 18)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}

	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(buffer)), nil
}

func digestToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
