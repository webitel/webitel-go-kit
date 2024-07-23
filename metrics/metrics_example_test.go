package metrics_test

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/webitel/wlog"

	"github.com/webitel/webitel-go-kit/metrics"
)

func ExampleNew() {
	ctx := context.Background()
	log := wlog.NewLogger(&wlog.LoggerConfiguration{EnableConsole: true})
	reg := metrics.New(log, "127.0.0.1:9090", "1.2.3", "commitHash", "main", time.Now().UTC().UnixMilli())
	if err := reg.Serve(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			log.Error("stop metrics server", wlog.Err(err))
		}
	}

	defer reg.Stop(ctx)

	metrics.Cache.Request.With(prometheus.Labels{"status": "hit"}).Inc()
}
