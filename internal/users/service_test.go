package users

import (
	"context"
	"testing"

	"github.com/brian-nunez/bbaas-api/internal/data"
)

func TestRegisterLoginAndAuthenticateSession(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := setupStore(t)
	service := NewService(store)

	registered, err := service.Register(ctx, "owner@example.com", "password123")
	if err != nil {
		t.Fatalf("register user: %v", err)
	}
	if !registered.IsAdmin() {
		t.Fatalf("first registered user should be admin")
	}

	_, sessionToken, err := service.Login(ctx, "owner@example.com", "password123")
	if err != nil {
		t.Fatalf("login user: %v", err)
	}

	authenticated, found, err := service.AuthenticateSession(ctx, sessionToken)
	if err != nil {
		t.Fatalf("authenticate session: %v", err)
	}
	if !found {
		t.Fatalf("expected authenticated session")
	}
	if authenticated.Email != "owner@example.com" {
		t.Fatalf("expected email owner@example.com, got %s", authenticated.Email)
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
