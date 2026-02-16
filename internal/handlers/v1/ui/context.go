package uihandlers

import "github.com/brian-nunez/bbaas-api/internal/users"

const currentUserContextKey = "current_user"

func setCurrentUser(c contextSetter, user users.User) {
	c.Set(currentUserContextKey, user)
}

func getCurrentUser(c contextGetter) (users.User, bool) {
	value := c.Get(currentUserContextKey)
	user, ok := value.(users.User)
	return user, ok
}

type contextSetter interface {
	Set(key string, value interface{})
}

type contextGetter interface {
	Get(key string) interface{}
}
