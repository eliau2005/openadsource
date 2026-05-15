// Package metrics holds the Prometheus collectors emitted by both the
// adserver and the worker binaries, plus a Handler() that wraps
// promhttp.Handler for use as an http.HandlerFunc.
//
// The metric namespace is oas_* (OpenAdSource). Cardinality is bounded:
// labels are limited to small, enumerable sets (route patterns, event
// names, result codes — never raw URLs or user IDs).
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// HTTP-side collectors.
var (
	// HTTPRequestsTotal counts every adserver request, labeled by chi
	// route pattern (not raw URL) + HTTP status code.
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "oas_http_requests_total",
		Help: "Adserver HTTP requests by route pattern and status code.",
	}, []string{"route", "code"})

	// HTTPRequestDuration is the end-to-end request duration histogram,
	// keyed off chi route pattern. Buckets cover sub-millisecond hot-path
	// wins up to multi-second tail cases.
	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "oas_http_request_duration_seconds",
		Help:    "Adserver HTTP request duration.",
		Buckets: []float64{0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
	}, []string{"route"})
)

// /vast composition + decision-engine reasons.
var (
	VASTResponsesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "oas_vast_responses_total",
		Help: "VAST response composition (inline vs. empty no-fill).",
	}, []string{"type"})

	BudgetRejectionsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "oas_budget_rejections_total",
		Help: "Candidate ads dropped by the campaign-budget enforcer.",
	})

	FreqRejectionsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "oas_freq_rejections_total",
		Help: "Candidate ads dropped by the per-user frequency cap.",
	})
)

// Registry snapshot.
var (
	SnapshotAds = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "oas_snapshot_ads",
		Help: "Active ads in the current in-memory snapshot.",
	})
	SnapshotLoadDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "oas_snapshot_load_duration_seconds",
		Help:    "Registry snapshot load duration.",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 14),
	})
)

// /track event ingestion.
var (
	TrackEventsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "oas_track_events_total",
		Help: "Inbound /track events by event name and status.",
	}, []string{"event", "status"})
)

// Worker-side collectors.
var (
	WorkerTickDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "oas_worker_tick_duration_seconds",
		Help:    "Worker tick end-to-end duration.",
		Buckets: prometheus.ExponentialBuckets(0.01, 2, 10),
	})
	WorkerTicksTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "oas_worker_ticks_total",
		Help: "Worker tick outcomes.",
	}, []string{"result"})
	WorkerDrainedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "oas_worker_drained_total",
		Help: "Events drained from Redis into Postgres daily_stats.",
	}, []string{"event"})
	WorkerCampaignsPausedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "oas_worker_campaigns_paused_total",
		Help: "Campaigns paused on budget exhaustion by the worker.",
	})
)

// Handler returns the http.Handler that serves the Prometheus exposition
// format. Mount at /metrics on whichever port the operator wants to
// scrape from.
func Handler() http.Handler { return promhttp.Handler() }
