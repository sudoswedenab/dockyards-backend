package internal

import (
	"strings"
	"unicode"
)

func isUpper(r rune) bool {
	return unicode.IsUpper(r)
}

func IsValidName(name string) bool {
	if len(name) == 0 {
		return false
	}

	if len(name) > 63 {
		return false
	}

	if strings.HasPrefix(name, "-") {
		return false
	}

	if strings.HasSuffix(name, "-") {
		return false
	}

	if strings.IndexFunc(name, isUpper) != -1 {
		return false
	}

	return true
}
