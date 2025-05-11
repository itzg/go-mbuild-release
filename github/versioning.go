package github

import (
	"fmt"
	"go-mbuild-release/stages"
	"go-mbuild-release/versioning"
)

// NewVersioning creates a GitHub versioning if the configuration is available for it, returns nil otherwise
func NewVersioning(config *Config) stages.Versioning {
	if config == nil {
		return nil
	} else if config.RefType != "" &&
		config.RefName != "" &&
		config.Sha != "" {
		return &Versioning{
			refType: config.RefType,
			refName: config.RefName,
			sha:     config.Sha,
		}
	} else {
		return nil
	}
}

type Versioning struct {
	refType string
	refName string
	sha     string
}

func (v *Versioning) GetVersion() (string, error) {
	switch v.refType {
	case "tag":
		return v.refName, nil

	case "branch":
		return fmt.Sprintf("%s-%s", v.refName, versioning.ShortenSha(v.sha)), nil
	}
	return "", fmt.Errorf("unsupported ref type: %s", v.refType)
}
