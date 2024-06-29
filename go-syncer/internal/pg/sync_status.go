package pg

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

func GetLastSyncAt(ctx context.Context, db *sqlx.DB, repo string) (*time.Time, error) {
	var lastSynced *time.Time
	err := db.GetContext(ctx, &lastSynced, `SELECT last_synced FROM sync_status WHERE repo = $1`, repo)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, errors.Wrap(err, "failed to get last synced time")
	}
	return lastSynced, nil
}

func UpdateLastSyncAt(ctx context.Context, db *sqlx.DB, repo string, lastSynced time.Time) error {
	if _, err := db.ExecContext(ctx, `
		INSERT INTO sync_status (repo, last_synced)
		VALUES ($1, $2)
		ON CONFLICT (repo) DO UPDATE
		SET last_synced = EXCLUDED.last_synced
	`, repo, lastSynced); err != nil {
		return errors.Wrap(err, "failed to update last synced time")
	}
	return nil
}
