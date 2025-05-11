package github

import (
	"context"
	"github.com/sethvargo/go-envconfig"
)

// Config maps in the environment variables provided by Github Actions
// https://docs.github.com/en/actions/writing-workflows/choosing-what-your-workflow-does/store-information-in-variables#default-environment-variables
type Config struct {
	RefName    string `env:"GITHUB_REF_NAME"`
	RefType    string `env:"GITHUB_REF_TYPE"`
	Sha        string `env:"GITHUB_SHA"`
	Token      string `env:"GITHUB_TOKEN"`
	Repository string `env:"GITHUB_REPOSITORY"`
}

// LoadConfig loads the GitHub config from the environment; however, this does not necessarily indicate that
// all fields are available for versioning, publishing, etc
func LoadConfig(ctx context.Context) (*Config, error) {
	config := &Config{}
	err := envconfig.Process(ctx, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
