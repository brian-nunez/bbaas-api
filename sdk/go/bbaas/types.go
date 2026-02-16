package bbaas

import "time"

type RegisterApplicationRequest struct {
	Name              string `json:"name"`
	Description       string `json:"description"`
	GitHubProfileLink string `json:"githubProfileLink"`
}

type Application struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	GitHubProfileLink string    `json:"githubProfileLink"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type RegisterApplicationResponse struct {
	Application Application `json:"application"`
	APIToken    string      `json:"apiToken"`
}

type SpawnBrowserRequest struct {
	Headless           *bool `json:"headless,omitempty"`
	IdleTimeoutSeconds *int  `json:"idleTimeoutSeconds,omitempty"`
}

type Browser struct {
	ID                 string    `json:"id"`
	CDPURL             string    `json:"cdpUrl"`
	CDPHTTPURL         string    `json:"cdpHttpUrl"`
	Headless           bool      `json:"headless"`
	CreatedAt          time.Time `json:"createdAt"`
	LastActiveAt       time.Time `json:"lastActiveAt"`
	IdleTimeoutSeconds int       `json:"idleTimeoutSeconds"`
	ExpiresAt          time.Time `json:"expiresAt"`
}

type SpawnBrowserResponse struct {
	Browser            Browser `json:"browser"`
	SpawnTaskProcessID string  `json:"spawnTaskProcessId"`
	SpawnedByWorkerID  int     `json:"spawnedByWorkerId"`
}
