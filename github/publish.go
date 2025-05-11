package github

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/go-github/v71/github"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strings"
)

type Publisher struct {
	token      string
	repository string
}

// NewPublisher creates a GitHub publisher if the configuration is available for it, returns nil otherwise
func NewPublisher(config *Config) *Publisher {
	if config.Token != "" && config.Repository != "" {
		return &Publisher{
			token:      config.Token,
			repository: config.Repository,
		}
	} else {
		return nil
	}
}

func (g *Publisher) Publish(ctx context.Context, archiveFilePaths []string, version string) error {
	if g.token == "" {
		return errors.New("github token is required")
	}
	if g.repository == "" {
		return errors.New("github repository is required")
	}

	client := github.NewClient(nil).
		WithAuthToken(g.token)

	ownerName := strings.SplitN(g.repository, "/", 2)
	if len(ownerName) != 2 {
		return errors.New("github repository must be in the form owner/repo")
	}
	owner := ownerName[0]
	repoName := ownerName[1]

	release, getReleaseResp, err := client.Repositories.GetReleaseByTag(ctx, owner, repoName, version)
	if err != nil {
		if getReleaseResp != nil && getReleaseResp.StatusCode == http.StatusNotFound {
			release, err = g.createRelease(ctx, client, owner, repoName, version)
			if err != nil {
				return fmt.Errorf("failed to create release: %w", err)
			}
		} else {
			return fmt.Errorf("failed to get release: %w", err)
		}
	} else {
		slog.Debug("uploading assets to existing GitHub release")
	}

	slog.Info("uploading GitHub release assets",
		"name", release.GetName(),
		"tag", release.GetTagName(),
		"owner", owner, "repo", repoName)
	for _, archiveFilePath := range archiveFilePaths {
		slog.Info("uploading release asset", "file", archiveFilePath)

		archiveFile, err := os.Open(archiveFilePath)
		if err != nil {
			return fmt.Errorf("failed to open archive file: %w", err)
		}
		//goland:noinspection GoUnhandledErrorResult
		defer archiveFile.Close()

		_, _, err = client.Repositories.UploadReleaseAsset(ctx, owner, repoName, release.GetID(),
			&github.UploadOptions{
				Name: path.Base(archiveFilePath),
			},
			archiveFile)
		if err != nil {
			return fmt.Errorf("failed to upload release asset: %w", err)
		}
	}

	return nil
}

func (g *Publisher) createRelease(ctx context.Context, client *github.Client, owner string, repoName string, version string) (*github.RepositoryRelease, error) {
	release, _, err := client.Repositories.CreateRelease(ctx, owner, repoName, &github.RepositoryRelease{
		TagName:              &version,
		Name:                 &version,
		GenerateReleaseNotes: github.Ptr(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create release: %w", err)
	}
	slog.Debug("created new GitHub release", "name", release.GetName(), "url", release.GetHTMLURL())

	return release, nil
}
