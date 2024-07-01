package github

import (
	"context"
	"net/http"
	"time"

	"github.com/google/go-github/v62/github"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"github.com/ilaif/athena-cycle/syncer/internal/logging"
)

type RotatableGithubClient struct {
	*github.Client
}

func newRotatableClient(ctx context.Context, token string) *RotatableGithubClient {
	return &RotatableGithubClient{
		Client: NewClient(ctx, token),
	}
}

func (c *RotatableGithubClient) SetToken(ctx context.Context, token string) {
	c.Client = NewClient(ctx, token)
}

func NewClient(ctx context.Context, token string) *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc)
}

func handleRateLimit(ctx context.Context, resp *github.Response, tokenManager *TokenManager) error {
	log := logging.MustFromContext(ctx)

	if resp.StatusCode != http.StatusForbidden {
		return errors.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	rateLimit := resp.Rate
	if rateLimit.Remaining != 0 {
		return errors.New("other than rate limit exceeded forbidden error")
	}

	log.Info("Rate limit exceeded, rotating token", "reset_time", rateLimit.Reset.String())
	tokenManager.RotateToken()
	if tokenManager.IsExhausted() {
		const backoffBuffer = 10 * time.Second
		backoffDur := rateLimit.Reset.UTC().Sub(time.Now().UTC()) + backoffBuffer
		log.Info("All tokens exhausted, applying backoff", "backoff_duration", backoffDur)
		tokenManager.WaitForRateLimitReset(backoffDur)
		tokenManager.ResetExhaustion()
		return nil
	}

	return nil
}
