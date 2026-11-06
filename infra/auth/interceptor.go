package auth

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/webitel/webitel-go-kit/pkg/errors"
)

// Methods that never need a session; reflection is opt-in, it would list every method.
var defaultPublic = []string{"/grpc.health."}

type interceptorOptions struct {
	public []string
}

// InterceptorOption configures NewInterceptor.
type InterceptorOption func(*interceptorOptions)

// WithPublicMethods lets the services under these prefixes, such as
// "/webitel.audit.Public/", through without a token.
func WithPublicMethods(prefixes ...string) InterceptorOption {
	return func(o *interceptorOptions) { o.public = append(o.public, prefixes...) }
}

// Interceptor authorizes gRPC calls against the generated WebitelAPI map.
type Interceptor struct {
	authorizer Authorizer
	rules      rules
	public     []string
}

// NewInterceptor builds the interceptor from the generated WebitelAPI variable.
// It fails if no method in it can be authorized.
func NewInterceptor(a Authorizer, webitelAPI any, opts ...InterceptorOption) (*Interceptor, error) {
	r, err := parseRules(webitelAPI)
	if err != nil {
		return nil, err
	}

	var o interceptorOptions
	for _, opt := range opts {
		opt(&o)
	}
	// "/pkg.Pub" would also open "/pkg.PubAdmin/...", and "/" would open everything.
	for _, p := range o.public {
		if len(p) < 3 || p[0] != '/' || p[len(p)-1] != '/' || strings.Count(p, "/") != 2 {
			return nil, fmt.Errorf("auth: public method prefix %q must look like \"/pkg.Service/\"", p)
		}
	}

	return &Interceptor{authorizer: a, rules: r, public: append(slices.Clone(defaultPublic), o.public...)}, nil
}

// Unary authorizes unary calls.
func (i *Interceptor) Unary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		ctx, err := i.authorize(ctx, info.FullMethod)
		if err != nil {
			return nil, err
		}

		return handler(ctx, req)
	}
}

// Stream authorizes streaming calls once, when the stream opens.
func (i *Interceptor) Stream() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx, err := i.authorize(ss.Context(), info.FullMethod)
		if err != nil {
			// pkg/interceptors has no stream error interceptor, so the code is set here.
			return status.Error(errors.Code(err), err.Error())
		}

		return handler(srv, &sessionStream{ServerStream: ss, ctx: ctx})
	}
}

// sessionStream gives the handler the context that carries the Session.
type sessionStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *sessionStream) Context() context.Context { return s.ctx }

func (i *Interceptor) authorize(ctx context.Context, fullMethod string) (context.Context, error) {
	for _, p := range i.public {
		if strings.HasPrefix(fullMethod, p) {
			return ctx, nil
		}
	}

	r, ok := i.rules.lookup(fullMethod)
	if !ok {
		// Fail closed on methods the map does not describe.
		return nil, ErrMethodUnknown
	}

	md, _ := metadata.FromIncomingContext(ctx)
	token := ""
	if v := md.Get(TokenHeader); len(v) > 0 {
		token = v[0]
	}
	if token == "" {
		return nil, ErrTokenMissing
	}

	s, err := i.authorizer.Authorize(ctx, token)
	if err != nil {
		return nil, err
	}
	if s == nil || s.Expired() {
		return nil, ErrSessionExpired
	}

	var missing []string
	for _, l := range r.licenses {
		if !s.HasLicense(l) {
			missing = append(missing, l)
		}
	}
	if len(missing) > 0 {
		return nil, errors.Wrap(ErrLicense, errors.WithValue(ValueLicenses, missing))
	}

	if !s.CanAccess(r.class, r.access) {
		return nil, errors.Wrap(ErrPermission, errors.WithValue(ValueClass, r.class))
	}

	return ContextWithSession(ctx, s), nil
}
