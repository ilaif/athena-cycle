package github

import (
	"context"
	"time"

	"github.com/google/go-github/v62/github"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"

	"github.com/ilaif/athena-cycle/syncer/internal/config"
	"github.com/ilaif/athena-cycle/syncer/internal/logging"
	"github.com/ilaif/athena-cycle/syncer/internal/pg"
)

const (
	prSyncConcurrency  = 3
	prsPerPage         = 100
	issueEventsPerPage = 100
	prReviewsPerPage   = 100
)

func Sync(ctx context.Context, db *sqlx.DB, repos []config.GitHubRepository, tokens []string) error {
	log := logging.MustFromContext(ctx)
	log.Info("Syncing repositories")
	tokenManager := NewTokenManager(tokens)

	for _, repoIdentifier := range repos {
		repoLog := log.WithValues("repo", repoIdentifier)
		repoCtx := logging.NewContext(ctx, repoLog)
		repoLog.Info("Syncing repository", "repo", repoIdentifier)

		repo, err := getRepo(repoCtx, tokenManager, repoIdentifier)
		if err != nil {
			return errors.Wrap(err, "failed to get repository")
		}

		if err := syncRepoPullRequests(repoCtx, db, tokenManager, repo); err != nil {
			repoLog.Error(err, "Failed to sync pull requests", "repo", repo.GetName())
		}

		repoLog.Info("Synced repository")
	}

	log.Info("Synced repositories")
	return nil
}

func syncRepoPullRequests(ctx context.Context, db *sqlx.DB, tokenManager *TokenManager,
	repo *github.Repository,
) error {
	log := logging.MustFromContext(ctx)
	log.Info("Syncing pull requests")

	lastSynced, err := pg.GetLastSyncAt(ctx, db, repo.GetFullName())
	if err != nil {
		return err
	}
	log.Info("Last synced time", "last_synced", lastSynced)

	opt := &github.PullRequestListOptions{
		ListOptions: github.ListOptions{PerPage: prsPerPage},
		State:       "all",
		Sort:        "updated",
		Direction:   "desc",
	}

	var latestPr *pullRequest

	for {
		prs, resp, err := listEntities(ctx, tokenManager, repo,
			func(ctx context.Context, client *github.Client) ([]*github.PullRequest, *github.Response, error) {
				return client.PullRequests.List(ctx, *repo.Owner.Login, *repo.Name, opt)
			},
		)
		if err != nil {
			return errors.Wrap(err, "failed to list pull requests")
		}
		if len(prs) == 0 {
			break
		}

		pullRequests, err := syncPullRequestsChunk(ctx, db, tokenManager, repo, prs)
		if err != nil {
			return errors.Wrap(err, "failed to sync pull requests chunk")
		}
		log.Info("Synced pull requests", "prs", len(pullRequests), "page", opt.Page, "last_pr_updated_at", pullRequests[0].UpdatedAt)

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage

		if latestPr == nil {
			latestPr = pullRequests[0]
		}

		earliestPr := pullRequests[len(pullRequests)-1]
		if earliestPr.UpdatedAt.Before(lastSynced) {
			log.Info("Reached last synced time", "last_synced", lastSynced)
			break
		}
	}

	log.Info("Updating last synced time", "last_synced", latestPr.UpdatedAt)
	if err := pg.UpdateLastSyncAt(ctx, db, repo.GetFullName(), latestPr.UpdatedAt); err != nil {
		return errors.Wrap(err, "failed to update last synced time")
	}

	return nil
}

