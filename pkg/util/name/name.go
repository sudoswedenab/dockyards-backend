package name

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

func encodeString(s string) string {
	return strings.ReplaceAll(s, "-", "--")
}

func decodeString(s string) string {
	return strings.ReplaceAll(s, "--", "-")
}

func EncodeName(org, cluster string) string {
	encodedOrg := encodeString(org)
	encodedCluster := encodeString(cluster)
	return encodedOrg + "-" + encodedCluster
}

func DecodeName(s string) (string, string) {
	var split [2]string
	i := 0
	t := len(s) - 1
	for i < t {
		if s[i] == '-' {
			if s[i-1] != '-' && s[i+1] != '-' {
				split[0] = s[0:i]
				split[1] = s[i+1:]
				break
			}
		}
		i += 1
	}

	// name has no dash in it, name is for a cluster without org
	if split[0] == "" {
		return "", s
	}

	decodedOrg := decodeString(split[0])
	decodedName := decodeString(split[1])

	return decodedOrg, decodedName
}
