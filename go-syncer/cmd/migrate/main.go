package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/ilaif/athena-cycle/syncer/internal/config"
	"github.com/ilaif/athena-cycle/syncer/internal/logging"
)

type MigrationDirection string

const (
	Up   MigrationDirection = "up"
	Down MigrationDirection = "down"
)

func main() {
	direction := Up
	if len(os.Args) > 1 && os.Args[1] == "down" {
		direction = Down
	}

	log, err := logging.NewLogger()
	if err != nil {
		panic(err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Error(err, "Failed to load config")
		return
	}

	migrator, err := migrate.New("file://migrations", cfg.PgURL)
	if err != nil {
		log.Error(err, "Failed to create migrate instance")
		return
	}
	defer migrator.Close()
	migrator.Log = &migrateLogger{log: log}

	log = log.WithValues("direction", direction)

	migrationFunc := migrator.Up
	if direction == Down {
		ver, _, err := migrator.Version()
		if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
			log.Error(err, "Failed to get migration version")
			return
		}
		migrationFunc = func() error {
			if ver == 1 {
				return migrator.Down()
			}
			return migrator.Migrate(ver - 1)
		}
	}
	log.Info("Running migrations")
	if err := migrationFunc(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Error(err, "Failed to run migrations")
		return
	}

	log.Info("Migrations ran successfully")
}

type migrateLogger struct {
	log logr.Logger
}

func (l *migrateLogger) Printf(format string, v ...interface{}) {
	l.log.Info(fmt.Sprintf(format, v...))
}

func (l *migrateLogger) Verbose() bool {
	return true
}
