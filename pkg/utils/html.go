package utils

import "strings"

func HTMLify(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", "<br>")
	return s
}
