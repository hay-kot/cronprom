package commands

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/hay-kot/cronprom/internal/data/config"
	"github.com/hay-kot/cronprom/internal/services/collector"
	"github.com/hay-kot/cronprom/internal/web"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

type FlagsServe struct {
	ConfigFile string
	Version    string
	Commit     string
	Date       string
}

func Serve(ctx context.Context, flags FlagsServe) error {
	cfg, err := config.LoadConfig(flags.ConfigFile)
	if err != nil {
		return fmt.Errorf("error loading configuration: %w", err)
	}

	registry := prometheus.NewRegistry()

	coll, err := collector.NewMetricCollector(cfg, registry)
	if err != nil {
		return fmt.Errorf("error initializing metric collector: %w", err)
	}

	metricHandler := web.NewMetricHandler(coll)

	registry.MustRegister(buildInfo)

	buildInfo.WithLabelValues(flags.Version, flags.Commit, flags.Date).Set(1)

	// Set up HTTP routes
	http.HandleFunc("/api/v1/push", metricHandler.PushHandler)
	http.Handle("/metrics", promhttp.HandlerFor(coll.GetRegistry(), promhttp.HandlerOpts{}))
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Start HTTP server
	go func() {
		log.Info().Str("addr", cfg.Web.Address).Msg("starting HTTP server")
		if err := http.ListenAndServe(cfg.Web.Address, nil); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				return
			}
			log.Fatal().Err(err).Msg("failed to start HTTP server")
		}
	}()

	// Wait for termination signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	log.Info().Msgf("Received signal %v, shutting down", sig)
	return nil
}

// buildInfo mostly exists to ensure the /metrics doesn't 404 when you start the application
// when no metrics are provided it will 404
var buildInfo = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "cronprom_build_info",
		Help: "Build information about the application",
	},
	[]string{"version", "commit_hash", "build_time"},
)
