package metrics

import (
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"

	"github.com/webitel/webitel-go-kit/internal"
)

func TestGathererPrefixWrapper_Gather(t *testing.T) {
	orig := &mockGatherer{}
	g := newAddPrefixWrapper(orig)

	t.Run("metrics with webitel and go prefix are not modified", func(t *testing.T) {
		originalMF := []*dto.MetricFamily{
			{Name: internal.ValueToPTR[string]("webitel_metric1")},
			{Name: internal.ValueToPTR[string]("metric2")},
			{Name: internal.ValueToPTR[string]("go_metric1")},
		}

		orig.GatherFunc = func() ([]*dto.MetricFamily, error) {
			return originalMF, nil
		}

		expectedMF := []*dto.MetricFamily{
			{Name: internal.ValueToPTR[string]("webitel_metric1")},
			{Name: internal.ValueToPTR[string]("webitel_metric2")},
			{Name: internal.ValueToPTR[string]("go_metric1")},
		}

		mf, err := g.Gather()
		require.NoError(t, err)
		require.Equal(t, expectedMF, mf)
	})

	t.Run("duplicate metrics result in an error", func(t *testing.T) {
		originalMF := []*dto.MetricFamily{
			{Name: internal.ValueToPTR[string]("webitel_metric1")},
			{Name: internal.ValueToPTR[string]("metric1")},
		}

		orig.GatherFunc = func() ([]*dto.MetricFamily, error) {
			return originalMF, nil
		}

		_, err := g.Gather()
		require.Error(t, err)
	})
}

type mockGatherer struct {
	GatherFunc func() ([]*dto.MetricFamily, error)
}

func (m *mockGatherer) Gather() ([]*dto.MetricFamily, error) {
	return m.GatherFunc()
}
