// Package correlation carries the domain and user IDs of a request in W3C
// baggage and copies them into log records.
package correlation

import (
	"context"
	"strconv"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	upstream "go.opentelemetry.io/otel/semconv/v1.43.0"

	semconv "github.com/webitel/webitel-go-kit/infra/otel/semconv/v0.3.0"
)

const (
	DomainIDKey = semconv.WebitelDomainIDKey
	UserIDKey   = upstream.UserIDKey
)

// WithIdentity returns a copy of ctx whose baggage carries domainID under
// [DomainIDKey] and userID under [UserIDKey], replacing inbound values. An ID
// that is not positive is removed and not set. Other baggage members are kept.
func WithIdentity(ctx context.Context, domainID, userID int64) context.Context {
	b := baggage.FromContext(ctx)
	b = setID(b, DomainIDKey, domainID)
	b = setID(b, UserIDKey, userID)
	return baggage.ContextWithBaggage(ctx, b)
}

func setID(b baggage.Baggage, key attribute.Key, id int64) baggage.Baggage {
	if id <= 0 {
		return b.DeleteMember(string(key))
	}

	m, err := baggage.NewMemberRaw(string(key), strconv.FormatInt(id, 10))
	if err != nil {
		otel.Handle(err)
		return b.DeleteMember(string(key))
	}

	set, err := b.SetMember(m)
	if err != nil {
		otel.Handle(err)
		return b.DeleteMember(string(key))
	}
	return set
}
