package tracker

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	msgsProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Subsystem: "airtrack",
		Name:      "messages_total",
		Help:      "The total number of processed messages",
	})
	msgsFiltered = promauto.NewCounter(prometheus.CounterOpts{
		Subsystem: "airtrack",
		Name:      "messages_filtered",
		Help:      "The total number of filtered messages",
	})
	aircraftCountVec = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: "airtrack",
			Name:      "sightings",
			Help:      "Number of active sightings",
		},
		[]string{},
	)
	inflightMsgVec = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: "airtrack",
			Name:      "messages_inflight",
			Help:      "Number of jobs inflight",
		},
		[]string{},
	)
	filterDurations = promauto.NewSummary(prometheus.SummaryOpts{
		Subsystem:  "airtrack",
		Name:       "filter_evaluation_duration_microseconds",
		Help:       "Filter evaluation latencies in seconds",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	})
	msgDurations = promauto.NewSummary(prometheus.SummaryOpts{
		Subsystem:  "airtrack",
		Name:       "message_duration_microseconds",
		Help:       "Request processing latencies in seconds",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	})
	//filterEvalDurations = promauto.NewSummary(prometheus.SummaryOpts{
	//	Subsystem:  "airtrack",
	//	Interface:       "filter_evaluation_durations",
	//	Help:       "Redis requests latencies in seconds",
	//	Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	//})
)
