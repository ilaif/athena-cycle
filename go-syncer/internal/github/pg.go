package github

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

func upsertPullRequests(ctx context.Context, db *sqlx.DB, pullRequests []*pullRequest) error {
	tx, err := db.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.NamedExec(`
			INSERT INTO pull_requests (
				pr_id, repo, repo_id, username, title, body, state, draft,
				merged_at, created_at, updated_at, last_ready_for_review_at, data
			)
			VALUES (
				:pr_id, :repo, :repo_id, :username, :title, :body, :state, :draft,
				:merged_at, :created_at, :updated_at, :last_ready_for_review_at, :data
			)
			ON CONFLICT (pr_id) DO UPDATE
			SET repo = EXCLUDED.repo,
				repo_id = EXCLUDED.repo_id,
				username = EXCLUDED.username,
				title = EXCLUDED.title,
				body = EXCLUDED.body,
				state = EXCLUDED.state,
				draft = EXCLUDED.draft,
				merged_at = EXCLUDED.merged_at,
				created_at = EXCLUDED.created_at,
				updated_at = EXCLUDED.updated_at,
				last_ready_for_review_at = EXCLUDED.last_ready_for_review_at,
				data = EXCLUDED.data
			`, pullRequests,
	); err != nil {
		return errors.Wrap(err, "failed to insert pull request")
	}
	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}
	return nil
}

func upsertPullRequestReviews(ctx context.Context, db *sqlx.DB, reviews []*pullRequestReview) error {
	tx, err := db.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer func() { _ = tx.Rollback() }()

	for _, review := range reviews {
		if _, err := tx.NamedExec(`
			INSERT INTO pull_request_reviews (review_id, pr_id, repo, username, state, submitted_at, commit_id, data)
			VALUES (:review_id, :pr_id, :repo, :username, :state, :submitted_at, :commit_id, :data)
			ON CONFLICT (review_id) DO UPDATE
			SET pr_id = EXCLUDED.pr_id,
				repo = EXCLUDED.repo,
				username = EXCLUDED.username,
				state = EXCLUDED.state,
				submitted_at = EXCLUDED.submitted_at,
				commit_id = EXCLUDED.commit_id,
				data = EXCLUDED.data
			`, review,
		); err != nil {
			return errors.Wrap(err, "failed to insert pull request review")
		}
	}
	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}
	return nil
}
