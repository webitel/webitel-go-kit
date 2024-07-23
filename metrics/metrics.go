package metrics

import (
	"context"
	"net/http"
	"runtime"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/webitel/wlog"
)

// ExporterName is used as namespace for exposing prometheus metrics
const ExporterName = "webitel"

type Metrics struct {
	log *wlog.Logger

	cli *addPrefixWrapper
	reg prometheus.Registerer

	srv *http.Server
}

func New(log *wlog.Logger, listen, version, revision, branch string, buildTimestamp int64) *Metrics {
	reg := prometheus.DefaultRegisterer
	ga := newAddPrefixWrapper(prometheus.DefaultGatherer)
	initMetricVars(reg)
	setBuildInformation(reg, version, revision, branch, buildTimestamp)

	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:     listen,
		ErrorLog: log.StdLog(),
		Handler:  mux,
	}

	mux.HandleFunc("/metrics", func(w http.ResponseWriter, req *http.Request) {
		promhttp.HandlerFor(ga, promhttp.HandlerOpts{EnableOpenMetrics: true}).ServeHTTP(w, req)
	})

	return &Metrics{
		log: log,
		cli: ga,
		reg: reg,
		srv: srv,
	}
}

func (m *Metrics) Gatherer() prometheus.Gatherer {
	return m.cli
}

func (m *Metrics) Registerer() prometheus.Registerer {
	return m.reg
}

func (m *Metrics) Serve() error {
	m.log.Info("http listening on", wlog.String("address", m.srv.Addr))

	return m.srv.ListenAndServe()
}

func (m *Metrics) Stop(ctx context.Context) error {
	if err := m.srv.Shutdown(ctx); err != nil {
		return err
	}

	return nil
}

// setBuildInformation sets the build information for this binary
func setBuildInformation(reg prometheus.Registerer, version, revision, branch string, buildTimestamp int64) {
	webitelBuildVersion := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name:      "build_info",
		Help:      "A metric with a constant '1' value labeled by version, revision, branch, and goversion from which Webitel WFM was built",
		Namespace: ExporterName,
	}, []string{"version", "revision", "branch", "goversion"})

	webitelBuildTimestamp := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name:      "build_timestamp",
		Help:      "A metric exposing when the binary was built in epoch",
		Namespace: ExporterName,
	}, []string{"version", "revision", "branch", "goversion"})

	reg.MustRegister(webitelBuildVersion, webitelBuildTimestamp)

	webitelBuildVersion.WithLabelValues(version, revision, branch, runtime.Version()).Set(1)
	webitelBuildTimestamp.WithLabelValues(version, revision, branch, runtime.Version()).Set(float64(buildTimestamp))
}

// SetEnvironmentInformation exposes environment values provided by the operators as an `_info` metric.
// If there are no environment metrics labels configured, this metric will not be exposed.
func SetEnvironmentInformation(reg prometheus.Registerer, labels map[string]string) error {
	if len(labels) == 0 {
		return nil
	}

	webitelEnvironmentInfo := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "environment_info",
		Help:        "A metric with a constant '1' value labeled by environment information about the running instance.",
		Namespace:   ExporterName,
		ConstLabels: labels,
	})

	reg.MustRegister(webitelEnvironmentInfo)
	webitelEnvironmentInfo.Set(1)

	return nil
}

func initMetricVars(reg prometheus.Registerer) {
	reg.MustRegister(Cache, GRPCServer)
}
