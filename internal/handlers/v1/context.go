package v1

import "github.com/brian-nunez/bbaas-api/internal/applications"

const apiKeyPrincipalContextKey = "api_key_principal"

func setAPIKeyPrincipal(c contextSetter, principal applications.APIKeyPrincipal) {
	c.Set(apiKeyPrincipalContextKey, principal)
}

func getAPIKeyPrincipal(c contextGetter) (applications.APIKeyPrincipal, bool) {
	value := c.Get(apiKeyPrincipalContextKey)
	principal, ok := value.(applications.APIKeyPrincipal)
	return principal, ok
}

type contextSetter interface {
	Set(key string, val interface{})
}

type contextGetter interface {
	Get(key string) interface{}
}
