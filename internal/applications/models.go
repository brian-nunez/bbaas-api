package applications

import "time"

type Application struct {
	ID          string
	OwnerUserID string
	Name        string
	Description string
	GitHubLink  string
	Domain      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type APIKey struct {
	ID            string
	ApplicationID string
	Name          string
	KeyPrefix     string
	CanRead       bool
	CanWrite      bool
	CanDelete     bool
	CreatedAt     time.Time
	LastUsedAt    *time.Time
	RevokedAt     *time.Time
}

type APIKeyPermissions struct {
	CanRead   bool
	CanWrite  bool
	CanDelete bool
}

type APIKeyPrincipal struct {
	KeyID         string
	ApplicationID string
	Permissions   APIKeyPermissions
}
