package browsers

import "time"

type SpawnRequest struct {
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

type SpawnResponse struct {
	Browser            Browser `json:"browser"`
	SpawnTaskProcessID string  `json:"spawnTaskProcessId"`
	SpawnedByWorkerID  int     `json:"spawnedByWorkerId"`
}
