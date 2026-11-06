package auth

import "github.com/webitel/webitel-go-kit/pkg/errors"

type valueKey string

const (
	ValueLicenses valueKey = "licenses" // []string: the missing licenses
	ValueClass    valueKey = "class"    // string: the object class that was denied
)

const idCanceled = "auth.request.canceled"

var (
	ErrInvalidToken   = errors.Unauthenticated("invalid access token", errors.WithID("auth.token.invalid"))
	ErrUnavailable    = errors.Unavailable("authorization service is unavailable", errors.WithID("auth.provider.unavailable"))
	ErrMethodUnknown  = errors.Forbidden("method is not exposed for authorization", errors.WithID("auth.method.unknown"))
	ErrTokenMissing   = errors.Unauthenticated("authorization token is missing", errors.WithID("auth.token.missing"))
	ErrSessionExpired = errors.Unauthenticated("session has expired", errors.WithID("auth.session.expired"))
	ErrLicense        = errors.Forbidden("missing required licenses", errors.WithID("auth.license"))
	ErrPermission     = errors.Forbidden("missing required permissions", errors.WithID("auth.permission"))
)
