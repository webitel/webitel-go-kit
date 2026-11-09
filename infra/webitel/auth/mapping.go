package auth

import (
	"time"

	authpb "buf.build/gen/go/webitel/webitel-go/protocolbuffers/go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/webitel/webitel-go-kit/pkg/errors"
)

// mapError keeps an outage of the auth service from looking like a bad token.
func mapError(err error) error {
	switch status.Code(err) {
	case codes.Unauthenticated, codes.PermissionDenied, codes.NotFound:
		return errors.Wrap(ErrInvalidToken, errors.WithCause(err))
	default:
		return errors.Wrap(ErrUnavailable, errors.WithCause(err))
	}
}

// millis converts milliseconds; 0 stays the zero time, which means no bound.
func millis(ms int64) time.Time {
	if ms <= 0 {
		return time.Time{}
	}

	return time.UnixMilli(ms)
}

func newSession(info *authpb.Userinfo) *Session {
	s := &Session{
		UserID:   info.GetUserId(),
		DomainID: info.GetDc(),
		Name:     info.GetName(),
		Username: info.GetUsername(),
		Scopes:   make(map[string]Scope, len(info.GetScope())),
	}
	s.ExpiresAt = millis(info.GetExpiresAt())
	for _, r := range info.GetRoles() {
		s.RoleIDs = append(s.RoleIDs, r.GetId())
	}
	for _, p := range info.GetPermissions() {
		s.Permissions = append(s.Permissions, p.GetId())
	}
	for _, l := range info.GetLicense() {
		s.Licenses = append(s.Licenses, License{
			Prod:      l.GetProd(),
			IssuedAt:  millis(l.GetIssuedAt()),
			ExpiresAt: millis(l.GetExpiresAt()),
		})
	}
	for _, sc := range info.GetScope() {
		s.Scopes[sc.GetClass()] = Scope{
			Class:  sc.GetClass(),
			Access: sc.GetAccess(),
			OBAC:   sc.GetObac(),
			RBAC:   sc.GetRbac(),
		}
	}

	return s
}
