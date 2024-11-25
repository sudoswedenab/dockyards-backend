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

import "testing"

func TestIsValidName(t *testing.T) {
	tt := []struct {
		name           string
		test           string
		expectedDetail string
		expected       bool
	}{
		{
			name:           "test empty",
			expectedDetail: detailEmptyName,
			expected:       false,
		},
		{
			name:     "test simple",
			test:     "simple",
			expected: true,
		},
		{
			name:           "test with dash prefix",
			test:           "-test",
			expectedDetail: detailDashPrefix,
			expected:       false,
		},
		{
			name:           "test with dash suffix",
			test:           "test-",
			expectedDetail: detailDashSuffix,
			expected:       false,
		},
		{
			name:     "test with dash infix",
			test:     "my-test",
			expected: true,
		},
		{
			name:           "test with title casing",
			test:           "Test",
			expectedDetail: detailInvalidCharacters,
			expected:       false,
		},
		{
			name:           "test with camel casing",
			test:           "MyTest",
			expectedDetail: detailInvalidCharacters,
			expected:       false,
		},
		{
			name:           "test with quotes",
			test:           "\"test'",
			expectedDetail: detailInvalidCharacters,
			expected:       false,
		},
		{
			name:           "test with semicolon",
			test:           "test;",
			expectedDetail: detailInvalidCharacters,
			expected:       false,
		},
		{
			name:     "test with number",
			test:     "test123",
			expected: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			actualDetail, actual := IsValidName(tc.test)
			if actual != tc.expected {
				t.Errorf("expected string '%s' to be %t, got %t", tc.test, tc.expected, actual)
			}

			if actualDetail != tc.expectedDetail {
				t.Errorf("expected detail '%s', got '%s'", tc.expectedDetail, actualDetail)
			}
		})
	}
}
