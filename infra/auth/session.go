package auth

import (
	"slices"
	"strings"
	"time"
)

// Action is what a method does; values match the proto option.Action enum.
type Action uint8

const (
	ActionCreate Action = iota
	ActionRead
	ActionUpdate
	ActionDelete
)

// flag is the Scope.Access letter of the action, 0 if unknown.
func (a Action) flag() byte {
	switch a {
	case ActionCreate:
		return 'x'
	case ActionRead:
		return 'r'
	case ActionUpdate:
		return 'w'
	case ActionDelete:
		return 'd'
	default:
		return 0
	}
}

// superPermission grants the action on any class in scope, "" if unknown.
func (a Action) superPermission() string {
	switch a {
	case ActionCreate:
		return "add"
	case ActionRead:
		return "read"
	case ActionUpdate:
		return "write"
	case ActionDelete:
		return "delete"
	default:
		return ""
	}
}

// Scope is the access a user has to one object class.
type Scope struct {
	Class string
	// Access is a mask of the flags r, w, x, d.
	Access string
	// OBAC enables the operation-based check of Access.
	OBAC bool
	// RBAC enables the record-based check; see Session.RequiresRBAC.
	RBAC bool
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
	// RoleIDs excludes the user id; ACLRoles adds it.
	RoleIDs     []int64
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
	return slices.Contains(s.Permissions, name)
}

func (s *Session) hasSuper(a Action) bool {
	return s.HasPermission(a.superPermission())
}

// CanAccess is the obac check for the action on the class.
func (s *Session) CanAccess(class string, a Action) bool {
	scope, ok := s.Scopes[class]
	if !ok || a > ActionDelete {
		return false
	}
	if !scope.OBAC || s.hasSuper(a) {
		return true
	}
	return strings.IndexByte(scope.Access, a.flag()) >= 0
}

// RequiresRBAC reports whether the handler must restrict records to ACLRoles.
func (s *Session) RequiresRBAC(class string, a Action) bool {
	scope, ok := s.Scopes[class]

	return ok && scope.RBAC && !s.hasSuper(a)
}

// ACLRoles returns the user id followed by the role ids, as record ACLs match them.
func (s *Session) ACLRoles() []int64 {
	roles := make([]int64, 0, 1+len(s.RoleIDs))

	return append(append(roles, s.UserID), s.RoleIDs...)
}
