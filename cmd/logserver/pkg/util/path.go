package util

import "strings"

func ParseBasePath(path string) string {
	index := strings.IndexByte(path, '?')
	if index < 0 {
		return path
	}

	return path[:index]
}
