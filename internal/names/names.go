package names

import (
	"strings"
	"unicode"
)

const (
	detailEmptyName         = "name must not be empty"
	detailLongName          = "name must not contain more than 63 characters"
	detailDashPrefix        = "name must not begin with a dash character"
	detailDashSuffix        = "name must not end with a dash character"
	detailInvalidCharacters = "name must contain only lowercase alphanumeric characters and the '-' character"
)

func isUpper(r rune) bool {
	return unicode.IsUpper(r)
}

func IsValidName(name string) (string, bool) {
	if len(name) == 0 {
		return detailEmptyName, false
	}

	if len(name) > 63 {
		return detailLongName, false
	}

	if strings.HasPrefix(name, "-") {
		return detailDashPrefix, false
	}

	if strings.HasSuffix(name, "-") {
		return detailDashSuffix, false
	}

	if strings.IndexFunc(name, isUpper) != -1 {
		return detailInvalidCharacters, false
	}

	return "", true
}
