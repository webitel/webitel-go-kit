package semconv

// gRPC framework log attribute keys. grpc-go's global logger (grpclog.LoggerV2)
// is context-free and hands the adapter either fmt operands or a printf
// format+args; pkg/depenlog attaches the originals under these keys so they stay
// queryable alongside the rendered message.
const (
	GRPCFormatKey = "grpc.format"
	GRPCArgsKey   = "grpc.args"
)
