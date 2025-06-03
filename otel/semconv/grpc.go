package semconv

import (
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Semantic conventions for attribute keys for gRPC.
const (
	// RPCGRPCStatusCodeKey is convention for numeric status code of a gRPC request.
	RPCGRPCStatusCodeKey = attribute.Key("rpc.grpc.status_code")

	// RPCNameKey Name of message transmitted or received.
	RPCNameKey = attribute.Key("name")

	// RPCMessageTypeKey Type of message transmitted or received.
	RPCMessageTypeKey = attribute.Key("message.type")

	// RPCMessageIDKey Identifier of message transmitted or received.
	RPCMessageIDKey = attribute.Key("message.id")

	// RPCMessageCompressedSizeKey The compressed size of the message
	// transmitted or received in bytes.
	RPCMessageCompressedSizeKey = attribute.Key("message.compressed_size")

	// RPCMessageUncompressedSizeKey The uncompressed size of the message
	// transmitted or received in bytes.
	RPCMessageUncompressedSizeKey = attribute.Key("message.uncompressed_size")
)

// Semantic conventions for common RPC attributes.
var (
	// RPCSystemGRPC Semantic convention for gRPC as the remoting system.
	RPCSystemGRPC = semconv.RPCSystemGRPC

	// RPCNameMessage Semantic convention for a message named message.
	RPCNameMessage = RPCNameKey.String("message")

	// RPCMessageTypeSent Semantic conventions for RPC message types.
	RPCMessageTypeSent     = RPCMessageTypeKey.String("SENT")
	RPCMessageTypeReceived = RPCMessageTypeKey.String("RECEIVED")
)
