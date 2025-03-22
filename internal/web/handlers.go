// Package server contains the HTTP layer for the application
package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hay-kot/cronprom/internal/data/config"
	"github.com/hay-kot/cronprom/internal/services/collector"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricHandler handles metric update requests
type MetricHandler struct {
	collector *collector.MetricCollector
}

// NewMetricHandler creates a new metric handler
func NewMetricHandler(collector *collector.MetricCollector) *MetricHandler {
	return &MetricHandler{
		collector: collector,
	}
}

// MetricUpdate represents a metric update request
type MetricUpdate struct {
	Name   string            `json:"name"`
	Type   string            `json:"type"`
	Value  float64           `json:"value"`
	Labels map[string]string `json:"labels"`
}

// PushHandler handles requests to update metrics
func (h *MetricHandler) PushHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow POST method
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse JSON
	var update MetricUpdate
	if err := json.Unmarshal(body, &update); err != nil {
		http.Error(w, "Error parsing JSON", http.StatusBadRequest)
		return
	}

	// Validate the update
	if update.Name == "" {
		http.Error(w, "Metric name is required", http.StatusBadRequest)
		return
	}

	metricType, err := config.ParseMetricType(update.Type)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Process the update based on metric type
	var updateErr error
	switch metricType {
	case config.MetricTypeGauge:
		updateErr = h.collector.UpdateGauge(update.Name, update.Value, update.Labels)
	case config.MetricTypeCounter:
		updateErr = h.collector.IncrementCounterBy(update.Name, update.Value, update.Labels)
	case config.MetricTypeHistogram:
		updateErr = h.collector.ObserveHistogram(update.Name, update.Value, update.Labels)
	case config.MetricTypeSummary:
		updateErr = h.collector.ObserveSummary(update.Name, update.Value, update.Labels)
	default:
		http.Error(w, fmt.Sprintf("Unsupported metric type: %s", update.Type), http.StatusBadRequest)
		return
	}

	if updateErr != nil {
		http.Error(w, updateErr.Error(), http.StatusBadRequest)
		return
	}

	// Return success
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"success"}`))
}

// PrometheusHandler exposes metrics in Prometheus format
func (h *MetricHandler) PrometheusHandler(w http.ResponseWriter, r *http.Request) {
	registry := h.collector.GetRegistry()
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	handler.ServeHTTP(w, r)
}
