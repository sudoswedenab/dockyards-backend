package internal

import "testing"

func TestIsValidName(t *testing.T) {
	tt := []struct {
		name     string
		test     string
		expected bool
	}{
		{
			name: "test empty",
		},
		{
			name:     "test simple",
			test:     "simple",
			expected: true,
		},
		{
			name:     "test with dash prefix",
			test:     "-test",
			expected: false,
		},
		{
			name:     "test with dash suffix",
			test:     "-test",
			expected: false,
		},
		{
			name:     "test with dash infix",
			test:     "my-test",
			expected: true,
		},
		{
			name:     "test with title casing",
			test:     "Test",
			expected: false,
		},
		{
			name:     "test with camel casing",
			test:     "MyTest",
			expected: false,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			actual := IsValidName(tc.test)
			if actual != tc.expected {
				t.Errorf("expected string '%s' to be %t, got %t", tc.test, tc.expected, actual)
			}
		})
	}
}
