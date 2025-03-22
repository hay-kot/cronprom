package config

import (
	"fmt"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

// MetricFactory creates Prometheus metrics from configurations
type MetricFactory struct {
	namespace string
	registry  *prometheus.Registry
}

// NewMetricFactory creates a new metric factory
func NewMetricFactory(namespace string, registry *prometheus.Registry) *MetricFactory {
	return &MetricFactory{
		namespace: namespace,
		registry:  registry,
	}
}

// CreateMetric creates a Prometheus metric from a metric configuration
func (f *MetricFactory) CreateMetric(metricConfig MetricConfig) (any, error) {
	metricName := sanitizeMetricName(metricConfig.Name)
	fullName := prometheus.BuildFQName(f.namespace, "", metricName)

	// Create label names array
	labelNames := make([]string, len(metricConfig.Labels))
	copy(labelNames, metricConfig.Labels)

	var metric any
	var err error

	switch metricConfig.Type {
	case MetricTypeGauge:
		opts := prometheus.GaugeOpts{
			Name: fullName,
			Help: metricConfig.Description,
		}

		if len(labelNames) == 0 {
			// Simple gauge without labels
			gauge := prometheus.NewGauge(opts)
			if metricConfig.DefaultValue != 0 {
				gauge.Set(metricConfig.DefaultValue)
			}
			err = f.registry.Register(gauge)
			metric = gauge
		} else {
			// Vector gauge with labels
			gauge := prometheus.NewGaugeVec(opts, labelNames)
			err = f.registry.Register(gauge)
			metric = gauge
		}

	case MetricTypeCounter:
		opts := prometheus.CounterOpts{
			Name: fullName,
			Help: metricConfig.Description,
		}

		if len(labelNames) == 0 {
			// Simple counter without labels
			counter := prometheus.NewCounter(opts)
			err = f.registry.Register(counter)
			metric = counter
		} else {
			// Vector counter with labels
			counter := prometheus.NewCounterVec(opts, labelNames)
			err = f.registry.Register(counter)
			metric = counter
		}

	case MetricTypeHistogram:
		opts := prometheus.HistogramOpts{
			Name:    fullName,
			Help:    metricConfig.Description,
			Buckets: metricConfig.Buckets,
		}

		if len(labelNames) == 0 {
			// Simple histogram without labels
			histogram := prometheus.NewHistogram(opts)
			err = f.registry.Register(histogram)
			metric = histogram
		} else {
			// Vector histogram with labels
			histogram := prometheus.NewHistogramVec(opts, labelNames)
			err = f.registry.Register(histogram)
			metric = histogram
		}

	case MetricTypeSummary:
		objectives := metricConfig.Objectives
		if objectives == nil {
			objectives = map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001}
		}

		opts := prometheus.SummaryOpts{
			Name:       fullName,
			Help:       metricConfig.Description,
			Objectives: objectives,
		}

		if len(labelNames) == 0 {
			// Simple summary without labels
			summary := prometheus.NewSummary(opts)
			err = f.registry.Register(summary)
			metric = summary
		} else {
			// Vector summary with labels
			summary := prometheus.NewSummaryVec(opts, labelNames)
			err = f.registry.Register(summary)
			metric = summary
		}

	default:
		return nil, fmt.Errorf("unsupported metric type: %s", metricConfig.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("error registering metric '%s': %w", metricConfig.Name, err)
	}

	return metric, nil
}

// sanitizeMetricName ensures the metric name follows Prometheus naming conventions
func sanitizeMetricName(name string) string {
	// Replace any non-alphanumeric characters with underscores
	sanitized := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return '_'
	}, name)

	// Ensure the name doesn't start with a digit
	if len(sanitized) > 0 && sanitized[0] >= '0' && sanitized[0] <= '9' {
		sanitized = "_" + sanitized
	}

	return sanitized
}
