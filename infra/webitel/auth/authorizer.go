package auth

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	"github.com/webitel/webitel-go-kit/pkg/errors"
)

// Authorizer authenticates a call and checks it against the generated WebitelAPI map.
type Authorizer struct {
	client *Client
	rules  rules
}

// NewAuthorizer parses the generated WebitelAPI variable.
// It fails if no method in it can be authorized.
func NewAuthorizer(c *Client, webitelAPI any) (*Authorizer, error) {
	if c == nil {
		return nil, fmt.Errorf("auth: nil Client")
	}

	r, err := parseRules(webitelAPI)
	if err != nil {
		return nil, err
	}

	return &Authorizer{client: c, rules: r}, nil
}

// Authorize has the shape of go-grpc-middleware's AuthFunc: the session of the
// incoming token goes into the returned context, with the rule of the method, if the licenses and obac of the method allow it.
func (a *Authorizer) Authorize(ctx context.Context) (context.Context, error) {
	method, ok := grpc.Method(ctx)
	if !ok {
		return nil, ErrMethodUnknown
	}

	r, ok := a.rules.lookup(method)
	if !ok {
		// Fail closed on methods the map does not describe.
		return nil, ErrMethodUnknown
	}

	ctx, err := a.client.Authenticate(ctx)
	if err != nil {
		return nil, err
	}
	s, _ := SessionFromContext(ctx)

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

	return contextWithRule(ctx, r), nil
}
