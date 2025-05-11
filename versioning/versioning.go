package versioning

import (
	"regexp"
)

var vPrefix = regexp.MustCompile(`^v\d`)

func Normalize(in string) string {
	if vPrefix.MatchString(in) {
		return in[1:]
	}
	return in
}

func ShortenSha(sha string) string {
	return sha[:7]
}
