package users

import "time"

type User struct {
	ID        string
	Email     string
	Role      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (u User) IsAdmin() bool {
	return u.Role == "admin"
}
