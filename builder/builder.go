package builder

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path"
)

type BuilderConfig struct {
	Main        string
	Binary      string
	OutDir      string `default:"out"`
	Concurrency int    `default:"4"`
}

type Builder struct {
	config  BuilderConfig
	version string
}

type BuildResult struct {
	Target          Target
	Err             error
	BuiltExecutable string
}

func (b *Builder) doBuild(ctx context.Context, jobs <-chan Target, results chan<- *BuildResult, done func()) {
	defer done()

	for {
		select {
		case <-ctx.Done():
			return

		case target, jobsOpen := <-jobs:
			if !jobsOpen {
				return
			}
			if binary, err := b.buildTarget(ctx, target); err != nil {
				results <- &BuildResult{
					Target: target,
					Err:    err,
				}
				return
			} else {
				results <- &BuildResult{
					Target:          target,
					BuiltExecutable: binary,
				}
			}
		}
	}
}

// buildTarget returns the resolved binary path and/or a build error
func (b *Builder) buildTarget(ctx context.Context, target Target) (string, error) {
	archDirName := fmt.Sprintf("%s_%s%s", target.Os, target.Arch, target.Variant)
	archDir := path.Join(b.config.OutDir, archDirName)
	err := os.MkdirAll(archDir, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	var binary string
	if target.Os == "windows" {
		binary = b.config.Binary + ".exe"
	} else {
		binary = b.config.Binary
	}
	resolvedBinary := path.Join(archDir, binary)
	slog.Info("building target",
		"os", target.Os, "arch", target.Arch, "variant", target.Variant,
		"binary", resolvedBinary)

	cmd := exec.CommandContext(ctx, "go", "build",
		"-o", resolvedBinary,
		"-ldflags", fmt.Sprintf(`-X "main.Version=%s"`, b.version),
		b.config.Main)
	cmd.Env = append(os.Environ(),
		"GOOS="+target.Os,
		"GOARCH="+target.Arch,
	)
	if target.Variant == "arm" {
		cmd.Env = append(cmd.Env, "GOARM="+target.Variant)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return resolvedBinary, cmd.Run()
}

func Build(ctx context.Context, config BuilderConfig, targets []Target, version string) ([]BuildResult, error) {
	if config.Main == "" {
		return nil, errors.New("builder main is required")
	}
	if config.Binary == "" {
		return nil, errors.New("builder binary is required")
	}

	if err := os.MkdirAll(config.OutDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	b := Builder{
		config:  config,
		version: version,
	}

	jobsCtx, cancelJobs := context.WithCancel(ctx)
	defer cancelJobs()

	workersCount := min(len(targets), config.Concurrency)
	workersDone := make(chan struct{}, workersCount)
	jobs := make(chan Target, len(targets))
	targetResultsChan := make(chan *BuildResult, workersCount)

	// Kick off the workers
	for i := 0; i < workersCount; i++ {
		go b.doBuild(jobsCtx, jobs, targetResultsChan, func() {
			workersDone <- struct{}{}
		})
	}

	results := make([]BuildResult, 0, len(targets))
	workersDoneCount := 0

	handleResult := func(result *BuildResult) {
		if result.Err != nil {
			cancelJobs()
		}
		slog.Info("built", "target", result.Target, "err", result.Err, "binary", result.BuiltExecutable)
		results = append(results, *result)
	}

	// Feed them jobs and collect early results
	for _, target := range targets {
		select {
		case jobs <- target:
		case <-ctx.Done():
			return nil, ctx.Err()
		case result := <-targetResultsChan:
			handleResult(result)
		}
	}

	// Tell workers no jobs left
	close(jobs)

	// Collect results and wait for workers
	for workersDoneCount < workersCount {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case result := <-targetResultsChan:
			handleResult(result)
		case <-workersDone:
			workersDoneCount++
		}
	}

	return results, nil
}

func GoModDownload(ctx context.Context) error {
	slog.Info("downloading modules...")
	cmd := exec.CommandContext(ctx, "go", "mod", "download")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
