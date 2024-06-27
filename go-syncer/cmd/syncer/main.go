package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-logr/logr"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"

	"github.com/ilaif/athena-cycle/syncer/internal/config"
	"github.com/ilaif/athena-cycle/syncer/internal/github"
	"github.com/ilaif/athena-cycle/syncer/internal/logging"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return errors.Wrap(err, "failed to load config")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log, err := logging.NewLogger()
	if err != nil {
		return errors.Wrap(err, "failed to create logger")
	}
	ctx = logr.NewContext(ctx, log)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		log.Info("Received termination signal, shutting down...")
		cancel()
	}()

	pgClient, err := sqlx.Connect("pgx", cfg.PgURL)
	if err != nil {
		return errors.Wrap(err, "failed to connect to database")
	}
	defer pgClient.Close()

	if err := pgClient.Ping(); err != nil {
		return errors.Wrap(err, "failed to ping database")
	}

	c := cron.New(
		cron.WithLogger(log),
		cron.WithChain(
			cron.Recover(log),
		),
	)
	// Initial sync
	if err := sync(ctx, pgClient, cfg); err != nil {
		return errors.Wrap(err, "failed to sync")
	}
	if _, err := c.AddFunc("@every 10m", func() {
		if err := sync(ctx, pgClient, cfg); err != nil {
			log.Error(err, "Failed to sync")
		}
	}); err != nil {
		return errors.Wrap(err, "failed to add sync job to cron")
	}
	c.Start()

	<-ctx.Done()
	c.Stop()

	log.Info("Syncer has shut down gracefully")
	return nil
}

func sync(ctx context.Context, db *sqlx.DB, cfg *config.Config) error {
	if err := github.Sync(ctx, db, cfg.GitHubRepositories, cfg.GitHubTokens); err != nil {
		return errors.Wrap(err, "failed to sync repositories")
	}
	return nil
}
