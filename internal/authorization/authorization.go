package authorization

import baccess "github.com/brian-nunez/baccess"

type WebSubject struct {
	UserID string
	Roles  []string
}

func (s WebSubject) GetRoles() []string {
	return s.Roles
}

type OwnedResource struct {
	OwnerUserID string
}

type WebAuthorizer struct {
	evaluator *baccess.Evaluator[WebSubject, OwnedResource]
}

func NewWebAuthorizer() *WebAuthorizer {
	evaluator := baccess.NewEvaluator[WebSubject, OwnedResource]()
	rbac := baccess.NewRBAC[WebSubject, OwnedResource]()

	adminRole := rbac.HasRole("admin")
	userRole := rbac.HasRole("user")
	ownerOnly := baccess.FieldEquals(
		func(subject WebSubject) string { return subject.UserID },
		func(resource OwnedResource) string { return resource.OwnerUserID },
	)

	evaluator.AddPolicy("applications.create", adminRole.Or(userRole))
	evaluator.AddPolicy("applications.read", adminRole.Or(userRole.And(ownerOnly)))
	evaluator.AddPolicy("api_keys.create", adminRole.Or(userRole.And(ownerOnly)))
	evaluator.AddPolicy("api_keys.delete", adminRole.Or(userRole.And(ownerOnly)))
	evaluator.AddPolicy("users.read", adminRole.Or(userRole))

	return &WebAuthorizer{evaluator: evaluator}
}

func (a *WebAuthorizer) Can(subject WebSubject, resource OwnedResource, action string) bool {
	return a.evaluator.Evaluate(baccess.AccessRequest[WebSubject, OwnedResource]{
		Subject:  subject,
		Resource: resource,
		Action:   action,
	})
}

type APIKeySubject struct {
	AppID     string
	Roles     []string
	CanRead   bool
	CanWrite  bool
	CanDelete bool
}

func (s APIKeySubject) GetRoles() []string {
	return s.Roles
}

type BrowserResource struct {
	AppID string
}

type APIAuthorizer struct {
	evaluator *baccess.Evaluator[APIKeySubject, BrowserResource]
}

func NewAPIAuthorizer() *APIAuthorizer {
	evaluator := baccess.NewEvaluator[APIKeySubject, BrowserResource]()
	rbac := baccess.NewRBAC[APIKeySubject, BrowserResource]()

	apiKeyRole := rbac.HasRole("api_key")
	sameApp := baccess.FieldEquals(
		func(subject APIKeySubject) string { return subject.AppID },
		func(resource BrowserResource) string { return resource.AppID },
	)
	canRead := baccess.SubjectMatches[APIKeySubject, BrowserResource, bool](
		func(subject APIKeySubject) bool { return subject.CanRead },
		true,
	)
	canWrite := baccess.SubjectMatches[APIKeySubject, BrowserResource, bool](
		func(subject APIKeySubject) bool { return subject.CanWrite },
		true,
	)
	canDelete := baccess.SubjectMatches[APIKeySubject, BrowserResource, bool](
		func(subject APIKeySubject) bool { return subject.CanDelete },
		true,
	)

	evaluator.AddPolicy("browsers.read", apiKeyRole.And(sameApp).And(canRead))
	evaluator.AddPolicy("browsers.write", apiKeyRole.And(sameApp).And(canWrite))
	evaluator.AddPolicy("browsers.delete", apiKeyRole.And(sameApp).And(canDelete))

	return &APIAuthorizer{evaluator: evaluator}
}

func (a *APIAuthorizer) Can(subject APIKeySubject, resource BrowserResource, action string) bool {
	return a.evaluator.Evaluate(baccess.AccessRequest[APIKeySubject, BrowserResource]{
		Subject:  subject,
		Resource: resource,
		Action:   action,
	})
}
