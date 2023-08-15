package openstack

import (
	"log/slog"
	"os"
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
)

func TestGetClosestFlavorID(t *testing.T) {
	tt := []struct {
		name            string
		flavors         []flavors.Flavor
		nodePoolOptions model.NodePoolOptions
		expected        string
	}{
		{
			name: "test empty",
		},
		{
			name: "test exact match",
			flavors: []flavors.Flavor{
				{
					ID:    "cpu-123",
					Disk:  10,
					RAM:   1024,
					VCPUs: 2,
				},
				{
					ID:    "ram-123",
					Disk:  10,
					RAM:   2048,
					VCPUs: 1,
				},
				{
					ID:    "disk-123",
					Disk:  100,
					RAM:   2048,
					VCPUs: 1,
				},
			},
			nodePoolOptions: model.NodePoolOptions{
				DiskSizeGB: 10,
				RAMSizeMB:  2048,
				CPUCount: 1,
			},
			expected: "ram-123",
		},
		{
			name: "test cpu match",
			flavors: []flavors.Flavor{
				{
					ID:    "cpu-123",
					Disk:  5,
					RAM:   1024,
					VCPUs: 2,
				},
				{
					ID:    "ram-123",
					Disk:  10,
					RAM:   2048,
					VCPUs: 1,
				},
				{
					ID:    "disk-123",
					Disk:  100,
					RAM:   8192,
					VCPUs: 4,
				},
			},
			nodePoolOptions: model.NodePoolOptions{
				CPUCount: 3,
			},
			expected: "cpu-123",
		},
		{
			name: "test ram match",
			flavors: []flavors.Flavor{
				{
					ID:    "cpu-123",
					Disk:  10,
					RAM:   1024,
					VCPUs: 2,
				},
				{
					ID:    "ram-123",
					Disk:  10,
					RAM:   2048,
					VCPUs: 1,
				},
				{
					ID:    "disk-123",
					Disk:  100,
					RAM:   2048,
					VCPUs: 1,
				},
			},
			nodePoolOptions: model.NodePoolOptions{
				RAMSizeMB: 2000,
			},
			expected: "ram-123",
		},
		{
			name: "test disk match",
			flavors: []flavors.Flavor{
				{
					ID:    "cpu-123",
					Disk:  10,
					RAM:   1,
					VCPUs: 2,
				},
				{
					ID:    "ram-123",
					Disk:  10,
					RAM:   2,
					VCPUs: 1,
				},
				{
					ID:    "disk-123",
					Disk:  100,
					RAM:   2,
					VCPUs: 1,
				},
			},
			nodePoolOptions: model.NodePoolOptions{
				DiskSizeGB: 75,
			},
			expected: "disk-123",
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError + 1}))

	s := openStackService{
		logger: logger,
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			actual := s.getClosestFlavorID(tc.flavors, &tc.nodePoolOptions)
			if actual != tc.expected {
				t.Errorf("expected '%s', got '%s'", tc.expected, actual)
			}
		})
	}
}
