package auth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCanAccess(t *testing.T) {
	sess := func(scope Scope, perms ...string) *Session {
		return &Session{Scopes: map[string]Scope{"forms": scope}, Permissions: perms}
	}

	tests := []struct {
		name   string
		s      *Session
		class  string
		access Action
		want   bool
	}{
		{"unknown class", sess(Scope{OBAC: true, Access: "rwxd"}), "other", ActionRead, false},
		{"obac off allows without flags", sess(Scope{}), "forms", ActionDelete, true},
		{"create needs x", sess(Scope{OBAC: true, Access: "rwd"}), "forms", ActionCreate, false},
		{"create with x", sess(Scope{OBAC: true, Access: "x"}), "forms", ActionCreate, true},
		{"read needs r", sess(Scope{OBAC: true, Access: "wxd"}), "forms", ActionRead, false},
		{"read with r", sess(Scope{OBAC: true, Access: "r"}), "forms", ActionRead, true},
		{"update needs w", sess(Scope{OBAC: true, Access: "rxd"}), "forms", ActionUpdate, false},
		{"update with w", sess(Scope{OBAC: true, Access: "w"}), "forms", ActionUpdate, true},
		{"delete needs d", sess(Scope{OBAC: true, Access: "rwx"}), "forms", ActionDelete, false},
		{"delete with d", sess(Scope{OBAC: true, Access: "d"}), "forms", ActionDelete, true},
		{"super add bypasses", sess(Scope{OBAC: true}, "add"), "forms", ActionCreate, true},
		{"super read does not cover update", sess(Scope{OBAC: true}, "read"), "forms", ActionUpdate, false},
		{"super write bypasses", sess(Scope{OBAC: true}, "write"), "forms", ActionUpdate, true},
		{"super delete bypasses", sess(Scope{OBAC: true}, "delete"), "forms", ActionDelete, true},
		{"unknown access is denied", sess(Scope{Access: "rwxd"}, "read", "write", "add", "delete"), "forms", Action(9), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.s.CanAccess(tt.class, tt.access))
		})
	}
}

func TestRequiresRBAC(t *testing.T) {
	rbac := map[string]Scope{"forms": {RBAC: true}}
	plain := map[string]Scope{"forms": {}}

	assert.True(t, (&Session{Scopes: rbac}).RequiresRBAC("forms", ActionRead))
	assert.False(t, (&Session{Scopes: plain}).RequiresRBAC("forms", ActionRead))
	assert.False(t, (&Session{Scopes: rbac, Permissions: []string{"read"}}).RequiresRBAC("forms", ActionRead))
	assert.True(t, (&Session{Scopes: rbac, Permissions: []string{"read"}}).RequiresRBAC("forms", ActionUpdate))
	assert.False(t, (&Session{Scopes: rbac}).RequiresRBAC("other", ActionRead))
}

func TestHasLicense(t *testing.T) {
	now := time.Now()
	s := &Session{Licenses: []License{
		{Prod: "active", IssuedAt: now.Add(-time.Hour), ExpiresAt: now.Add(time.Hour)},
		{Prod: "expired", IssuedAt: now.Add(-2 * time.Hour), ExpiresAt: now.Add(-time.Hour)},
		{Prod: "future", IssuedAt: now.Add(time.Hour), ExpiresAt: now.Add(2 * time.Hour)},
		{Prod: "unbounded"},
		{Prod: "no-end", IssuedAt: now.Add(-time.Hour)},
		{Prod: "no-start", ExpiresAt: now.Add(time.Hour)},
		{Prod: "no-start-expired", ExpiresAt: now.Add(-time.Hour)},
	}}

	assert.True(t, s.HasLicense("unbounded"), "zero bounds mean no bound")
	assert.True(t, s.HasLicense("no-end"))
	assert.True(t, s.HasLicense("no-start"))
	assert.False(t, s.HasLicense("no-start-expired"))
	assert.True(t, s.HasLicense("active"))
	assert.False(t, s.HasLicense("expired"))
	assert.False(t, s.HasLicense("future"))
	assert.False(t, s.HasLicense("absent"))
}

func TestExpired(t *testing.T) {
	assert.False(t, (&Session{}).Expired(), "zero ExpiresAt means no expiry")
	assert.False(t, (&Session{ExpiresAt: time.Now().Add(time.Minute)}).Expired())
	assert.True(t, (&Session{ExpiresAt: time.Now().Add(-time.Minute)}).Expired())
}

func TestACLRoles(t *testing.T) {
	s := &Session{UserID: 7, RoleIDs: []int64{1, 2}}

	assert.Equal(t, []int64{7, 1, 2}, s.ACLRoles())
	assert.Equal(t, []int64{1, 2}, s.RoleIDs, "must not alias RoleIDs")
}

func TestSessionFromContext(t *testing.T) {
	_, ok := SessionFromContext(context.Background())
	assert.False(t, ok)

	want := &Session{UserID: 1}
	got, ok := SessionFromContext(ContextWithSession(context.Background(), want))
	assert.True(t, ok)
	assert.Same(t, want, got)
}
