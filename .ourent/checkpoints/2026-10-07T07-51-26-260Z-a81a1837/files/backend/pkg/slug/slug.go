// Package slug creates URL-safe slugs.
package slug

import (
	"regexp"
	"strings"
)

var re = regexp.MustCompile(`[^a-z0-9]+`)
var Valid = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func Make(s string) string {
	s = strings.Trim(re.ReplaceAllString(strings.ToLower(s), "-"), "-")
	if len(s) > 80 {
		s = strings.Trim(s[:80], "-")
	}
	if s == "" {
		s = "item"
	}
	return s
}
