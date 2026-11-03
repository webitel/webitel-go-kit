package auth

import (
	"strings"
	"time"
)

// Access is the action a method performs; values match the proto option.Action enum.
type Access uint8

const (
	Create Access = iota
	Read
	Update
	Delete
)

// letter is the Scope.Access flag; super is the permission granting it on any class in scope.
func (a Access) letter() (byte, string) {
	switch a {
	case Create:
		return 'x', "add"
	case Read:
		return 'r', "read"
	case Update:
		return 'w', "write"
	case Delete:
		return 'd', "delete"
	default:
		return 0, ""
	}
}

// Scope is the access a user has to one object class.
type Scope struct {
	Class string
	// Access is a mask of the flags r, w, x, d.
	Access string
	// Obac enables the operation-based check of Access.
	Obac bool
	// Rbac enables the record-based check; see Session.RequiresRBAC.
	Rbac bool
}

// License is a product license; a zero IssuedAt or ExpiresAt means no bound.
type License struct {
	Prod      string
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// Session is the caller identity; it is shared once cached, so keep it read-only.
type Session struct {
	UserID   int64
	DomainID int64
	Name     string
	Username string
	// Roles holds role ids only; ACLRoles adds the user id.
	Roles       []int64
	Permissions []string
	Licenses    []License
	// Scopes is keyed by object class.
	Scopes map[string]Scope
	// ExpiresAt is zero when the token has no expiry.
	ExpiresAt time.Time
}

// Expired reports whether the token is past its expiry.
func (s *Session) Expired() bool {
	return !s.ExpiresAt.IsZero() && !time.Now().Before(s.ExpiresAt)
}

// HasLicense reports whether a license for the product is valid right now.
func (s *Session) HasLicense(prod string) bool {
	now := time.Now()
	for _, l := range s.Licenses {
		started := l.IssuedAt.IsZero() || !now.Before(l.IssuedAt)
		running := l.ExpiresAt.IsZero() || now.Before(l.ExpiresAt)
		if l.Prod == prod && started && running {
			return true
		}
	}

	return false
}

// HasPermission reports whether the user has the named permission.
func (s *Session) HasPermission(name string) bool {
	for _, p := range s.Permissions {
		if p == name {
			return true
		}
	}

	return false
}

func (s *Session) hasSuper(a Access) bool {
	_, super := a.letter()

	return s.HasPermission(super)
}

// CanAccess is the obac check for the action on the class.
func (s *Session) CanAccess(class string, a Access) bool {
	scope, ok := s.Scopes[class]
	if !ok || a > Delete {
		return false
	}
	if !scope.Obac || s.hasSuper(a) {
		return true
	}
	letter, _ := a.letter()

	return strings.IndexByte(scope.Access, letter) >= 0
}

// RequiresRBAC reports whether the handler must restrict records to ACLRoles.
func (s *Session) RequiresRBAC(class string, a Access) bool {
	scope, ok := s.Scopes[class]

	return ok && scope.Rbac && !s.hasSuper(a)
}

// ACLRoles returns the user id followed by the role ids, as record ACLs match them.
func (s *Session) ACLRoles() []int64 {
	roles := make([]int64, 0, 1+len(s.Roles))

	return append(append(roles, s.UserID), s.Roles...)
}
