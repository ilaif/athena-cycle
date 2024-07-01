package github

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

func upsertPullRequests(ctx context.Context, db *sqlx.DB, pullRequests []*pullRequest) error {
	if len(pullRequests) == 0 {
		return nil
	}
	if _, err := db.NamedExecContext(ctx, `
			INSERT INTO pull_requests (
				pr_id, repo, repo_id, number, username, title, body, state, draft, additions, deletions, changed_files,
				merged_at, created_at, updated_at, last_ready_for_review_at, data
			)
			VALUES (
				:pr_id, :repo, :repo_id, :number, :username, :title, :body, :state, :draft, :additions, :deletions, :changed_files,
				:merged_at, :created_at, :updated_at, :last_ready_for_review_at, :data
			)
			ON CONFLICT (pr_id) DO UPDATE
			SET repo = EXCLUDED.repo,
				repo_id = EXCLUDED.repo_id,
				number = EXCLUDED.number,
				username = EXCLUDED.username,
				title = EXCLUDED.title,
				body = EXCLUDED.body,
				state = EXCLUDED.state,
				draft = EXCLUDED.draft,
				additions = EXCLUDED.additions,
				deletions = EXCLUDED.deletions,
				changed_files = EXCLUDED.changed_files,
				merged_at = EXCLUDED.merged_at,
				created_at = EXCLUDED.created_at,
				updated_at = EXCLUDED.updated_at,
				last_ready_for_review_at = EXCLUDED.last_ready_for_review_at,
				data = EXCLUDED.data
			`, pullRequests,
	); err != nil {
		return errors.Wrap(err, "failed to insert pull request")
	}
	return nil
}

func upsertPullRequestReviews(ctx context.Context, db *sqlx.DB, reviews []*pullRequestReview) error {
	if len(reviews) == 0 {
		return nil
	}
	if _, err := db.NamedExecContext(ctx, `
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
			`, reviews,
	); err != nil {
		return errors.Wrap(err, "failed to insert pull request review")
	}
	return nil
}
