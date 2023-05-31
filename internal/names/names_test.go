package names

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
