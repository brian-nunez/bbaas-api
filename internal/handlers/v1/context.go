package v1

import "github.com/brian-nunez/bbaas-api/internal/applications"

const applicationContextKey = "application"

func setAuthenticatedApplication(c contextSetter, application applications.Application) {
	c.Set(applicationContextKey, application)
}

func getAuthenticatedApplication(c contextGetter) (applications.Application, bool) {
	value := c.Get(applicationContextKey)
	application, ok := value.(applications.Application)
	return application, ok
}

type contextSetter interface {
	Set(key string, val interface{})
}

type contextGetter interface {
	Get(key string) interface{}
}
