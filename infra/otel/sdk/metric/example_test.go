package metric_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/webitel/webitel-go-kit/infra/otel/sdk/metric"
	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/metric/stdout"
)

func ExampleNewProvider() {
	ctx := context.Background()
	dir, _ := os.MkdirTemp("", "metrics")
	defer func() { _ = os.RemoveAll(dir) }()
	path := filepath.Join(dir, "metrics.json")

	mp, err := metric.NewProvider(ctx, "file:"+path)
	if err != nil {
		panic(err)
	}
	requests, _ := mp.Meter("example").Int64Counter("requests")
	requests.Add(ctx, 1)
	_ = mp.Shutdown(ctx) // exports what is pending

	out, _ := os.ReadFile(path)
	fmt.Println(strings.Contains(string(out), `"Name":"requests"`))
	// Output: true
}