func syncPullRequestsChunk(ctx context.Context, db *sqlx.DB, tokenManager *TokenManager,
	repo *github.Repository, prs []*github.PullRequest,
) ([]*pullRequest, error) {
	prChan := make(chan *pullRequest, len(prs))
	sem := make(chan struct{}, prSyncConcurrency)
	eg := errgroup.Group{}
	for _, pr := range prs {
		pr := pr
		sem <- struct{}{}
		eg.Go(func() error {
			defer func() { <-sem }()
			var mergedAt *time.Time
			if pr.MergedAt != nil {
				mergedAt = &pr.MergedAt.Time
			}
			pullRequest := &pullRequest{
				PrID:      int(pr.GetID()),
				RepoID:    int(repo.GetID()),
				Repo:      repo.GetFullName(),
				Number:    pr.GetNumber(),
				Username:  pr.GetUser().GetLogin(),
				Title:     pr.GetTitle(),
				Body:      pr.Body,
				State:     pr.GetState(),
				Draft:     pr.GetDraft(),
				MergedAt:  mergedAt,
				CreatedAt: pr.GetCreatedAt().Time,
				UpdatedAt: pr.GetUpdatedAt().Time,
				Data:      pr,
			}
			if err := enrichPullRequest(ctx, tokenManager, repo, pullRequest); err != nil {
				return errors.Wrap(err, "failed to enrich pull request")
			}
			if err := syncPullRequestReviews(ctx, db, tokenManager, repo, pullRequest); err != nil {
				return errors.Wrap(err, "failed to sync pull request reviews")
			}
			prChan <- pullRequest
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, errors.Wrap(err, "failed to enrich pull request chunk")
	}
	close(prChan)
	pullRequests := make([]*pullRequest, 0, len(prs))
	for pr := range prChan {
		pullRequests = append(pullRequests, pr)
	}
	if err := upsertPullRequests(ctx, db, pullRequests); err != nil {
		return nil, errors.Wrap(err, "failed to insert pull requests")
	}
	return pullRequests, nil
}

func enrichPullRequest(ctx context.Context, tokenManager *TokenManager,
	repo *github.Repository, pr *pullRequest,
) error {
	log := logging.MustFromContext(ctx).WithValues("pr", pr.Number)
	ctx = logging.NewContext(ctx, log)

	log.Info("Enriching pull request", "pr", pr.Number, "pr_updated_at", pr.UpdatedAt)

	lastReadyForReviewEvent, err := getLastReadyForReviewEvent(ctx, tokenManager, repo, pr.Number)
	if err != nil {
		return errors.Wrap(err, "failed to get last ready for review event")
	}
	if lastReadyForReviewEvent != nil {
		log.Info("Found last ready for review event", "event", lastReadyForReviewEvent)
		pr.LastReadyForReviewAt = &lastReadyForReviewEvent.CreatedAt.Time
	}

	return nil
}

func getLastReadyForReviewEvent(ctx context.Context, tokenManager *TokenManager,
	repo *github.Repository, prNumber int,
) (*github.IssueEvent, error) {
	var lastReadyForReviewEvent *github.IssueEvent
	opt := &github.ListOptions{PerPage: issueEventsPerPage}
	for {
		// default order is desc so we get the latest events first
		prEvents, resp, err := listEntities(ctx, tokenManager, repo,
			func(ctx context.Context, client *github.Client) ([]*github.IssueEvent, *github.Response, error) {
				return client.Issues.ListIssueEvents(ctx, *repo.Owner.Login, *repo.Name, prNumber, opt)
			},
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list issue events")
		}
		for _, event := range prEvents {
			if event.GetEvent() == "ready_for_review" {
				lastReadyForReviewEvent = event
				break
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return lastReadyForReviewEvent, nil
}

func syncPullRequestReviews(ctx context.Context, db *sqlx.DB, tokenManager *TokenManager, repo *github.Repository, pr *pullRequest) error {
	opt := &github.ListOptions{PerPage: prReviewsPerPage}
	for {
		// default order is desc so we get the latest events first
		reviews, resp, err := listEntities(ctx, tokenManager, repo,
			func(ctx context.Context, client *github.Client) ([]*github.PullRequestReview, *github.Response, error) {
				return client.PullRequests.ListReviews(ctx, *repo.Owner.Login, *repo.Name, pr.Number, opt)
			},
		)
		if err != nil {
			return errors.Wrap(err, "failed to list issue events")
		}
		if err := upsertPullRequestReviews(ctx, db,
			lo.Map(reviews, func(review *github.PullRequestReview, _ int) *pullRequestReview {
				return &pullRequestReview{
					ReviewID:    int(review.GetID()),
					PrID:        pr.PrID,
					Repo:        repo.GetFullName(),
					Username:    review.GetUser().GetLogin(),
					State:       review.GetState(),
					SubmittedAt: review.GetSubmittedAt().Time,
					CommitID:    review.GetCommitID(),
					Data:        review,
				}
			}),
		); err != nil {
			return errors.Wrap(err, "failed to insert pull requests")
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return nil
}
