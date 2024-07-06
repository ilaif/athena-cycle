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
	syncBackTime       = 6 * 30 * 24 * time.Hour
	prSyncConcurrency  = 3
	prsPerPage         = 100
	issueEventsPerPage = 100
	prReviewsPerPage   = 100
	prFilesPerPage     = 100
)

func Sync(ctx context.Context, db *sqlx.DB, repos []config.GitHubRepository, tokens []string) error {
	log := logging.MustFromContext(ctx)
	log.Info("Syncing repositories")
	tokenManager := NewTokenManager(tokens)

	for _, repoIdentifier := range repos {
		repoLog := log.WithValues("repo", repoIdentifier)
		repoCtx := logging.NewContext(ctx, repoLog)
		repoLog.Info("Syncing repository", "repo", repoIdentifier)

		repo, _, err := getEntity(repoCtx, tokenManager,
			func(ctx context.Context, client *github.Client) (*github.Repository, *github.Response, error) {
				return client.Repositories.Get(ctx, repoIdentifier.Owner(), repoIdentifier.Name())
			},
		)
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

	direction := "asc"
	if lastSynced != nil {
		log.Info("Last synced time found, syncing from latest to last sync time", "last_synced", lastSynced)
		direction = "desc"
	} else {
		log.Info("No last synced time found, starting from the first page")
	}

	opt := &github.PullRequestListOptions{
		ListOptions: github.ListOptions{PerPage: prsPerPage},
		State:       "all",
		Sort:        "updated",
		Direction:   direction,
	}

	var latestPr *pullRequest

	for {
		prs, resp, err := listEntities(ctx, tokenManager,
			func(ctx context.Context, client *github.Client) ([]*github.PullRequest, *github.Response, error) {
				return client.PullRequests.List(ctx, *repo.Owner.Login, *repo.Name, opt)
			},
		)
		if err != nil {
			return errors.Wrap(err, "failed to list pull requests")
		}
		// Filter out pull requests that were already synced
		prsToSync := lo.Filter(prs, func(pr *github.PullRequest, _ int) bool {
			prUpdatedAt := pr.GetUpdatedAt()
			syncEarliestTime := time.Now().UTC().Add(-syncBackTime)
			if prUpdatedAt.Before(syncEarliestTime) {
				return false
			}
			if lastSynced == nil {
				return true
			}
			return prUpdatedAt.After(*lastSynced)
		})
		if len(prsToSync) == 0 {
			if direction == "desc" {
				log.Info("No new pull requests found")
				break
			}

			lastPr := prs[len(prs)-1]
			log.Info("Skipped all pull requests in page, moving to next page", "last_pr_updated_at", lastPr.UpdatedAt)
			if resp.NextPage == 0 {
				break
			}
			opt.Page = resp.NextPage
			continue
		}

		pullRequests, err := syncPullRequestsChunk(ctx, db, tokenManager, repo, prsToSync)
		if err != nil {
			return errors.Wrap(err, "failed to sync pull requests chunk")
		}
		log.Info("Synced pull requests", "prs", len(pullRequests), "page", opt.Page, "last_pr_updated_at", pullRequests[0].UpdatedAt)

		if direction == "desc" { // If we're syncing in descending order, we need to stop when we reach the last synced time
			if latestPr == nil {
				latestPr = pullRequests[0] // The latest PR is the first one in the first page
			}
		} else {
			latestPr = pullRequests[len(pullRequests)-1]
			log.Info("Updating last synced time", "last_synced", latestPr.UpdatedAt)
			if err := pg.UpdateLastSyncAt(ctx, db, repo.GetFullName(), latestPr.UpdatedAt); err != nil {
				return errors.Wrap(err, "failed to update last synced time")
			}
		}

		if direction == "desc" {
			if len(prsToSync) < len(prs) { // If we filtered, it means we reached the last page
				log.Info("Reached last synced time", "last_synced", lastSynced)
				break
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	if latestPr != nil {
		if err := pg.UpdateLastSyncAt(ctx, db, repo.GetFullName(), latestPr.UpdatedAt); err != nil {
			return errors.Wrap(err, "failed to update last synced time")
		}
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

	additions, deletions, numberOfChangedFiles, err := getPRFileChanges(ctx, tokenManager, repo, pr.Number)
	if err != nil {
		return errors.Wrap(err, "failed to get pull request file changes")
	}
	pr.Additions, pr.Deletions, pr.ChangedFiles = additions, deletions, numberOfChangedFiles

	return nil
}

func getLastReadyForReviewEvent(ctx context.Context, tokenManager *TokenManager,
	repo *github.Repository, prNumber int,
) (*github.IssueEvent, error) {
	var lastReadyForReviewEvent *github.IssueEvent
	opt := &github.ListOptions{PerPage: issueEventsPerPage}
	for {
		// default order is desc so we get the latest events first
		prEvents, resp, err := listEntities(ctx, tokenManager,
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

func getPRFileChanges(ctx context.Context, tokenManager *TokenManager, repo *github.Repository, prNumber int) (int, int, int, error) {
	additions, deletions, numberOfFiles := 0, 0, 0
	opts := &github.ListOptions{PerPage: prFilesPerPage}
	for {
		files, res, err := listEntities(ctx, tokenManager,
			func(ctx context.Context, client *github.Client) ([]*github.CommitFile, *github.Response, error) {
				return client.PullRequests.ListFiles(ctx, *repo.Owner.Login, *repo.Name, prNumber, opts)
			},
		)
		if err != nil {
			return 0, 0, 0, errors.Wrap(err, "failed to get pull request files")
		}
		for _, file := range files {
			additions += file.GetAdditions()
			deletions += file.GetDeletions()
		}
		numberOfFiles += len(files)
		if res.NextPage == 0 {
			break
		}
		opts.Page = res.NextPage
	}
	return additions, deletions, numberOfFiles, nil
}

func syncPullRequestReviews(ctx context.Context, db *sqlx.DB, tokenManager *TokenManager, repo *github.Repository, pr *pullRequest) error {
	opt := &github.ListOptions{PerPage: prReviewsPerPage}
	for {
		// default order is desc so we get the latest events first
		reviews, resp, err := listEntities(ctx, tokenManager,
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
