package openstack

import (
	"errors"
	"log/slog"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/loggers"
	"bitbucket.org/sudosweden/dockyards-backend/internal/util"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/util/ipam"
	"github.com/glebarez/sqlite"
	"github.com/google/go-cmp/cmp"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"gorm.io/gorm"
)

func TestCreateMetalLBDeployment(t *testing.T) {
	tt := []struct {
		name     string
		network  networks.Network
		cluster  v1.Cluster
		expected v1.Deployment
	}{
		{
			name: "test simple",
			network: networks.Network{
				Name: "test",
				Tags: []string{
					"asn=12345",
					"ipv4=1.2.3.4/31",
					"ipv6=123::4/127",
					"peer=169.254.169.254",
					"extratag=shouldbeignored",
					"shouldbeignored",
				},
			},
			cluster: v1.Cluster{
				ID: "cluster-123",
			},
			expected: v1.Deployment{
				Type:      v1.DeploymentTypeKustomize,
				ClusterID: "cluster-123",
				Name:      util.Ptr("metallb"),
				Namespace: util.Ptr("metallb-system"),
				Kustomize: &map[string][]byte{
					"bgpadvertisement.yaml": []byte(
						strings.Join([]string{
							"apiVersion: metallb.io/v1beta1",
							"kind: BGPAdvertisement",
							"metadata:",
							"  name: test",
							"spec:",
							"  ipAddressPools:",
							"  - test",
							"",
						}, "\n"),
					),
					"bgppeer.yaml": []byte(strings.Join([]string{
						"apiVersion: metallb.io/v1beta1",
						"kind: BGPPeer",
						"metadata:",
						"  name: test",
						"spec:",
						"  ebgpMultiHop: true",
						"  myASN: 12345",
						"  peerASN: 64700",
						"  peerAddress: 169.254.169.254",
						"",
					}, "\n")),
					"ipaddresspool.yaml": []byte(strings.Join([]string{
						"apiVersion: metallb.io/v1beta1",
						"kind: IPAddressPool",
						"metadata:",
						"  name: test",
						"spec:",
						"  addresses:",
						"  - 1.2.3.4/32",
						"  - 123::4/128",
						"",
					}, "\n")),
					"kustomization.yaml": []byte(strings.Join([]string{
						"apiVersion: kustomize.config.k8s.io/v1beta1",
						"kind: Kustomization",
						"patches:",
						"- patch: |-",
						"    - op: add",
						"      path: /spec/template/spec/nodeSelector/node-role.dockyards.io~1load-balancer",
						"      value: \"\"",
						"    - op: add",
						"      path: /spec/template/spec/tolerations/-",
						"      value:",
						"        effect: NoSchedule",
						"        key: node-role.dockyards.io/load-balancer",
						"        operator: Exists",
						"  target:",
						"    kind: DaemonSet",
						"    name: speaker",
						"resources:",
						"- github.com/metallb/metallb/config/frr?ref=v0.13.11",
						"- bgppeer.yaml",
						"- ipaddresspool.yaml",
						"- bgpadvertisement.yaml",
						"",
					}, "\n")),
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
			gormSlogger := loggers.NewGormSlogger(logger)

			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormSlogger, TranslateError: true})
			if err != nil {
				t.Fatalf("unexpected error creating test database: %s", err)
			}

			ipManagerOptions := []ipam.ManagerOption{
				ipam.WithLogger(logger),
				ipam.WithDB(db),
			}
			ipManager, err := ipam.NewIPManager(ipManagerOptions...)
			if err != nil {
				t.Fatalf("unexpected error creating test ip manager: %s", err)
			}

			s := openStackService{
				logger:    logger,
				db:        db,
				ipManager: ipManager,
			}

			actual, err := s.createMetalLBDeployment(&tc.network, &tc.cluster)
			if err != nil {
				t.Fatalf("error creating metallb deployment: %s", err)
			}

			if !cmp.Equal(actual, &tc.expected) {
				t.Errorf("diff: %s", cmp.Diff(&tc.expected, actual))
			}
		})
	}
}

func TestCreateClusterMetalLBDeploymentErrors(t *testing.T) {
	tt := []struct {
		name        string
		network     networks.Network
		cluster     v1.Cluster
		prefix      netip.Prefix
		allocations int
		expected    error
	}{
		{
			name: "test missing tag asn",
			network: networks.Network{
				Name: "test",
				Tags: []string{
					"ipv4=1.2.3.4/32",
					"ipv6=123::4/128",
					"peer=169.254.169.254",
				},
			},
			expected: ErrTagMissingASN,
		},
		{
			name: "test missing tag peer",
			network: networks.Network{
				Name: "test",
				Tags: []string{
					"asn=12345",
					"ipv4=1.2.3.4/32",
					"ipv6=123::4/128",
				},
			},
			expected: ErrTagMissingPeer,
		},
		{
			name: "test missing addresses",
			network: networks.Network{
				Name: "test",
				Tags: []string{
					"asn=12345",
					"peer=169.254.169.254",
				},
			},
		},
		{
			name: "test full prefix ipv4",
			network: networks.Network{
				Name: "test",
				Tags: []string{
					"ipv4=1.2.3.4/31",
				},
			},
			prefix:      netip.MustParsePrefix("1.2.3.4/31"),
			allocations: 2,
			expected:    ipam.ErrPrefixFull,
		},
		{
			name: "test full prefix ipv6",
			network: networks.Network{
				Name: "test",
				Tags: []string{
					"ipv6=123::4/126",
				},
			},
			prefix:      netip.MustParsePrefix("123::4/126"),
			allocations: 4,
			expected:    ipam.ErrPrefixFull,
		},
		{
			name: "test invalid asn",
			network: networks.Network{
				Name: "test",
				Tags: []string{
					"asn=1234.5",
				},
			},
			expected: strconv.ErrSyntax,
		},
		{
			name: "test invalid ipv4 prefix",
			network: networks.Network{
				Name: "test",
				Tags: []string{
					"ipv4=1.2.3.4",
				},
			},
		},
		{
			name: "test invalid ipv6 prefix",
			network: networks.Network{
				Name: "test",
				Tags: []string{
					"ipv4=123::4",
				},
			},
		},
		{
			name: "test invalid peer ipv4",
			network: networks.Network{
				Name: "test",
				Tags: []string{
					"peer=1.2.3.4/32",
				},
			},
		},
		{
			name: "test invalid peer ipv6",
			network: networks.Network{
				Name: "test",
				Tags: []string{
					"peer=123::4/128",
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError + 1}))
			gormSlogger := loggers.NewGormSlogger(logger)

			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormSlogger, TranslateError: true})
			if err != nil {
				t.Fatalf("unexpected error creating test database: %s", err)
			}

			ipManagerOptions := []ipam.ManagerOption{
				ipam.WithLogger(logger),
				ipam.WithDB(db),
			}
			ipManager, err := ipam.NewIPManager(ipManagerOptions...)
			if err != nil {
				t.Fatalf("unexpected error creating test ip manager: %s", err)
			}
			for i := 0; i < tc.allocations; i++ {
				_, err := ipManager.AllocateAddr(tc.prefix)
				if err != nil {
					t.Fatalf("unexpected error allocting addr from test ip manager: %s", err)
				}
			}

			s := openStackService{
				logger:    logger,
				db:        db,
				ipManager: ipManager,
			}

			_, err = s.createMetalLBDeployment(&tc.network, &tc.cluster)
			if tc.expected != nil && !errors.Is(err, tc.expected) {
				t.Errorf("expected error '%s', got '%s'", tc.expected, err)
			}

			if err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	}
}
