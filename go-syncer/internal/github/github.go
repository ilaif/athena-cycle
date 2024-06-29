package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v62/github"
	"github.com/pkg/errors"

	"github.com/ilaif/athena-cycle/syncer/internal/config"
	"github.com/ilaif/athena-cycle/syncer/internal/logging"
)

func getRepo(ctx context.Context, tokenManager *TokenManager,
	repoIdentifier config.GitHubRepository,
) (*github.Repository, error) {
	log := logging.MustFromContext(ctx)
	log.Info("Getting repository")
	client := newRotatableClient(ctx, tokenManager.GetToken())
	repo, resp, err := client.Repositories.Get(ctx, repoIdentifier.Owner(), repoIdentifier.Name())
	if err != nil {
		if resp != nil {
			if err := handleRateLimit(ctx, resp, tokenManager); err != nil {
				return nil, err
			}
			if !tokenManager.IsExhausted() {
				return getRepo(ctx, tokenManager, repoIdentifier)
			}
		}
		return nil, errors.Wrap(err, "failed to get repository")
	}
	return repo, nil
}

func listEntities[T any](ctx context.Context,
	tokenManager *TokenManager,
	repo *github.Repository,
	listFunc func(ctx context.Context, client *github.Client) ([]*T, *github.Response, error),
) ([]*T, *github.Response, error) {
	log := logging.MustFromContext(ctx)
	log.Info("Listing entities", "entity", fmt.Sprintf("%T", new(T)))
	client := newRotatableClient(ctx, tokenManager.GetToken())
	reviews, resp, err := listFunc(ctx, client.Client)
	if resp != nil {
		log.V(1).Info("Rate limit remaining", "remaining", resp.Rate.Remaining)
	}
	if err != nil {
		if resp != nil {
			if err := handleRateLimit(ctx, resp, tokenManager); err != nil {
				return nil, resp, err
			}
			return listEntities(ctx, tokenManager, repo, listFunc)
		}
		return nil, resp, errors.Wrap(err, "failed to list pull request reviews")
	}
	return reviews, resp, nil
}
