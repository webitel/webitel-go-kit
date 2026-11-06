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
		access Access
		want   bool
	}{
		{"unknown class", sess(Scope{Obac: true, Access: "rwxd"}), "other", Read, false},
		{"obac off allows without flags", sess(Scope{}), "forms", Delete, true},
		{"create needs x", sess(Scope{Obac: true, Access: "rwd"}), "forms", Create, false},
		{"create with x", sess(Scope{Obac: true, Access: "x"}), "forms", Create, true},
		{"read needs r", sess(Scope{Obac: true, Access: "wxd"}), "forms", Read, false},
		{"read with r", sess(Scope{Obac: true, Access: "r"}), "forms", Read, true},
		{"update needs w", sess(Scope{Obac: true, Access: "rxd"}), "forms", Update, false},
		{"update with w", sess(Scope{Obac: true, Access: "w"}), "forms", Update, true},
		{"delete needs d", sess(Scope{Obac: true, Access: "rwx"}), "forms", Delete, false},
		{"delete with d", sess(Scope{Obac: true, Access: "d"}), "forms", Delete, true},
		{"super add bypasses", sess(Scope{Obac: true}, "add"), "forms", Create, true},
		{"super read does not cover update", sess(Scope{Obac: true}, "read"), "forms", Update, false},
		{"super write bypasses", sess(Scope{Obac: true}, "write"), "forms", Update, true},
		{"super delete bypasses", sess(Scope{Obac: true}, "delete"), "forms", Delete, true},
		{"unknown access is denied", sess(Scope{Access: "rwxd"}, "read", "write", "add", "delete"), "forms", Access(9), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.s.CanAccess(tt.class, tt.access))
		})
	}
}

func TestRequiresRBAC(t *testing.T) {
	rbac := map[string]Scope{"forms": {Rbac: true}}
	plain := map[string]Scope{"forms": {}}

	assert.True(t, (&Session{Scopes: rbac}).RequiresRBAC("forms", Read))
	assert.False(t, (&Session{Scopes: plain}).RequiresRBAC("forms", Read))
	assert.False(t, (&Session{Scopes: rbac, Permissions: []string{"read"}}).RequiresRBAC("forms", Read))
	assert.True(t, (&Session{Scopes: rbac, Permissions: []string{"read"}}).RequiresRBAC("forms", Update))
	assert.False(t, (&Session{Scopes: rbac}).RequiresRBAC("other", Read))
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
	s := &Session{UserID: 7, Roles: []int64{1, 2}}

	assert.Equal(t, []int64{7, 1, 2}, s.ACLRoles())
	assert.Equal(t, []int64{1, 2}, s.Roles, "must not alias Roles")
}

func TestSessionFromContext(t *testing.T) {
	_, ok := SessionFromContext(context.Background())
	assert.False(t, ok)

	want := &Session{UserID: 1}
	got, ok := SessionFromContext(ContextWithSession(context.Background(), want))
	assert.True(t, ok)
	assert.Same(t, want, got)
}
