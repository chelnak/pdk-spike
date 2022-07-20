// Package stringutils contains utility functions for working with strings.
package stringutils

import "regexp"

// IsGitURL returns true if the given string is a valid git uri.
func IsGitURL(s string) bool {
	pattern := "^(?:git|ssh|git|http|https@|)(?:.*)(?:.*)(?:.git)$"
	reg := regexp.MustCompile(pattern)
	return reg.MatchString(s)
}

// IsTarGZ returns true if the given string is a tar.gz.
func IsTarGZ(s string) bool {
	pattern := "^(?:.*)(?:.tar.gz)$"
	reg := regexp.MustCompile(pattern)
	return reg.MatchString(s)
}
