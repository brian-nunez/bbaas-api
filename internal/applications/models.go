package applications

import "time"

type Application struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	GitHubProfileLink string    `json:"githubProfileLink"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type RegisterInput struct {
	Name              string
	Description       string
	GitHubProfileLink string
}

type RegisterOutput struct {
	Application Application `json:"application"`
	APIToken    string      `json:"apiToken"`
}
