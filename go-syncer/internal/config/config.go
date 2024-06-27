package config

import (
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/pkg/errors"
)

type GitHubRepository string

func (r GitHubRepository) Valid() bool {
	return strings.Contains(string(r), "/")
}

func (r GitHubRepository) Owner() string {
	return strings.Split(string(r), "/")[0]
}

func (r GitHubRepository) Name() string {
	return strings.Split(string(r), "/")[1]
}

type Config struct {
	PgURL              string             `env:"PG_URL"`
	GitHubTokens       []string           `env:"GITHUB_TOKENS"`
	GitHubRepositories []GitHubRepository `env:"GITHUB_REPOSITORIES"`
}

func LoadConfig() (*Config, error) {
	var cfg Config
	err := env.Parse(&cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse environment variables")
	}

	for _, repo := range cfg.GitHubRepositories {
		if !repo.Valid() {
			return nil, errors.Errorf("invalid GitHub repository: %s", repo)
		}
	}

	return &cfg, nil
}
