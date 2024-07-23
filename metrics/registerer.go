package metrics

import (
	"fmt"
	"regexp"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

var _ prometheus.Gatherer = (*addPrefixWrapper)(nil)

// addPrefixWrapper wraps a prometheus.Gatherer, and ensures that all metric names are prefixed with `webitel_`.
// metrics with the prefix `webitel_` or `go_` are not modified.
type addPrefixWrapper struct {
	orig prometheus.Gatherer
	reg  *regexp.Regexp
}

func newAddPrefixWrapper(orig prometheus.Gatherer) *addPrefixWrapper {
	return &addPrefixWrapper{
		orig: orig,
		reg:  regexp.MustCompile("^((?:webitel_|go_).*)"),
	}
}

func (g *addPrefixWrapper) Gather() ([]*dto.MetricFamily, error) {
	mf, err := g.orig.Gather()
	if err != nil {
		return nil, err
	}

	names := make(map[string]struct{})

	for i := 0; i < len(mf); i++ {
		m := mf[i]
		if m.Name != nil && !g.reg.MatchString(*m.Name) {
			*m.Name = "webitel_" + *m.Name

			// since we are modifying the name, we need to check for duplicates in the gatherer
			if _, exists := names[*m.Name]; exists {
				return nil, fmt.Errorf("duplicate metric name: %s", *m.Name)
			}
		}

		// keep track of names to detect duplicates
		names[*m.Name] = struct{}{}
	}

	return mf, nil
}
