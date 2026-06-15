package main

import (
	"context"
	"net/http"
	"time"

	gokitlog "github.com/webitel/webitel-go-kit/pkg/depenlog"
)

func main() {
	l, err := gokitlog.New(gokitlog.Config{Level: "info", JSON: true, Console: true})
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/hello", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hi\n"))
	})

	srv := &http.Server{
		Addr: "127.0.0.1:8080",
		// Middleware logs method/path/status/duration for every request, using
		// the request context so trace_id/span_id are attached when present.
		Handler: gokitlog.Middleware(l)(mux),
		// ErrorLog pipes net/http's internal errors into the unified logger.
		ErrorLog: gokitlog.ErrorLog(l),
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			l.Error("http server failed", "err", err)
		}
	}()

	// Self-request to produce one access-log line, then shut down — keeps the
	// example runnable and self-terminating.
	time.Sleep(100 * time.Millisecond)
	if resp, err := http.Get("http://127.0.0.1:8080/hello"); err == nil {
		_ = resp.Body.Close()
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
