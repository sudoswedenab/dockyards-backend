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
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/util/ipam"
	openstackv1alpha1 "bitbucket.org/sudosweden/dockyards-openstack/api/v1alpha1"
	"github.com/glebarez/sqlite"
	"github.com/google/go-cmp/cmp"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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
		tag         string
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
				_, err := ipManager.AllocateAddr(tc.prefix, tc.tag)
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

func TestGetClusterDeployments(t *testing.T) {
	tt := []struct {
		name         string
		organization v1alpha1.Organization
		cluster      v1.Cluster
		lists        []client.ObjectList
	}{
		{
			name: "test without loadbalancer",
			organization: v1alpha1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "testing",
					UID:       "9da90bc2-f0bd-42ed-9409-9dd3f5da0c66",
				},
				Spec: v1alpha1.OrganizationSpec{
					CloudRef: &v1alpha1.CloudReference{
						APIVersion: openstackv1alpha1.GroupVersion.String(),
						Kind:       openstackv1alpha1.OpenstackProjectKind,
						Name:       "project",
						SecretRef:  "project",
					},
				},
			},
			lists: []client.ObjectList{
				&openstackv1alpha1.OpenstackProjectList{
					Items: []openstackv1alpha1.OpenstackProject{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "project",
								Namespace: "testing",
								UID:       "9852c3cf-d455-4d29-9aa9-d6586f681f1f",
							},
							Spec: openstackv1alpha1.OpenstackProjectSpec{
								IdentityEndpoint: "http://localhost:5000/v3",
								ProjectID:        "0e0a09b79e277bc0a8262cc2b4a7b688",
								SecretRef: &corev1.ObjectReference{
									APIVersion: "v1",
									Kind:       "Secret",
									Name:       "project",
									UID:        "41e492df-3933-467b-a598-50e3a067f9b8",
								},
							},
						},
					},
				},
				&corev1.SecretList{
					Items: []corev1.Secret{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "project",
								Namespace: "testing",
								UID:       "41e492df-3933-467b-a598-50e3a067f9b8",
							},
							Data: map[string][]byte{},
						},
					},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError + 1}))

			scheme := scheme.Scheme
			v1alpha1.AddToScheme(scheme)
			openstackv1alpha1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(tc.lists...).Build()

			s := openStackService{
				logger:           logger,
				controllerClient: fakeClient,
			}

			_, err := s.GetClusterDeployments(&tc.organization, &tc.cluster)
			if err != nil {
				t.Errorf("error getting cluster deployments: %s", err)
			}
		})
	}
}
