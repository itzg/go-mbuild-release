package builder

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
)

type Archiver struct {
	outDir  string
	project string
	version string
}

type ArchiverOption func(*Archiver)

var WithProject = func(project string) ArchiverOption {
	return func(archiver *Archiver) {
		archiver.project = project
	}
}

func NewArchiver(config ArchiveConfig, version string, options ...ArchiverOption) *Archiver {
	archiver := &Archiver{
		outDir:  config.OutDir,
		version: version,
	}
	for _, option := range options {
		option(archiver)
	}

	return archiver
}

type ArchiveConfig struct {
	OutDir string `default:"out"`
}

type ArchiveResult struct {
	Target  Target
	Archive string
}

func (a *Archiver) ArchiveBuildResults(buildResults []BuildResult) ([]*ArchiveResult, error) {
	results := make([]*ArchiveResult, len(buildResults))
	for i, buildResult := range buildResults {
		if buildResult.Err != nil {
			continue
		}

		result, err := a.archive(buildResult)
		if err != nil {
			return nil, fmt.Errorf("failed to archive: %w", err)
		}
		slog.Info("archived", "target", result.Target, "archive", result.Archive)
		results[i] = result
	}
	return results, nil
}

func (a *Archiver) archive(buildResult BuildResult) (*ArchiveResult, error) {
	slog.Info("archiving", "target", buildResult.Target)

	if buildResult.Target.Os == "windows" {
		return a.archiveZip(buildResult)
	} else {
		return a.archiveTarGz(buildResult)
	}
}

func (a *Archiver) archiveTarGz(buildResult BuildResult) (*ArchiveResult, error) {
	outFile, err := os.Create(a.formFilename(buildResult, "tar.gz"))
	if err != nil {
		return nil, fmt.Errorf("failed to create archive file: %w", err)
	}
	//goland:noinspection GoUnhandledErrorResult
	defer outFile.Close()

	gzipWriter := gzip.NewWriter(outFile)
	//goland:noinspection GoUnhandledErrorResult
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	//goland:noinspection GoUnhandledErrorResult
	defer tarWriter.Close()

	err = a.writeFileToTar(buildResult.BuiltExecutable, tarWriter)
	if err != nil {
		return nil, fmt.Errorf("failed to write file %s to tar: %w", buildResult.BuiltExecutable, err)
	}

	// Add README.md if exists
	if _, err := os.Stat("README.md"); err == nil {
		err = a.writeFileToTar("README.md", tarWriter)
		if err != nil {
			return nil, fmt.Errorf("failed to write README.md to archive: %w", err)
		}
	}

	return &ArchiveResult{
		Target:  buildResult.Target,
		Archive: outFile.Name(),
	}, nil
}

func (a *Archiver) archiveZip(buildResult BuildResult) (*ArchiveResult, error) {
	outFile, err := os.Create(a.formFilename(buildResult, "zip"))
	if err != nil {
		return nil, fmt.Errorf("failed to create archive file: %w", err)
	}
	//goland:noinspection GoUnhandledErrorResult
	defer outFile.Close()

	zipWriter := zip.NewWriter(outFile)
	//goland:noinspection GoUnhandledErrorResult
	defer zipWriter.Close()

	err = a.writeFileToZip(buildResult.BuiltExecutable, zipWriter)
	if err != nil {
		return nil, fmt.Errorf("failed to write file %s to zip: %w", buildResult.BuiltExecutable, err)
	}

	// Add README.md if exists
	if _, err := os.Stat("README.md"); err == nil {
		err = a.writeFileToZip("README.md", zipWriter)
		if err != nil {
			return nil, fmt.Errorf("failed to write README.md to archive: %w", err)
		}
	}

	return &ArchiveResult{
		Target:  buildResult.Target,
		Archive: outFile.Name(),
	}, nil
}

func (a *Archiver) writeFileToTar(filePath string, tarWriter *tar.Writer) error {
	inFile, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file to archive: %w", err)
	}
	//goland:noinspection GoUnhandledErrorResult
	defer inFile.Close()

	stat, err := inFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file to archive: %w", err)
	}

	err = tarWriter.WriteHeader(&tar.Header{
		Name: path.Base(filePath),
		Mode: 0755,
		Size: stat.Size(),
	})
	if err != nil {
		return fmt.Errorf("failed to write tar header: %w", err)
	}
	_, err = io.Copy(tarWriter, inFile)
	if err != nil {
		return fmt.Errorf("failed to copy content into tar: %w", err)
	}

	return nil
}

func (a *Archiver) writeFileToZip(filePath string, zipWriter *zip.Writer) error {
	entryWriter, err := zipWriter.Create(path.Base(filePath))
	inFile, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file to archive: %w", err)
	}
	//goland:noinspection GoUnhandledErrorResult
	defer inFile.Close()

	_, err = io.Copy(entryWriter, inFile)
	if err != nil {
		return fmt.Errorf("failed to copy content into zip: %w", err)
	}
	return nil
}

func (a *Archiver) formFilename(result BuildResult, suffix string) string {
	return path.Join(a.outDir, fmt.Sprintf("%s_%s_%s_%s%s.%s",
		a.project, a.version,
		result.Target.Os,
		result.Target.Arch,
		result.Target.Variant,
		suffix,
	))
}
