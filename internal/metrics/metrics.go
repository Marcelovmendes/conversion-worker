package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	JobsReceived = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "conversion_jobs_received_total",
		Help: "Total number of jobs received from the queue",
	})

	JobsCompleted = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "conversion_jobs_completed_total",
		Help: "Total number of jobs completed successfully",
	})

	JobsFailed = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "conversion_jobs_failed_total",
		Help: "Total number of jobs that failed",
	}, []string{"reason"})

	JobDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "conversion_job_duration_seconds",
		Help:    "Time spent processing a conversion job",
		Buckets: prometheus.ExponentialBuckets(1, 2, 10),
	})

	JobsInProgress = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "conversion_jobs_in_progress",
		Help: "Number of jobs currently being processed",
	})

	TracksMatched = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "conversion_tracks_matched_total",
		Help: "Total number of tracks matched successfully",
	})

	TracksNotFound = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "conversion_tracks_not_found_total",
		Help: "Total number of tracks not found",
	})
)

func init() {
	prometheus.MustRegister(
		JobsReceived,
		JobsCompleted,
		JobsFailed,
		JobDuration,
		JobsInProgress,
		TracksMatched,
		TracksNotFound,
	)
}

func StartServer(addr string) {
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(addr, nil)
	}()
}
