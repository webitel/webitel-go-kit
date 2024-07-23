package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	grpcServerSubsystem = "grpc_server"

	GRPCServer *grpcServer
)

type grpcServer struct {
	HandledCounter   *prometheus.CounterVec
	HandledHistogram *prometheus.HistogramVec
}

func init() {
	GRPCServer = &grpcServer{
		HandledCounter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: ExporterName,
			Subsystem: grpcServerSubsystem,
			Name:      "handled_total",
			Help:      "Total number of RPCs completed on the server, regardless of success or failure.",
		}, []string{"grpc_type", "grpc_service", "grpc_method", "grpc_code"}),
		HandledHistogram: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: ExporterName,
			Subsystem: grpcServerSubsystem,
			Name:      "handling_seconds",
			Help:      "Histogram of response latency (seconds) of gRPC that had been application-level handled by the server.",
			Buckets:   prometheus.DefBuckets,
		}, []string{"grpc_type", "grpc_service", "grpc_method"}),
	}
}

// Describe sends the super-set of all possible descriptors of metrics
// collected by this Collector to the provided channel and returns once
// the last descriptor has been sent.
func (g *grpcServer) Describe(ch chan<- *prometheus.Desc) {
	g.HandledCounter.Describe(ch)
	g.HandledHistogram.Describe(ch)

}

// Collect is called by the Prometheus registry when collecting
// metrics. The implementation sends each collected metric via the
// provided channel and returns once the last metric has been sent.
func (g *grpcServer) Collect(ch chan<- prometheus.Metric) {
	g.HandledCounter.Collect(ch)
	g.HandledHistogram.Collect(ch)
}
