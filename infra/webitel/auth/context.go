package auth

import "context"

type sessionKey struct{}

func ContextWithSession(ctx context.Context, s *Session) context.Context {
	return context.WithValue(ctx, sessionKey{}, s)
}

func SessionFromContext(ctx context.Context) (*Session, bool) {
	s, ok := ctx.Value(sessionKey{}).(*Session)

	return s, ok && s != nil
}

type ruleKey struct{}

func contextWithRule(ctx context.Context, r rule) context.Context {
	return context.WithValue(ctx, ruleKey{}, r)
}

func ruleFromContext(ctx context.Context) (rule, bool) {
	r, ok := ctx.Value(ruleKey{}).(rule)

	return r, ok
}

// RequiresRBAC reports whether the handler must restrict records to ACLRoles for the current method; it returns true without a session or rule in ctx.
func RequiresRBAC(ctx context.Context) bool {
	s, ok := SessionFromContext(ctx)
	r, hasRule := ruleFromContext(ctx)
	if !ok || !hasRule {
		return true
	}

	return s.RequiresRBAC(r.class, r.access)
}
