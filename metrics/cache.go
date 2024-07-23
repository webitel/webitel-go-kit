package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	cacheSubsystem = "cache"
	Cache          *cache
)

type cache struct {
	Request          *prometheus.CounterVec
	RefreshRequest   prometheus.Counter
	MissingRecord    prometheus.Counter
	ForcedEviction   prometheus.Counter
	EntriesEvicted   prometheus.Counter
	ShardRequest     *prometheus.CounterVec
	BatchRefreshSize prometheus.Counter
	Size             prometheus.Gauge
}

func init() {
	Cache = &cache{
		Request: NewCounterVecStartingAtZero(prometheus.CounterOpts{
			Namespace: ExporterName,
			Subsystem: cacheSubsystem,
			Name:      "request_total",
			Help:      "Total number of cache requests",
		}, []string{"status"}, map[string][]string{"status": {"hit", "miss"}}),
		RefreshRequest: NewCounterStartingAtZero(prometheus.CounterOpts{
			Namespace: ExporterName,
			Subsystem: cacheSubsystem,
			Name:      "refresh_request_total",
			Help:      "Total number of refresh request results",
		}),
		MissingRecord: NewCounterStartingAtZero(prometheus.CounterOpts{
			Namespace: ExporterName,
			Subsystem: cacheSubsystem,
			Name:      "missing_record_total",
			Help:      "Total number of request that resulted in missing records",
		}),
		ForcedEviction: NewCounterStartingAtZero(prometheus.CounterOpts{
			Namespace: ExporterName,
			Subsystem: cacheSubsystem,
			Name:      "forced_eviction_total",
			Help:      "Total number of force evicted keys when reaches max cache capacity",
		}),
		EntriesEvicted: NewCounterStartingAtZero(prometheus.CounterOpts{
			Namespace: ExporterName,
			Subsystem: cacheSubsystem,
			Name:      "entries_evicted_total",
			Help:      "Total number of evicted keys",
		}),
		ShardRequest: NewCounterVecStartingAtZero(prometheus.CounterOpts{
			Namespace: ExporterName,
			Subsystem: cacheSubsystem,
			Name:      "shard_request_total",
			Help:      "Total number of cache requests split by shard",
		}, []string{"shard"}, map[string][]string{"shard": {"0"}}),
		BatchRefreshSize: NewCounterStartingAtZero(prometheus.CounterOpts{
			Namespace: ExporterName,
			Subsystem: cacheSubsystem,
			Name:      "batch_refresh_size",
			Help:      "Size of the batch refresh",
		}),
		Size: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: ExporterName,
			Subsystem: cacheSubsystem,
			Name:      "size_total",
			Help:      "Size of the cache",
		}),
	}
}

// Describe sends the super-set of all possible descriptors of metrics
// collected by this Collector to the provided channel and returns once
// the last descriptor has been sent.
func (c *cache) Describe(ch chan<- *prometheus.Desc) {
	c.Request.Describe(ch)
	c.RefreshRequest.Describe(ch)
	c.MissingRecord.Describe(ch)
	c.ForcedEviction.Describe(ch)
	c.EntriesEvicted.Describe(ch)
	c.ShardRequest.Describe(ch)
	c.BatchRefreshSize.Describe(ch)
	c.Size.Describe(ch)
}

// Collect is called by the Prometheus registry when collecting
// metrics. The implementation sends each collected metric via the
// provided channel and returns once the last metric has been sent.
func (c *cache) Collect(ch chan<- prometheus.Metric) {
	c.Request.Collect(ch)
	c.RefreshRequest.Collect(ch)
	c.MissingRecord.Collect(ch)
	c.ForcedEviction.Collect(ch)
	c.EntriesEvicted.Collect(ch)
	c.ShardRequest.Collect(ch)
	c.BatchRefreshSize.Collect(ch)
	c.Size.Collect(ch)
}
