package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v62/github"
	"github.com/pkg/errors"

	"github.com/ilaif/athena-cycle/syncer/internal/logging"
)

func getEntity[T any](ctx context.Context,
	tokenManager *TokenManager,
	getFunc func(ctx context.Context, client *github.Client) (*T, *github.Response, error),
) (*T, *github.Response, error) {
	log := logging.MustFromContext(ctx)
	log.Info("Getting entity", "entity", fmt.Sprintf("%T", new(T)))
	client := newRotatableClient(ctx, tokenManager.GetToken())
	entity, resp, err := getFunc(ctx, client.Client)
	if resp != nil {
		log.V(1).Info("Rate limit remaining", "remaining", resp.Rate.Remaining)
	}
	if err != nil {
		if resp != nil {
			if err := handleRateLimit(ctx, resp, tokenManager); err != nil {
				return nil, resp, err
			}
			return getEntity(ctx, tokenManager, getFunc)
		}
		return nil, resp, errors.Wrap(err, "failed to get entity")
	}
	return entity, resp, nil
}

func listEntities[T any](ctx context.Context,
	tokenManager *TokenManager,
	listFunc func(ctx context.Context, client *github.Client) ([]*T, *github.Response, error),
) ([]*T, *github.Response, error) {
	log := logging.MustFromContext(ctx)
	log.Info("Listing entities", "entity", fmt.Sprintf("%T", new(T)))
	client := newRotatableClient(ctx, tokenManager.GetToken())
	entities, resp, err := listFunc(ctx, client.Client)
	if resp != nil {
		log.V(1).Info("Rate limit remaining", "remaining", resp.Rate.Remaining)
	}
	if err != nil {
		if resp != nil {
			if err := handleRateLimit(ctx, resp, tokenManager); err != nil {
				return nil, resp, err
			}
			return listEntities(ctx, tokenManager, listFunc)
		}
		return nil, resp, errors.Wrap(err, "failed to list entities")
	}
	return entities, resp, nil
}
