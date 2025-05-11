package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/itzg/go-flagsfiller"
	"go-mbuild-release/builder"
	"go-mbuild-release/github"
	"go-mbuild-release/stages"
	"go-mbuild-release/versioning"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
)

type Args struct {
	Platform []string
	Builder  builder.BuilderConfig
	Archive  builder.ArchiveConfig
	Project  string
}

func main() {
	var args Args

	err := flagsfiller.Parse(&args, flagsfiller.WithEnv(""))
	if err != nil {
		slog.Error("failed to parse args", "err", err)
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if len(args.Platform) == 0 {
		args.Platform = []string{runtime.GOOS + "/" + runtime.GOARCH}
		slog.Info("platform list is empty using current platform")
	}
	targets, err := builder.ParsePlatforms(args.Platform)
	if err != nil {
		slog.Error("failed to parse platforms", "err", err)
		os.Exit(2)
	}

	githubConfig, err := github.LoadConfig(ctx)
	if err != nil {
		slog.Error("failed to load github config", "err", err)
		os.Exit(1)
	}

	version, err := determineVersion(ctx, githubConfig)
	if err != nil {
		slog.Error("failed to determine version", "err", err)
		os.Exit(1)
	}
	slog.Info("determined version", "version", version)

	err = builder.GoModDownload(ctx)
	if err != nil {
		slog.Error("failed to download go modules", "err", err)
		os.Exit(1)
	}

	buildResults, err := builder.Build(ctx, args.Builder, targets, versioning.Normalize(version))
	if err != nil {
		slog.Error("failed to build", "err", err)
		os.Exit(1)
	}

	project := args.Project
	if project == "" {
		project = args.Builder.Binary
	}
	archiver := builder.NewArchiver(args.Archive, versioning.Normalize(version),
		builder.WithProject(project))
	archiveResults, err := archiver.ArchiveBuildResults(buildResults)
	if err != nil {
		slog.Error("failed to archive", "err", err)
		os.Exit(1)
	}

	archiveFilePaths := make([]string, len(archiveResults))
	for i, archiveResult := range archiveResults {
		archiveFilePaths[i] = archiveResult.Archive
	}

	var publisher stages.Publisher
	if githubConfig != nil {
		publisher = github.NewPublisher(githubConfig)
	}

	if publisher != nil {
		err = publisher.Publish(ctx, archiveFilePaths, version)
		if err != nil {
			slog.Error("failed to publish", "err", err)
			os.Exit(1)
		}
	} else {
		slog.Info("no publisher configured, skipping publish")
	}
}

func determineVersion(ctx context.Context, githubConfig *github.Config) (string, error) {
	var v stages.Versioning
	v = github.NewVersioning(githubConfig)

	if v == nil {
		var err error
		v, err = versioning.NewGitVersioning(".")
		if err != nil {
			return "", fmt.Errorf("failed to determine version from git: %w", err)
		}
	}

	if v == nil {
		return "", errors.New("failed to locate versioning strategy")
	}

	version, err := v.GetVersion()
	if err != nil {
		slog.Error("failed to determine version", "err", err)
		os.Exit(1)
	}
	return version, err
}
