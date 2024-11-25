// Copyright 2024 Sudo Sweden AB
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package name

import (
	"strings"
)

const (
	detailEmptyName         = "name must not be empty"
	detailLongName          = "name must not contain more than 63 characters"
	detailDashPrefix        = "name must not begin with a dash character"
	detailDashSuffix        = "name must not end with a dash character"
	detailInvalidCharacters = "name must contain only lowercase alphanumeric characters and the '-' character"
)

func isInvalid(r rune) bool {
	if r == '-' {
		return false
	}

	if r >= '0' && r <= '9' {
		return false
	}

	if r >= 'a' && r <= 'z' {
		return false
	}

	return true
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

	if strings.IndexFunc(name, isInvalid) != -1 {
		return detailInvalidCharacters, false
	}

	return "", true
}
