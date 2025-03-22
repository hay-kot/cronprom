package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hay-kot/cronprom/internal/web"
	"github.com/rs/zerolog/log"
)

type FlagsPush struct {
	URL    string   `json:"url"`
	Name   string   `json:"name"`
	Type   string   `json:"type"`
	Labels []string `json:"labels"`
	Value  float64  `json:"value"`
}

func Push(ctx context.Context, flags FlagsPush) error {
	if !isValidMetricType(flags.Type) {
		return fmt.Errorf("invalid metric type: %s", flags.Type)
	}

	// Parse labels
	labels := make(map[string]string)

	for _, label := range flags.Labels {
		key, val, ok := parseLabel(label)
		if !ok {
			return fmt.Errorf("invalid label format: %s (expected key=value)", label)
		}
		labels[key] = val
	}

	// Create metric update
	update := web.MetricUpdate{
		Name:   flags.Name,
		Type:   flags.Type,
		Value:  flags.Value,
		Labels: labels,
	}

	// Send request
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	return sendMetricUpdate(ctx, httpClient, flags.URL, update)
}

// isValidMetricType checks if the provided metric type is valid
func isValidMetricType(metricType string) bool {
	validTypes := map[string]bool{
		"gauge":     true,
		"counter":   true,
		"histogram": true,
		"summary":   true,
	}
	return validTypes[metricType]
}

// parseLabel parses a label in the format "key=value"
func parseLabel(label string) (string, string, bool) {
	for i, c := range label {
		if c == '=' {
			return label[:i], label[i+1:], true
		}
	}
	return "", "", false
}

// sendMetricUpdate sends the metric update to the API
func sendMetricUpdate(ctx context.Context, client *http.Client, url string, update web.MetricUpdate) error {
	// Marshal the update to JSON
	payload, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("failed to marshal update: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	log.Debug().
		Str("url", url).
		Str("metric", update.Name).
		Str("type", update.Type).
		Float64("value", update.Value).
		Interface("labels", update.Labels).
		Msg("sending metric update")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	log.Info().
		Str("metric", update.Name).
		Str("type", update.Type).
		Float64("value", update.Value).
		Msg("metric update sent successfully")

	return nil
}
