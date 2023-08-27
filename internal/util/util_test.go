package util

import (
	"testing"
)

func TestPtr(t *testing.T) {
	tt := []struct {
		name  string
		value any
	}{
		{
			name: "test empty",
		},
		{
			name:  "test bool",
			value: true,
		},
		{
			name:  "test int",
			value: 1,
		},
		{
			name:  "test string",
			value: "test",
		},
		{
			name:  "test int64",
			value: int64(123),
		},
		{
			name:  "test float",
			value: 1.23,
		},
		{
			name:  "test struct",
			value: struct{ test bool }{test: true},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			actual := Ptr(tc.value)
			if *actual != tc.value {
				t.Errorf("error")
			}
		})

	}
}
