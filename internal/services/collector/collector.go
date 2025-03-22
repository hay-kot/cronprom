// Package collector defines the MetricCollector struct that manages all metrics defined in the configuration.
package collector

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/hay-kot/cronprom/internal/data/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

// MetricCollector manages all metrics defined in the configuration
type MetricCollector struct {
	config     *config.Config
	registry   *prometheus.Registry
	gauges     map[string]*prometheus.GaugeVec
	counters   map[string]*prometheus.CounterVec
	histograms map[string]*prometheus.HistogramVec
	summaries  map[string]*prometheus.SummaryVec
	mutex      sync.RWMutex
}

// NewMetricCollector creates a new metric collector
func NewMetricCollector(cfg *config.Config, registry *prometheus.Registry) (*MetricCollector, error) {
	if cfg == nil {
		return nil, errors.New("config cannot be nil")
	}

	collector := &MetricCollector{
		config:     cfg,
		registry:   registry,
		gauges:     make(map[string]*prometheus.GaugeVec),
		counters:   make(map[string]*prometheus.CounterVec),
		histograms: make(map[string]*prometheus.HistogramVec),
		summaries:  make(map[string]*prometheus.SummaryVec),
	}

	// Register metrics from config
	if err := collector.registerMetrics(); err != nil {
		return nil, err
	}

	return collector, nil
}

// cleanLabels returns a list of labels with fillers for missing labels, labels are assumed
// to be in order.
func (c *MetricCollector) cleanLabels(metricName string, labels map[string]string) (map[string]string, error) {
	const Filler = "<missing>"

	for _, metricCfg := range c.config.Metrics {
		if metricCfg.Name == metricName {
			// Check if all labels are present
			for _, label := range metricCfg.Labels {
				if _, exists := labels[label]; !exists {
					// Fill missing label
					labels[label] = Filler
					log.Info().Str("metric", metricName).Str("label", label).Msg("adding missing label")
				}
			}

			// Remove extra labels
			maps.DeleteFunc(labels, func(key string, _ string) bool {
				toRemove := !slices.Contains(metricCfg.Labels, key)
				if toRemove {
					log.Info().Str("metric", metricName).Str("label", key).Msg("removing extra label")
				}

				return toRemove
			})

			return labels, nil
		}
	}

	return nil, fmt.Errorf("metric '%s' not found", metricName)
}

// registerMetrics creates and registers all metrics defined in the configuration
func (c *MetricCollector) registerMetrics() error {
	for _, metricCfg := range c.config.Metrics {
		if err := c.registerMetric(metricCfg); err != nil {
			return err
		}
	}
	return nil
}

// registerMetric creates and registers a single metric
func (c *MetricCollector) registerMetric(metricCfg config.MetricConfig) error {
	namespace := c.config.Global.Namespace
	metricName := metricCfg.Name

	switch metricCfg.Type {
	case config.MetricTypeGauge:
		opts := prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      metricName,
			Help:      metricCfg.Description,
		}
		gaugeVec := prometheus.NewGaugeVec(opts, metricCfg.Labels)
		if err := c.registry.Register(gaugeVec); err != nil {
			return fmt.Errorf("failed to register gauge '%s': %w", metricName, err)
		}
		c.gauges[metricName] = gaugeVec

	case config.MetricTypeCounter:
		opts := prometheus.CounterOpts{
			Namespace: namespace,
			Name:      metricName,
			Help:      metricCfg.Description,
		}
		counterVec := prometheus.NewCounterVec(opts, metricCfg.Labels)
		if err := c.registry.Register(counterVec); err != nil {
			return fmt.Errorf("failed to register counter '%s': %w", metricName, err)
		}
		c.counters[metricName] = counterVec

	case config.MetricTypeHistogram:
		opts := prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      metricName,
			Help:      metricCfg.Description,
			Buckets:   metricCfg.Buckets,
		}
		histogramVec := prometheus.NewHistogramVec(opts, metricCfg.Labels)
		if err := c.registry.Register(histogramVec); err != nil {
			return fmt.Errorf("failed to register histogram '%s': %w", metricName, err)
		}
		c.histograms[metricName] = histogramVec

	case config.MetricTypeSummary:
		opts := prometheus.SummaryOpts{
			Namespace:  namespace,
			Name:       metricName,
			Help:       metricCfg.Description,
			Objectives: metricCfg.Objectives,
		}
		summaryVec := prometheus.NewSummaryVec(opts, metricCfg.Labels)
		if err := c.registry.Register(summaryVec); err != nil {
			return fmt.Errorf("failed to register summary '%s': %w", metricName, err)
		}
		c.summaries[metricName] = summaryVec

	default:
		return fmt.Errorf("unsupported metric type: %s", metricCfg.Type)
	}

	return nil
}

// GetRegistry returns the Prometheus registry
func (c *MetricCollector) GetRegistry() *prometheus.Registry {
	return c.registry
}

// UpdateGauge updates a gauge metric with the given value and labels
func (c *MetricCollector) UpdateGauge(name string, value float64, labels map[string]string) error {
	c.mutex.RLock()
	gauge, exists := c.gauges[name]
	c.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("gauge metric '%s' not found", name)
	}

	labelsWithFillers, err := c.cleanLabels(name, labels)
	if err != nil {
		return err
	}

	gauge.With(labelsWithFillers).Set(value)
	return nil
}

// IncrementCounter increments a counter metric with the given labels
func (c *MetricCollector) IncrementCounter(name string, labels map[string]string) error {
	return c.IncrementCounterBy(name, 1, labels)
}

// IncrementCounterBy increments a counter metric by the given value with the given labels
func (c *MetricCollector) IncrementCounterBy(name string, value float64, labels map[string]string) error {
	c.mutex.RLock()
	counter, exists := c.counters[name]
	c.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("counter metric '%s' not found", name)
	}

	labelsWithFillers, err := c.cleanLabels(name, labels)
	if err != nil {
		return err
	}

	counter.With(labelsWithFillers).Add(value)
	return nil
}

// ObserveHistogram observes a value in a histogram metric with the given labels
func (c *MetricCollector) ObserveHistogram(name string, value float64, labels map[string]string) error {
	c.mutex.RLock()
	histogram, exists := c.histograms[name]
	c.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("histogram metric '%s' not found", name)
	}

	labelsWithFillers, err := c.cleanLabels(name, labels)
	if err != nil {
		return err
	}

	histogram.With(labelsWithFillers).Observe(value)
	return nil
}

// ObserveSummary observes a value in a summary metric with the given labels
func (c *MetricCollector) ObserveSummary(name string, value float64, labels map[string]string) error {
	c.mutex.RLock()
	summary, exists := c.summaries[name]
	c.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("summary metric '%s' not found", name)
	}

	labelsWithFillers, err := c.cleanLabels(name, labels)
	if err != nil {
		return err
	}

	summary.With(labelsWithFillers).Observe(value)
	return nil
}
