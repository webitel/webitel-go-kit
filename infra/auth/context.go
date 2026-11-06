package auth

import "context"

type sessionKey struct{}

// ContextWithSession returns ctx carrying s; tests use it to build a handler context.
func ContextWithSession(ctx context.Context, s *Session) context.Context {
	return context.WithValue(ctx, sessionKey{}, s)
}

// SessionFromContext returns the session the interceptor stored in ctx.
func SessionFromContext(ctx context.Context) (*Session, bool) {
	s, ok := ctx.Value(sessionKey{}).(*Session)

	return s, ok && s != nil
}
