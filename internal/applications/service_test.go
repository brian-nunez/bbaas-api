package applications

import (
	"context"
	"testing"

	"github.com/brian-nunez/bbaas-api/internal/authorization"
	"github.com/brian-nunez/bbaas-api/internal/data"
	"github.com/brian-nunez/bbaas-api/internal/users"
)

func TestRegisterApplicationAndAPIKeyLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := setupStore(t)
	usersService := users.NewService(store)
	appsService := NewService(store, authorization.NewWebAuthorizer())

	user, err := usersService.Register(ctx, "builder@example.com", "password123")
	if err != nil {
		t.Fatalf("register user: %v", err)
	}

	application, err := appsService.RegisterApplication(ctx, user, RegisterApplicationInput{
		Name:        "CDP Suite",
		Description: "Runner",
		GitHubLink:  "https://github.com/example-org",
		Domain:      "example.com",
	})
	if err != nil {
		t.Fatalf("register application: %v", err)
	}

	createdKey, err := appsService.CreateAPIKey(ctx, user, application.ID, CreateAPIKeyInput{
		Name: "Primary",
		Permissions: APIKeyPermissions{
			CanRead:  true,
			CanWrite: true,
		},
	})
	if err != nil {
		t.Fatalf("create API key: %v", err)
	}
	if createdKey.Token == "" {
		t.Fatalf("expected API key token")
	}

	principal, err := appsService.AuthenticateAPIKey(ctx, createdKey.Token)
	if err != nil {
		t.Fatalf("authenticate API key: %v", err)
	}
	if principal.ApplicationID != application.ID {
		t.Fatalf("expected application id %s, got %s", application.ID, principal.ApplicationID)
	}
	if !principal.Permissions.CanWrite {
		t.Fatalf("expected write permission")
	}

	if err := appsService.RevokeAPIKey(ctx, user, application.ID, createdKey.APIKey.ID); err != nil {
		t.Fatalf("revoke API key: %v", err)
	}

	if _, err := appsService.AuthenticateAPIKey(ctx, createdKey.Token); err == nil {
		t.Fatalf("expected revoked key to fail authentication")
	}
}

func setupStore(t *testing.T) *data.Store {
	t.Helper()

	db, _, err := data.Open(data.Config{
		Driver: "sqlite",
		DSN:    "file::memory:?cache=shared",
	})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := data.RunMigrations(context.Background(), db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	return data.NewStore(db)
}
