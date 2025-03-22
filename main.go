package main

import (
	"context"
	"fmt"
	"os"

	"github.com/hay-kot/cronprom/internal/commands"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

var (
	// Build information. Populated at build-time via -ldflags flag.
	version = "dev"
	commit  = "HEAD"
	date    = "now"
)

func build() string {
	short := commit
	if len(commit) > 7 {
		short = commit[:7]
	}

	return fmt.Sprintf("%s (%s) %s", version, short, date)
}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	app := &cli.Command{
		Name:    "cronprom",
		Usage:   ``,
		Version: build(),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "log-level",
				Usage:   "log level (debug, info, warn, error, fatal, panic)",
				Sources: cli.EnvVars("LOG_FORMAT"),
				Value:   "info",
			},
		},
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			level, err := zerolog.ParseLevel(c.String("log-level"))
			if err != nil {
				return ctx, fmt.Errorf("failed to parse log level: %w", err)
			}

			log.Logger = log.Level(level)

			return ctx, nil
		},
		Commands: []*cli.Command{
			{
				Name:  "push",
				Usage: "push metrics to cronmon",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "url",
						Usage:    "URL of the cronprom API (e.g., http://localhost:8080/api/v1/push)",
						Required: true,
						Sources:  cli.EnvVars("CRONPROM_URL"),
					},
					&cli.StringFlag{
						Name:     "name",
						Usage:    "Name of the metric to update",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "type",
						Usage:    "Type of metric (gauge, counter, histogram, summary)",
						Required: true,
					},
					&cli.FloatFlag{
						Name:     "value",
						Usage:    "Value to update the metric with",
						Required: true,
					},
					&cli.StringSliceFlag{
						Name:  "label",
						Usage: "Label in the format key=value (can be specified multiple times)",
					},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					return commands.Push(ctx, commands.FlagsPush{
						URL:    c.String("url"),
						Name:   c.String("name"),
						Type:   c.String("type"),
						Labels: c.StringSlice("label"),
						Value:  c.Float("value"),
					})
				},
			},
			{
				Name:  "serve",
				Usage: "serve the http backup for cronmon",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "config-path",
						Usage:    "config path",
						Sources:  cli.EnvVars("CRONPROM_CONFIG_PATH"),
						Required: true,
					},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					return commands.Serve(ctx, commands.FlagsServe{
						ConfigFile: c.String("config-path"),
						Version:    version,
						Commit:     commit,
						Date:       date,
					})
				},
			},
		},
	}

	ctx := context.Background()

	if err := app.Run(ctx, os.Args); err != nil {
		log.Fatal().Err(err).Msg("failed to run cronprom")
	}
}
