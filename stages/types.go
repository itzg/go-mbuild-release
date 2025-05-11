package stages

import "context"

type Publisher interface {
	Publish(ctx context.Context, archiveFilePaths []string, version string) error
}

type Versioning interface {
	GetVersion() (string, error)
}
