package builder

import (
	"fmt"
	"regexp"
)

type Target struct {
	Os      string
	Arch    string
	Variant string
}

var platformPattern = regexp.MustCompile("(.+?)/(.+?)(/(.+))?$")

func ParsePlatforms(platforms []string) ([]Target, error) {
	result := make([]Target, 0, len(platforms))
	for _, str := range platforms {
		results := platformPattern.FindStringSubmatch(str)
		if len(results) == 0 {
			return nil, fmt.Errorf("invalid pattern: %s", str)
		}
		result = append(result, Target{
			Os:      results[1],
			Arch:    results[2],
			Variant: results[4],
		})
	}

	return result, nil
}
