package applications

import (
	"context"
	"errors"
	"testing"
)

func TestRegisterAndAuthenticateToken(t *testing.T) {
	t.Parallel()

	service := NewService(NewInMemoryRepository())

	registered, err := service.Register(context.Background(), RegisterInput{
		Name:              "My App",
		Description:       "E2E flows",
		GitHubProfileLink: "https://github.com/example",
	})
	if err != nil {
		t.Fatalf("expected successful registration, got error: %v", err)
	}
	if registered.APIToken == "" {
		t.Fatalf("expected API token")
	}
	if registered.Application.ID == "" {
		t.Fatalf("expected application ID")
	}

	authenticated, err := service.AuthenticateToken(context.Background(), registered.APIToken)
	if err != nil {
		t.Fatalf("expected successful authentication, got error: %v", err)
	}
	if authenticated.ID != registered.Application.ID {
		t.Fatalf("expected app id %q, got %q", registered.Application.ID, authenticated.ID)
	}
}

func TestRegisterValidation(t *testing.T) {
	t.Parallel()

	service := NewService(NewInMemoryRepository())

	_, err := service.Register(context.Background(), RegisterInput{
		Name:              "",
		Description:       "desc",
		GitHubProfileLink: "https://github.com/example",
	})
	if !errors.Is(err, ErrNameRequired) {
		t.Fatalf("expected ErrNameRequired, got %v", err)
	}

	_, err = service.Register(context.Background(), RegisterInput{
		Name:              "App",
		Description:       "desc",
		GitHubProfileLink: "https://gitlab.com/example",
	})
	if !errors.Is(err, ErrInvalidGitHubLink) {
		t.Fatalf("expected ErrInvalidGitHubLink, got %v", err)
	}
}

func TestAuthenticateInvalidToken(t *testing.T) {
	t.Parallel()

	service := NewService(NewInMemoryRepository())

	_, err := service.AuthenticateToken(context.Background(), "")
	if !errors.Is(err, ErrInvalidAPIToken) {
		t.Fatalf("expected ErrInvalidAPIToken for empty token, got %v", err)
	}

	_, err = service.AuthenticateToken(context.Background(), "bbaas_invalid")
	if !errors.Is(err, ErrInvalidAPIToken) {
		t.Fatalf("expected ErrInvalidAPIToken for unknown token, got %v", err)
	}
}
