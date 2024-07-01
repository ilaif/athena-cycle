package github

import (
	"time"

	"github.com/google/go-github/v62/github"
)

type pullRequest struct {
	PrID                 int                 `db:"pr_id"`
	RepoID               int                 `db:"repo_id"`
	Repo                 string              `db:"repo"`
	Number               int                 `db:"number"`
	Username             string              `db:"username"`
	Title                string              `db:"title"`
	Body                 *string             `db:"body"`
	State                string              `db:"state"`
	Draft                bool                `db:"draft"`
	Additions            int                 `db:"additions"`
	Deletions            int                 `db:"deletions"`
	ChangedFiles         int                 `db:"changed_files"`
	MergedAt             *time.Time          `db:"merged_at"`
	CreatedAt            time.Time           `db:"created_at"`
	UpdatedAt            time.Time           `db:"updated_at"`
	LastReadyForReviewAt *time.Time          `db:"last_ready_for_review_at"`
	Data                 *github.PullRequest `db:"data"`
}

type pullRequestReview struct {
	ReviewID    int                       `db:"review_id"`
	PrID        int                       `db:"pr_id"`
	Repo        string                    `db:"repo"`
	Username    string                    `db:"username"`
	State       string                    `db:"state"`
	SubmittedAt time.Time                 `db:"submitted_at"`
	CommitID    string                    `db:"commit_id"`
	Data        *github.PullRequestReview `db:"data"`
}
