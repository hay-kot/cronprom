// Package config contains the configuration structures and functions for loading and validating the configuration.
package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the root configuration structure
type Config struct {
	Global  GlobalConfig   `yaml:"global"`
	Metrics []MetricConfig `yaml:"metrics"`
	Web     Web            `yaml:"web"`
}

type Web struct {
	Address string `yaml:"address"`
}

// GlobalConfig contains global settings
type GlobalConfig struct {
	Namespace       string        `yaml:"namespace"`
	RefreshInterval string        `yaml:"refresh_interval"`
	parsedInterval  time.Duration // Used internally after parsing
}

// ParsedRefreshInterval returns the parsed refresh interval
func (g *GlobalConfig) ParsedRefreshInterval() (time.Duration, error) {
	if g.parsedInterval != 0 {
		return g.parsedInterval, nil
	}

	var err error
	g.parsedInterval, err = time.ParseDuration(g.RefreshInterval)
	if err != nil {
		return 0, fmt.Errorf("invalid refresh interval: %w", err)
	}
	return g.parsedInterval, nil
}

// MetricType represents the type of metric
// ENUM(gauge, counter, histogram, summary)
type MetricType string

// MetricConfig represents a single metric configuration
type MetricConfig struct {
	Name         string              `yaml:"name"`
	Description  string              `yaml:"description"`
	Type         MetricType          `yaml:"type"`
	Labels       []string            `yaml:"labels"`
	DefaultValue float64             `yaml:"default_value,omitempty"`
	Buckets      []float64           `yaml:"buckets,omitempty"`    // For histogram
	Objectives   map[float64]float64 `yaml:"objectives,omitempty"` // For summary
}

// Validate checks if the metric configuration is valid
func (m *MetricConfig) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("metric name cannot be empty")
	}

	switch m.Type {
	case MetricTypeGauge, MetricTypeCounter:
		// No specific validation needed
	case MetricTypeHistogram:
		if len(m.Buckets) == 0 {
			return fmt.Errorf("histogram metric '%s' must define buckets", m.Name)
		}
	case MetricTypeSummary:
		if len(m.Objectives) == 0 {
			return fmt.Errorf("summary metric '%s' must define objectives", m.Name)
		}
	default:
		return fmt.Errorf("unknown metric type '%s' for metric '%s'", m.Type, m.Name)
	}

	return nil
}

// LoadConfig loads the configuration from a YAML file
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	config := Config{
		Web: Web{Address: ":8080"},
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate global settings
	if c.Global.Namespace == "" {
		return fmt.Errorf("global namespace cannot be empty")
	}

	if _, err := c.Global.ParsedRefreshInterval(); err != nil {
		return err
	}

	// Validate metrics
	metricNames := make(map[string]bool)
	for i, metric := range c.Metrics {
		if err := metric.Validate(); err != nil {
			return err
		}

		// Check for duplicate metric names
		if metricNames[metric.Name] {
			return fmt.Errorf("duplicate metric name: %s", metric.Name)
		}
		metricNames[metric.Name] = true

		// Store the validated metric back in the slice
		c.Metrics[i] = metric
	}

	return nil
}
