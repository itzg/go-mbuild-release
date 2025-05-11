package versioning

import (
	"errors"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"go-mbuild-release/stages"
)

type GitVersioning struct {
	repo *git.Repository
}

// NewGitVersioning creates a Git versioning if the repository is available for it, returns nil otherwise
// path must be a git repository path, e.g. /path/to/repo/.git
func NewGitVersioning(path string) (stages.Versioning, error) {
	repo, err := git.PlainOpen(path)
	if err != nil {
		if errors.Is(err, git.ErrRepositoryNotExists) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to open repo: %w", err)
	}

	return &GitVersioning{
		repo: repo,
	}, nil
}

func (gv *GitVersioning) deriveRelativeVersion(hashToTag map[plumbing.Hash]string, headHash plumbing.Hash) (string, error) {
	commitItr, err := gv.repo.Log(&git.LogOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get commit log: %w", err)
	}
	defer commitItr.Close()

	distance := 0
	for {
		c, err := commitItr.Next()
		if c == nil {
			return ShortenSha(headHash.String()), nil
		}

		if err != nil {
			return "", fmt.Errorf("failed to get next commit: %w", err)
		}
		if tag, ok := hashToTag[c.Hash]; ok {
			// format like git describe
			return fmt.Sprintf("%s-%d-%s", tag, distance, ShortenSha(headHash.String())), nil
		}
		distance += 1
	}

}

func (gv *GitVersioning) GetVersion() (string, error) {
	tagsItr, err := gv.repo.Tags()
	if err != nil {
		return "", fmt.Errorf("failed to get tags: %w", err)
	}
	defer tagsItr.Close()

	hashToTag := make(map[plumbing.Hash]string)
	err = tagsItr.ForEach(func(ref *plumbing.Reference) error {
		if ref.Type() == plumbing.HashReference {
			hashToTag[ref.Hash()] = ref.Name().Short()
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to iterate tags: %w", err)
	}

	head, err := gv.repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get head: %w", err)
	}
	tag, ok := hashToTag[head.Hash()]
	if ok {
		return tag, nil
	}

	return gv.deriveRelativeVersion(hashToTag, head.Hash())
}
