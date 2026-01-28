package main

import (
	"sync"

	"github.com/d0ugal/promexporter/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

// Topic filtering maps
var (
	// ignoreKeyMetrics lists topics that should be ignored
	ignoreKeyMetrics = map[string]string{
		"$SYS/broker/timestamp":        "The timestamp at which this particular build of the broker was made. Static.",
		"$SYS/broker/version":          "The version of the broker. Static.",
		"$SYS/broker/clients/active":   "deprecated in favour of $SYS/broker/clients/connected",
		"$SYS/broker/clients/inactive": "deprecated in favour of $SYS/broker/clients/disconnected",
	}

	// counterKeyMetrics lists topics that should be treated as counters
	counterKeyMetrics = map[string]string{
		"$SYS/broker/bytes/received":            "The total number of bytes received since the broker started.",
		"$SYS/broker/bytes/sent":                "The total number of bytes sent since the broker started.",
		"$SYS/broker/messages/received":         "The total number of messages of any type received since the broker started.",
		"$SYS/broker/messages/sent":             "The total number of messages of any type sent since the broker started.",
		"$SYS/broker/publish/bytes/received":    "The total number of PUBLISH bytes received since the broker started.",
		"$SYS/broker/publish/bytes/sent":        "The total number of PUBLISH bytes sent since the broker started.",
		"$SYS/broker/publish/messages/received": "The total number of PUBLISH messages received since the broker started.",
		"$SYS/broker/publish/messages/sent":     "The total number of PUBLISH messages sent since the broker started.",
		"$SYS/broker/publish/messages/dropped":  "The total number of PUBLISH messages that have been dropped due to inflight/queuing limits.",
		"$SYS/broker/uptime":                    "The total number of seconds since the broker started.",
		"$SYS/broker/clients/maximum":           "The maximum number of clients connected simultaneously since the broker started",
		"$SYS/broker/clients/total":             "The total number of clients connected since the broker started.",
	}
)

// MosquittoMetrics manages all Prometheus metrics for the Mosquitto exporter
type MosquittoMetrics struct {
	registry       *metrics.Registry
	counterMetrics map[string]*MosquittoCounter
	gaugeMetrics   map[string]prometheus.Gauge
	mu             sync.RWMutex
}

// NewMosquittoMetrics creates a new metrics registry
func NewMosquittoMetrics() *MosquittoMetrics {
	registry := metrics.NewRegistry("mosquitto_exporter_info")

	return &MosquittoMetrics{
		registry:       registry,
		counterMetrics: make(map[string]*MosquittoCounter),
		gaugeMetrics:   make(map[string]prometheus.Gauge),
	}
}

// GetRegistry returns the underlying Prometheus registry
func (mm *MosquittoMetrics) GetRegistry() *metrics.Registry {
	return mm.registry
}

// ShouldIgnoreTopic returns true if the topic should be ignored
func (mm *MosquittoMetrics) ShouldIgnoreTopic(topic string) bool {
	_, ok := ignoreKeyMetrics[topic]
	return ok
}

// IsCounterTopic returns true if the topic should be treated as a counter
func (mm *MosquittoMetrics) IsCounterTopic(topic string) bool {
	_, ok := counterKeyMetrics[topic]
	return ok
}

// GetOrCreateCounter gets or creates a counter metric for the given topic
func (mm *MosquittoMetrics) GetOrCreateCounter(topic, help string) *MosquittoCounter {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if counter, ok := mm.counterMetrics[topic]; ok {
		return counter
	}

	// Create new counter
	counter := NewMosquittoCounter(prometheus.NewDesc(
		topic,
		help,
		[]string{},
		prometheus.Labels{},
	))

	mm.counterMetrics[topic] = counter
	mm.registry.GetRegistry().MustRegister(counter)

	// Add metric info for web UI
	mm.registry.AddMetricInfo(topic, help, []string{})

	return counter
}

// GetOrCreateGauge gets or creates a gauge metric for the given topic
func (mm *MosquittoMetrics) GetOrCreateGauge(topic, help string) prometheus.Gauge {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if gauge, ok := mm.gaugeMetrics[topic]; ok {
		return gauge
	}

	// Create new gauge
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: topic,
		Help: help,
	})

	mm.gaugeMetrics[topic] = gauge
	mm.registry.GetRegistry().MustRegister(gauge)

	// Add metric info for web UI
	mm.registry.AddMetricInfo(topic, help, []string{})

	return gauge
}

// SetCounterValue sets the value of a counter metric
func (mm *MosquittoMetrics) SetCounterValue(topic string, value float64) {
	help := counterKeyMetrics[topic]
	if help == "" {
		help = topic
	}

	counter := mm.GetOrCreateCounter(topic, help)
	counter.Set(value)
}

// SetGaugeValue sets the value of a gauge metric
func (mm *MosquittoMetrics) SetGaugeValue(topic string, value float64) {
	gauge := mm.GetOrCreateGauge(topic, topic)
	gauge.Set(value)
}
