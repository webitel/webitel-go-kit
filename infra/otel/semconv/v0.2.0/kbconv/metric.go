// Code generated from semantic convention specification. DO NOT EDIT.

// Copyright (c) 2026 Webitel
// SPDX-License-Identifier: MIT

// Package kbconv provides types and functionality for OpenTelemetry semantic
// conventions in the "webitel.kb" namespace.
package kbconv

import (
	"context"

	"github.com/webitel/webitel-go-kit/infra/otel/semconv/internal/metricpool"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

// ErrorTypeAttr is an attribute conforming to the error.type semantic
// conventions. It represents the describes a class of error the operation ended
// with.
type ErrorTypeAttr string

var (
	// ErrorTypeOther is a fallback error value to be used when the instrumentation
	// doesn't define a custom value.
	ErrorTypeOther ErrorTypeAttr = "_OTHER"
)

// ArticleIndexStateAttr is an attribute conforming to the
// webitel.kb.article.index.state semantic conventions. It represents the state
// of the index of an article.
type ArticleIndexStateAttr string

var (
	// ArticleIndexStatePending is the latest version of the article is waiting to
	// be indexed.
	ArticleIndexStatePending ArticleIndexStateAttr = "pending"
	// ArticleIndexStateIndexing is the latest version of the article is being
	// indexed.
	ArticleIndexStateIndexing ArticleIndexStateAttr = "indexing"
	// ArticleIndexStateIndexed is the latest version of the article is searchable.
	ArticleIndexStateIndexed ArticleIndexStateAttr = "indexed"
	// ArticleIndexStateFailed is the latest version of the article could not be
	// indexed.
	ArticleIndexStateFailed ArticleIndexStateAttr = "failed"
)

// ArticleIndexCount is an instrument used to record metric values conforming to
// the "webitel.kb.article.index.count" semantic conventions. It represents the
// number of article indexes that are currently in the state described by the
// `webitel.kb.article.index.state` attribute.
type ArticleIndexCount struct {
	metric.Int64UpDownCounter
}

var newArticleIndexCountOpts = []metric.Int64UpDownCounterOption{
	metric.WithDescription("Number of article indexes that are currently in the state described by the `webitel.kb.article.index.state` attribute."),
	metric.WithUnit("{article}"),
}

// NewArticleIndexCount returns a new ArticleIndexCount instrument.
func NewArticleIndexCount(
	m metric.Meter,
	opt ...metric.Int64UpDownCounterOption,
) (ArticleIndexCount, error) {
	// Check if the meter is nil.
	if m == nil {
		return ArticleIndexCount{noop.Int64UpDownCounter{}}, nil
	}

	if len(opt) == 0 {
		opt = newArticleIndexCountOpts
	} else {
		opt = append(opt, newArticleIndexCountOpts...)
	}

	i, err := m.Int64UpDownCounter(
		"webitel.kb.article.index.count",
		opt...,
	)
	if err != nil {
		return ArticleIndexCount{noop.Int64UpDownCounter{}}, err
	}
	return ArticleIndexCount{i}, nil
}

// Inst returns the underlying metric instrument.
func (m ArticleIndexCount) Inst() metric.Int64UpDownCounter {
	return m.Int64UpDownCounter
}

// Name returns the semantic convention name of the instrument.
func (ArticleIndexCount) Name() string {
	return "webitel.kb.article.index.count"
}

// Unit returns the semantic convention unit of the instrument
func (ArticleIndexCount) Unit() string {
	return "{article}"
}

// Description returns the semantic convention description of the instrument
func (ArticleIndexCount) Description() string {
	return "Number of article indexes that are currently in the state described by the `webitel.kb.article.index.state` attribute."
}

// Add adds incr to the existing count for attrs.
//
// The articleIndexState is the the state of the index of an article.
//
// Each live article has one index, in the state of its latest version. This
// metric MUST be reported by a single node.
func (m ArticleIndexCount) Add(
	ctx context.Context,
	incr int64,
	articleIndexState ArticleIndexStateAttr,
	attrs ...attribute.KeyValue,
) {
	if !m.Int64UpDownCounter.Enabled(ctx) {
		return
	}
	if len(attrs) == 0 {
		m.Int64UpDownCounter.Add(ctx, incr, metric.WithAttributes(
			attribute.String("webitel.kb.article.index.state", string(articleIndexState)),
		))
		return
	}

	o := metricpool.AddOptions()
	defer metricpool.PutAddOptions(o)

	*o = append(
		*o,
		metric.WithAttributes(
			append(
				attrs[:len(attrs):len(attrs)],
				attribute.String("webitel.kb.article.index.state", string(articleIndexState)),
			)...,
		),
	)

	m.Int64UpDownCounter.Add(ctx, incr, *o...)
}

// AddSet adds incr to the existing count for set.
//
// Each live article has one index, in the state of its latest version. This
// metric MUST be reported by a single node.
func (m ArticleIndexCount) AddSet(ctx context.Context, incr int64, set attribute.Set) {
	if !m.Int64UpDownCounter.Enabled(ctx) {
		return
	}
	if set.Len() == 0 {
		m.Int64UpDownCounter.Add(ctx, incr)
		return
	}

	o := metricpool.AddOptions()
	defer metricpool.PutAddOptions(o)

	*o = append(*o, metric.WithAttributeSet(set))
	m.Int64UpDownCounter.Add(ctx, incr, *o...)
}

// ArticleIndexCountObservable is an instrument used to record metric values
// conforming to the "webitel.kb.article.index.count" semantic conventions. It
// represents the number of article indexes that are currently in the state
// described by the `webitel.kb.article.index.state` attribute.
type ArticleIndexCountObservable struct {
	metric.Int64ObservableUpDownCounter
}

var newArticleIndexCountObservableOpts = []metric.Int64ObservableUpDownCounterOption{
	metric.WithDescription("Number of article indexes that are currently in the state described by the `webitel.kb.article.index.state` attribute."),
	metric.WithUnit("{article}"),
}

// NewArticleIndexCountObservable returns a new ArticleIndexCountObservable
// instrument.
func NewArticleIndexCountObservable(
	m metric.Meter,
	opt ...metric.Int64ObservableUpDownCounterOption,
) (ArticleIndexCountObservable, error) {
	// Check if the meter is nil.
	if m == nil {
		return ArticleIndexCountObservable{noop.Int64ObservableUpDownCounter{}}, nil
	}

	if len(opt) == 0 {
		opt = newArticleIndexCountObservableOpts
	} else {
		opt = append(opt, newArticleIndexCountObservableOpts...)
	}

	i, err := m.Int64ObservableUpDownCounter(
		"webitel.kb.article.index.count",
		opt...,
	)
	if err != nil {
		return ArticleIndexCountObservable{noop.Int64ObservableUpDownCounter{}}, err
	}
	return ArticleIndexCountObservable{i}, nil
}

// Inst returns the underlying metric instrument.
func (m ArticleIndexCountObservable) Inst() metric.Int64ObservableUpDownCounter {
	return m.Int64ObservableUpDownCounter
}

// Name returns the semantic convention name of the instrument.
func (ArticleIndexCountObservable) Name() string {
	return "webitel.kb.article.index.count"
}

// Unit returns the semantic convention unit of the instrument
func (ArticleIndexCountObservable) Unit() string {
	return "{article}"
}

// Description returns the semantic convention description of the instrument
func (ArticleIndexCountObservable) Description() string {
	return "Number of article indexes that are currently in the state described by the `webitel.kb.article.index.state` attribute."
}

// AttrArticleIndexState returns a required attribute for the
// "webitel.kb.article.index.state" semantic convention. It represents the state
// of the index of an article.
func (ArticleIndexCountObservable) AttrArticleIndexState(val ArticleIndexStateAttr) attribute.KeyValue {
	return attribute.String("webitel.kb.article.index.state", string(val))
}

// ArticleIndexDuration is an instrument used to record metric values conforming
// to the "webitel.kb.article.index.duration" semantic conventions. It represents
// the duration from an article edit to the new version of the article becoming
// searchable.
type ArticleIndexDuration struct {
	metric.Float64Histogram
}

var newArticleIndexDurationOpts = []metric.Float64HistogramOption{
	metric.WithDescription("Duration from an article edit to the new version of the article becoming searchable."),
	metric.WithUnit("s"),
}

// NewArticleIndexDuration returns a new ArticleIndexDuration instrument.
func NewArticleIndexDuration(
	m metric.Meter,
	opt ...metric.Float64HistogramOption,
) (ArticleIndexDuration, error) {
	// Check if the meter is nil.
	if m == nil {
		return ArticleIndexDuration{noop.Float64Histogram{}}, nil
	}

	if len(opt) == 0 {
		opt = newArticleIndexDurationOpts
	} else {
		opt = append(opt, newArticleIndexDurationOpts...)
	}

	i, err := m.Float64Histogram(
		"webitel.kb.article.index.duration",
		opt...,
	)
	if err != nil {
		return ArticleIndexDuration{noop.Float64Histogram{}}, err
	}
	return ArticleIndexDuration{i}, nil
}

// Inst returns the underlying metric instrument.
func (m ArticleIndexDuration) Inst() metric.Float64Histogram {
	return m.Float64Histogram
}

// Name returns the semantic convention name of the instrument.
func (ArticleIndexDuration) Name() string {
	return "webitel.kb.article.index.duration"
}

// Unit returns the semantic convention unit of the instrument
func (ArticleIndexDuration) Unit() string {
	return "s"
}

// Description returns the semantic convention description of the instrument
func (ArticleIndexDuration) Description() string {
	return "Duration from an article edit to the new version of the article becoming searchable."
}

// Record records val to the current distribution for attrs.
//
// The articleIndexEmbedded is the whether the article version was embedded for
// vector search.
//
// This metric MUST be recorded once per version that becomes searchable, and
// MUST NOT be recorded for a job that makes no version searchable. A negative
// duration, caused by a clock skew between hosts, MUST be recorded as 0.
func (m ArticleIndexDuration) Record(
	ctx context.Context,
	val float64,
	articleIndexEmbedded bool,
	attrs ...attribute.KeyValue,
) {
	if !m.Float64Histogram.Enabled(ctx) {
		return
	}
	if len(attrs) == 0 {
		m.Float64Histogram.Record(ctx, val, metric.WithAttributes(
			attribute.Bool("webitel.kb.article.index.embedded", articleIndexEmbedded),
		))
		return
	}

	o := metricpool.RecordOptions()
	defer metricpool.PutRecordOptions(o)

	*o = append(
		*o,
		metric.WithAttributes(
			append(
				attrs[:len(attrs):len(attrs)],
				attribute.Bool("webitel.kb.article.index.embedded", articleIndexEmbedded),
			)...,
		),
	)

	m.Float64Histogram.Record(ctx, val, *o...)
}

// RecordSet records val to the current distribution for set.
//
// This metric MUST be recorded once per version that becomes searchable, and
// MUST NOT be recorded for a job that makes no version searchable. A negative
// duration, caused by a clock skew between hosts, MUST be recorded as 0.
func (m ArticleIndexDuration) RecordSet(ctx context.Context, val float64, set attribute.Set) {
	if !m.Float64Histogram.Enabled(ctx) {
		return
	}
	if set.Len() == 0 {
		m.Float64Histogram.Record(ctx, val)
		return
	}

	o := metricpool.RecordOptions()
	defer metricpool.PutRecordOptions(o)

	*o = append(*o, metric.WithAttributeSet(set))
	m.Float64Histogram.Record(ctx, val, *o...)
}

// ArticleIndexJobCount is an instrument used to record metric values conforming
// to the "webitel.kb.article.index.job.count" semantic conventions. It
// represents the number of indexing jobs that are currently waiting in the
// broker in the state described by the `webitel.kb.article.index.state`
// attribute.
type ArticleIndexJobCount struct {
	metric.Int64Gauge
}

var newArticleIndexJobCountOpts = []metric.Int64GaugeOption{
	metric.WithDescription("Number of indexing jobs that are currently waiting in the broker in the state described by the `webitel.kb.article.index.state` attribute."),
	metric.WithUnit("{job}"),
}

// NewArticleIndexJobCount returns a new ArticleIndexJobCount instrument.
func NewArticleIndexJobCount(
	m metric.Meter,
	opt ...metric.Int64GaugeOption,
) (ArticleIndexJobCount, error) {
	// Check if the meter is nil.
	if m == nil {
		return ArticleIndexJobCount{noop.Int64Gauge{}}, nil
	}

	if len(opt) == 0 {
		opt = newArticleIndexJobCountOpts
	} else {
		opt = append(opt, newArticleIndexJobCountOpts...)
	}

	i, err := m.Int64Gauge(
		"webitel.kb.article.index.job.count",
		opt...,
	)
	if err != nil {
		return ArticleIndexJobCount{noop.Int64Gauge{}}, err
	}
	return ArticleIndexJobCount{i}, nil
}

// Inst returns the underlying metric instrument.
func (m ArticleIndexJobCount) Inst() metric.Int64Gauge {
	return m.Int64Gauge
}

// Name returns the semantic convention name of the instrument.
func (ArticleIndexJobCount) Name() string {
	return "webitel.kb.article.index.job.count"
}

// Unit returns the semantic convention unit of the instrument
func (ArticleIndexJobCount) Unit() string {
	return "{job}"
}

// Description returns the semantic convention description of the instrument
func (ArticleIndexJobCount) Description() string {
	return "Number of indexing jobs that are currently waiting in the broker in the state described by the `webitel.kb.article.index.state` attribute."
}

// Record records val to the current distribution for attrs.
//
// The articleIndexState is the the state of the index of an article.
//
// A `pending` job waits in the `kb.reindex` queue, a `failed` job in the
// `kb.reindex.dlq` queue. A job that is being processed MUST NOT be counted.
// This metric MUST NOT be reported while the broker is unreachable.
func (m ArticleIndexJobCount) Record(
	ctx context.Context,
	val int64,
	articleIndexState ArticleIndexStateAttr,
	attrs ...attribute.KeyValue,
) {
	if !m.Int64Gauge.Enabled(ctx) {
		return
	}
	if len(attrs) == 0 {
		m.Int64Gauge.Record(ctx, val, metric.WithAttributes(
			attribute.String("webitel.kb.article.index.state", string(articleIndexState)),
		))
		return
	}

	o := metricpool.RecordOptions()
	defer metricpool.PutRecordOptions(o)

	*o = append(
		*o,
		metric.WithAttributes(
			append(
				attrs[:len(attrs):len(attrs)],
				attribute.String("webitel.kb.article.index.state", string(articleIndexState)),
			)...,
		),
	)

	m.Int64Gauge.Record(ctx, val, *o...)
}

// RecordSet records val to the current distribution for set.
//
// A `pending` job waits in the `kb.reindex` queue, a `failed` job in the
// `kb.reindex.dlq` queue. A job that is being processed MUST NOT be counted.
// This metric MUST NOT be reported while the broker is unreachable.
func (m ArticleIndexJobCount) RecordSet(ctx context.Context, val int64, set attribute.Set) {
	if !m.Int64Gauge.Enabled(ctx) {
		return
	}
	if set.Len() == 0 {
		m.Int64Gauge.Record(ctx, val)
		return
	}

	o := metricpool.RecordOptions()
	defer metricpool.PutRecordOptions(o)

	*o = append(*o, metric.WithAttributeSet(set))
	m.Int64Gauge.Record(ctx, val, *o...)
}

// ArticleIndexJobCountObservable is an instrument used to record metric values
// conforming to the "webitel.kb.article.index.job.count" semantic conventions.
// It represents the number of indexing jobs that are currently waiting in the
// broker in the state described by the `webitel.kb.article.index.state`
// attribute.
type ArticleIndexJobCountObservable struct {
	metric.Int64ObservableGauge
}

var newArticleIndexJobCountObservableOpts = []metric.Int64ObservableGaugeOption{
	metric.WithDescription("Number of indexing jobs that are currently waiting in the broker in the state described by the `webitel.kb.article.index.state` attribute."),
	metric.WithUnit("{job}"),
}

// NewArticleIndexJobCountObservable returns a new ArticleIndexJobCountObservable
// instrument.
func NewArticleIndexJobCountObservable(
	m metric.Meter,
	opt ...metric.Int64ObservableGaugeOption,
) (ArticleIndexJobCountObservable, error) {
	// Check if the meter is nil.
	if m == nil {
		return ArticleIndexJobCountObservable{noop.Int64ObservableGauge{}}, nil
	}

	if len(opt) == 0 {
		opt = newArticleIndexJobCountObservableOpts
	} else {
		opt = append(opt, newArticleIndexJobCountObservableOpts...)
	}

	i, err := m.Int64ObservableGauge(
		"webitel.kb.article.index.job.count",
		opt...,
	)
	if err != nil {
		return ArticleIndexJobCountObservable{noop.Int64ObservableGauge{}}, err
	}
	return ArticleIndexJobCountObservable{i}, nil
}

// Inst returns the underlying metric instrument.
func (m ArticleIndexJobCountObservable) Inst() metric.Int64ObservableGauge {
	return m.Int64ObservableGauge
}

// Name returns the semantic convention name of the instrument.
func (ArticleIndexJobCountObservable) Name() string {
	return "webitel.kb.article.index.job.count"
}

// Unit returns the semantic convention unit of the instrument
func (ArticleIndexJobCountObservable) Unit() string {
	return "{job}"
}

// Description returns the semantic convention description of the instrument
func (ArticleIndexJobCountObservable) Description() string {
	return "Number of indexing jobs that are currently waiting in the broker in the state described by the `webitel.kb.article.index.state` attribute."
}

// AttrArticleIndexState returns a required attribute for the
// "webitel.kb.article.index.state" semantic convention. It represents the state
// of the index of an article.
func (ArticleIndexJobCountObservable) AttrArticleIndexState(val ArticleIndexStateAttr) attribute.KeyValue {
	return attribute.String("webitel.kb.article.index.state", string(val))
}

// ArticleIndexJobFailed is an instrument used to record metric values conforming
// to the "webitel.kb.article.index.job.failed" semantic conventions. It
// represents the number of indexing jobs sent to the dead letter queue.
type ArticleIndexJobFailed struct {
	metric.Int64Counter
}

var newArticleIndexJobFailedOpts = []metric.Int64CounterOption{
	metric.WithDescription("Number of indexing jobs sent to the dead letter queue."),
	metric.WithUnit("{job}"),
}

// NewArticleIndexJobFailed returns a new ArticleIndexJobFailed instrument.
func NewArticleIndexJobFailed(
	m metric.Meter,
	opt ...metric.Int64CounterOption,
) (ArticleIndexJobFailed, error) {
	// Check if the meter is nil.
	if m == nil {
		return ArticleIndexJobFailed{noop.Int64Counter{}}, nil
	}

	if len(opt) == 0 {
		opt = newArticleIndexJobFailedOpts
	} else {
		opt = append(opt, newArticleIndexJobFailedOpts...)
	}

	i, err := m.Int64Counter(
		"webitel.kb.article.index.job.failed",
		opt...,
	)
	if err != nil {
		return ArticleIndexJobFailed{noop.Int64Counter{}}, err
	}
	return ArticleIndexJobFailed{i}, nil
}

// Inst returns the underlying metric instrument.
func (m ArticleIndexJobFailed) Inst() metric.Int64Counter {
	return m.Int64Counter
}

// Name returns the semantic convention name of the instrument.
func (ArticleIndexJobFailed) Name() string {
	return "webitel.kb.article.index.job.failed"
}

// Unit returns the semantic convention unit of the instrument
func (ArticleIndexJobFailed) Unit() string {
	return "{job}"
}

// Description returns the semantic convention description of the instrument
func (ArticleIndexJobFailed) Description() string {
	return "Number of indexing jobs sent to the dead letter queue."
}

// Add adds incr to the existing count for attrs.
//
// The errorType is the describes a class of error the operation ended with.
//
// A job MUST be counted once, when it is sent to the dead letter queue. Failed
// attempts that are retried MUST NOT be counted.
func (m ArticleIndexJobFailed) Add(
	ctx context.Context,
	incr int64,
	errorType ErrorTypeAttr,
	attrs ...attribute.KeyValue,
) {
	if !m.Int64Counter.Enabled(ctx) {
		return
	}
	if len(attrs) == 0 {
		m.Int64Counter.Add(ctx, incr, metric.WithAttributes(
			attribute.String("error.type", string(errorType)),
		))
		return
	}

	o := metricpool.AddOptions()
	defer metricpool.PutAddOptions(o)

	*o = append(
		*o,
		metric.WithAttributes(
			append(
				attrs[:len(attrs):len(attrs)],
				attribute.String("error.type", string(errorType)),
			)...,
		),
	)

	m.Int64Counter.Add(ctx, incr, *o...)
}

// AddSet adds incr to the existing count for set.
//
// A job MUST be counted once, when it is sent to the dead letter queue. Failed
// attempts that are retried MUST NOT be counted.
func (m ArticleIndexJobFailed) AddSet(ctx context.Context, incr int64, set attribute.Set) {
	if !m.Int64Counter.Enabled(ctx) {
		return
	}
	if set.Len() == 0 {
		m.Int64Counter.Add(ctx, incr)
		return
	}

	o := metricpool.AddOptions()
	defer metricpool.PutAddOptions(o)

	*o = append(*o, metric.WithAttributeSet(set))
	m.Int64Counter.Add(ctx, incr, *o...)
}

// ArticleIndexJobFailedObservable is an instrument used to record metric values
// conforming to the "webitel.kb.article.index.job.failed" semantic conventions.
// It represents the number of indexing jobs sent to the dead letter queue.
type ArticleIndexJobFailedObservable struct {
	metric.Int64ObservableCounter
}

var newArticleIndexJobFailedObservableOpts = []metric.Int64ObservableCounterOption{
	metric.WithDescription("Number of indexing jobs sent to the dead letter queue."),
	metric.WithUnit("{job}"),
}

// NewArticleIndexJobFailedObservable returns a new
// ArticleIndexJobFailedObservable instrument.
func NewArticleIndexJobFailedObservable(
	m metric.Meter,
	opt ...metric.Int64ObservableCounterOption,
) (ArticleIndexJobFailedObservable, error) {
	// Check if the meter is nil.
	if m == nil {
		return ArticleIndexJobFailedObservable{noop.Int64ObservableCounter{}}, nil
	}

	if len(opt) == 0 {
		opt = newArticleIndexJobFailedObservableOpts
	} else {
		opt = append(opt, newArticleIndexJobFailedObservableOpts...)
	}

	i, err := m.Int64ObservableCounter(
		"webitel.kb.article.index.job.failed",
		opt...,
	)
	if err != nil {
		return ArticleIndexJobFailedObservable{noop.Int64ObservableCounter{}}, err
	}
	return ArticleIndexJobFailedObservable{i}, nil
}

// Inst returns the underlying metric instrument.
func (m ArticleIndexJobFailedObservable) Inst() metric.Int64ObservableCounter {
	return m.Int64ObservableCounter
}

// Name returns the semantic convention name of the instrument.
func (ArticleIndexJobFailedObservable) Name() string {
	return "webitel.kb.article.index.job.failed"
}

// Unit returns the semantic convention unit of the instrument
func (ArticleIndexJobFailedObservable) Unit() string {
	return "{job}"
}

// Description returns the semantic convention description of the instrument
func (ArticleIndexJobFailedObservable) Description() string {
	return "Number of indexing jobs sent to the dead letter queue."
}

// AttrErrorType returns a required attribute for the "error.type" semantic
// convention. It represents the describes a class of error the operation ended
// with.
func (ArticleIndexJobFailedObservable) AttrErrorType(val ErrorTypeAttr) attribute.KeyValue {
	return attribute.String("error.type", string(val))
}

// RerankDuration is an instrument used to record metric values conforming to the
// "webitel.kb.rerank.duration" semantic conventions. It represents the duration
// of rerank calls to a provider.
type RerankDuration struct {
	metric.Float64Histogram
}

var newRerankDurationOpts = []metric.Float64HistogramOption{
	metric.WithDescription("Duration of rerank calls to a provider."),
	metric.WithUnit("s"),
}

// NewRerankDuration returns a new RerankDuration instrument.
func NewRerankDuration(
	m metric.Meter,
	opt ...metric.Float64HistogramOption,
) (RerankDuration, error) {
	// Check if the meter is nil.
	if m == nil {
		return RerankDuration{noop.Float64Histogram{}}, nil
	}

	if len(opt) == 0 {
		opt = newRerankDurationOpts
	} else {
		opt = append(opt, newRerankDurationOpts...)
	}

	i, err := m.Float64Histogram(
		"webitel.kb.rerank.duration",
		opt...,
	)
	if err != nil {
		return RerankDuration{noop.Float64Histogram{}}, err
	}
	return RerankDuration{i}, nil
}

// Inst returns the underlying metric instrument.
func (m RerankDuration) Inst() metric.Float64Histogram {
	return m.Float64Histogram
}

// Name returns the semantic convention name of the instrument.
func (RerankDuration) Name() string {
	return "webitel.kb.rerank.duration"
}

// Unit returns the semantic convention unit of the instrument
func (RerankDuration) Unit() string {
	return "s"
}

// Description returns the semantic convention description of the instrument
func (RerankDuration) Description() string {
	return "Duration of rerank calls to a provider."
}

// Record records val to the current distribution for attrs.
//
// The rerankModel is the the name of the model a rerank request is made to, as
// named by the provider.
//
// The rerankProvider is the the name of the rerank provider.
//
// All additional attrs passed are included in the recorded value.
func (m RerankDuration) Record(
	ctx context.Context,
	val float64,
	rerankModel string,
	rerankProvider string,
	attrs ...attribute.KeyValue,
) {
	if !m.Float64Histogram.Enabled(ctx) {
		return
	}
	if len(attrs) == 0 {
		m.Float64Histogram.Record(ctx, val, metric.WithAttributes(
			attribute.String("webitel.kb.rerank.model", rerankModel),
			attribute.String("webitel.kb.rerank.provider", rerankProvider),
		))
		return
	}

	o := metricpool.RecordOptions()
	defer metricpool.PutRecordOptions(o)

	*o = append(
		*o,
		metric.WithAttributes(
			append(
				attrs[:len(attrs):len(attrs)],
				attribute.String("webitel.kb.rerank.model", rerankModel),
				attribute.String("webitel.kb.rerank.provider", rerankProvider),
			)...,
		),
	)

	m.Float64Histogram.Record(ctx, val, *o...)
}

// RecordSet records val to the current distribution for set.
func (m RerankDuration) RecordSet(ctx context.Context, val float64, set attribute.Set) {
	if !m.Float64Histogram.Enabled(ctx) {
		return
	}
	if set.Len() == 0 {
		m.Float64Histogram.Record(ctx, val)
		return
	}

	o := metricpool.RecordOptions()
	defer metricpool.PutRecordOptions(o)

	*o = append(*o, metric.WithAttributeSet(set))
	m.Float64Histogram.Record(ctx, val, *o...)
}

// AttrErrorType returns an optional attribute for the "error.type" semantic
// convention. It represents the describes a class of error the operation ended
// with.
func (RerankDuration) AttrErrorType(val ErrorTypeAttr) attribute.KeyValue {
	return attribute.String("error.type", string(val))
}
