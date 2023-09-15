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

func TestEncodeDecodeString(t *testing.T) {
	tt := []struct {
		name     string
		test     string
		expected string
	}{
		{
			name: "test empty",
		},
		{
			name:     "test simple",
			test:     "simple",
			expected: "simple",
		},
		{
			name:     "test single dash",
			test:     "single-dash",
			expected: "single--dash",
		},
		{
			name:     "test multiple dashes",
			test:     "test-multiple-dashes",
			expected: "test--multiple--dashes",
		},
		{
			name:     "test consecutive dashes",
			test:     "consecutive--dashes",
			expected: "consecutive----dashes",
		},
		{
			name:     "test several consecutive dashes",
			test:     "consecutive----dashes",
			expected: "consecutive--------dashes",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			actual := encodeString(tc.test)
			if actual != tc.expected {
				t.Errorf("expected encoded '%s', got '%s'", tc.expected, actual)
			}

			actual = decodeString(tc.expected)
			if actual != tc.test {
				t.Errorf("expected decoded '%s', got '%s'", tc.test, actual)
			}
		})
	}
}

func TestDecodeName(t *testing.T) {
	tt := []struct {
		name            string
		rancherName     string
		expectedOrg     string
		expectedCluster string
	}{
		{
			name: "test empty",
		},
		{
			name:            "test simple",
			rancherName:     "test-simple",
			expectedOrg:     "test",
			expectedCluster: "simple",
		},
		{
			name:            "test org with dashes",
			rancherName:     "test--org-cluster",
			expectedOrg:     "test-org",
			expectedCluster: "cluster",
		},
		{
			name:            "test cluster with dashes",
			rancherName:     "test-my--cluster",
			expectedOrg:     "test",
			expectedCluster: "my-cluster",
		},
		{
			name:            "test org and cluster with dashes",
			rancherName:     "test--org-test--cluster",
			expectedOrg:     "test-org",
			expectedCluster: "test-cluster",
		},
		{
			name:            "test local",
			rancherName:     "local",
			expectedOrg:     "",
			expectedCluster: "local",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			actualOrg, actualCluster := DecodeName(tc.rancherName)
			if actualOrg != tc.expectedOrg {
				t.Errorf("expected org '%s', got '%s'", tc.expectedOrg, actualOrg)
			}
			if actualCluster != tc.expectedCluster {
				t.Errorf("expected name '%s', got '%s'", tc.expectedCluster, actualCluster)
			}
		})
	}
}

func TestEncodeName(t *testing.T) {
	tt := []struct {
		name     string
		org      string
		cluster  string
		expected string
	}{
		{
			name:     "test empty",
			expected: "-",
		},
		{
			name:     "test simple",
			org:      "test",
			cluster:  "simple",
			expected: "test-simple",
		},
		{
			name:     "test org with dashes",
			org:      "my-test",
			cluster:  "cluster",
			expected: "my--test-cluster",
		},
		{
			name:     "test cluster with dashes",
			org:      "test",
			cluster:  "my-cluster",
			expected: "test-my--cluster",
		},
		{
			name:     "test org and cluster with dashes",
			org:      "my-cool",
			cluster:  "test-cluster",
			expected: "my--cool-test--cluster",
		},
		{
			name:     "test org with double dashes",
			org:      "my--org",
			cluster:  "test",
			expected: "my----org-test",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			actual := EncodeName(tc.org, tc.cluster)
			if actual != tc.expected {
				t.Errorf("expected %s, got %s", tc.expected, actual)
			}
		})
	}
}
