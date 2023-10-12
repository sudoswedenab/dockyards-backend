package openstack

import (
	"log/slog"
	"os"
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestGetClosestFlavorID(t *testing.T) {
	tt := []struct {
		name     string
		flavors  []flavors.Flavor
		nodePool v1alpha1.NodePool
		expected string
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
			nodePool: v1alpha1.NodePool{
				Spec: v1alpha1.NodePoolSpec{
					Resources: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("10Gi"),
						corev1.ResourceMemory:  resource.MustParse("2Gi"),
						corev1.ResourceCPU:     resource.MustParse("1"),
					},
				},
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
			nodePool: v1alpha1.NodePool{
				Spec: v1alpha1.NodePoolSpec{
					Resources: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("1"),
					},
				},
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
			nodePool: v1alpha1.NodePool{
				Spec: v1alpha1.NodePoolSpec{
					Resources: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("2G"),
					},
				},
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
			nodePool: v1alpha1.NodePool{
				Spec: v1alpha1.NodePoolSpec{
					Resources: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("75Gi"),
					},
				},
			},
			expected: "disk-123",
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	s := openStackService{
		logger: logger,
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			actual := s.getClosestFlavorID(tc.flavors, &tc.nodePool)
			if actual != tc.expected {
				t.Errorf("expected '%s', got '%s'", tc.expected, actual)
			}
		})
	}
}
