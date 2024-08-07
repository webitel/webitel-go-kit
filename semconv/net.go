package semconv

import (
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

var (
	NetworkPeerAddressKey = semconv.NetworkPeerAddressKey
	NetworkPeerPortKey    = semconv.NetworkPeerPortKey

	ClientAddressKey = semconv.ClientAddressKey
	ClientPortKey    = semconv.ClientPortKey
)

func NetworkPeerAddress(val string) attribute.KeyValue {
	return NetworkPeerAddressKey.String(val)
}

func NetworkPeerPort(val int) attribute.KeyValue {
	return NetworkPeerPortKey.Int(val)
}
