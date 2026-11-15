package utils

import (
	"regexp"
	"strings"
)

var miscRegex = regexp.MustCompile(`\s?【[^】]*】`)

// StandardSongName preserves versions such as (Live), while removing platform annotations.
func StandardSongName(name string) string {
	name = strings.NewReplacer("（", " (", "）", ")").Replace(name)
	return strings.TrimSpace(miscRegex.ReplaceAllString(name, ""))
}
