package auth

import "github.com/webitel/webitel-go-kit/pkg/errors"

// Keys of the values attached to ErrLicense and ErrPermission, read with errors.Value.
const (
	ValueLicenses = "licenses" // []string: the missing licenses
	ValueClass    = "class"    // string: the object class that was denied
)

var (
	// ErrInvalidToken means go.webitel.app rejected the token.
	ErrInvalidToken = errors.Unauthenticated("invalid access token", errors.WithID("auth.token.invalid"))
	// ErrUnavailable means go.webitel.app could not answer.
	ErrUnavailable = errors.Unavailable("authorization service is unavailable", errors.WithID("auth.provider.unavailable"))
	// ErrMethodUnknown means the method is not in the WebitelAPI table.
	ErrMethodUnknown = errors.Forbidden("method is not exposed for authorization", errors.WithID("auth.method.unknown"))
	// ErrTokenMissing means the request carries no token.
	ErrTokenMissing = errors.Unauthenticated("authorization token is missing", errors.WithID("auth.token.missing"))
	// ErrSessionExpired means there is no session or it has expired.
	ErrSessionExpired = errors.Unauthenticated("session has expired", errors.WithID("auth.session.expired"))
	// ErrLicense means required licenses are missing; ValueLicenses lists them.
	ErrLicense = errors.Forbidden("missing required licenses", errors.WithID("auth.license"))
	// ErrPermission means obac denied the call; ValueClass names the class.
	ErrPermission = errors.Forbidden("missing required permissions", errors.WithID("auth.permission"))
)
