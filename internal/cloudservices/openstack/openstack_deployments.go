package openstack

import (
	"errors"
	"net/netip"
	"strconv"
	"strings"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/util"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"github.com/gophercloud/gophercloud/openstack"
	networksv2 "github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"sigs.k8s.io/yaml"
)

var (
	ErrAddressesEmpty = errors.New("addresses is empty")
	ErrTagMissingASN  = errors.New("network missing tag asn")
	ErrTagMissingPeer = errors.New("network missing tag peer")
)

func (s *openStackService) createMetalLBDeployment(network *networksv2.Network, cluster *v1.Cluster) (*v1.Deployment, error) {
	addresses := make([]string, 0)
	bgpPeerSpec := map[string]any{
		"peerASN":      64700,
		"ebgpMultiHop": true,
	}

	for _, tag := range network.Tags {
		split := strings.Split(tag, "=")
		if len(split) != 2 {
			s.logger.Debug("ignoring tag split into more or less than two elements", "tag", tag)

			continue
		}

		switch split[0] {
		case "asn":
			asn, err := strconv.Atoi(split[1])
			if err != nil {
				s.logger.Error("error parsing asn as integer", "err", err)

				return nil, err
			}
			bgpPeerSpec["myASN"] = asn
		case "ipv4":
			fallthrough
		case "ipv6":
			prefix, err := netip.ParsePrefix(split[1])
			if err != nil {
				s.logger.Error("error parsing prefix", "err", err)

				return nil, err
			}

			addr, err := s.ipManager.AllocateAddr(prefix, network.ID)
			if err != nil {
				s.logger.Error("error allocating address", "err", err)

				return nil, err
			}

			bits := strconv.Itoa(addr.BitLen())
			addresses = append(addresses, addr.String()+"/"+bits)
		case "peer":
			peerAddr, err := netip.ParseAddr(split[1])
			if err != nil {
				s.logger.Error("error parsing peer as address", "err", err)

				return nil, err
			}

			bgpPeerSpec["peerAddress"] = peerAddr.String()
		default:
			s.logger.Debug("ignoring tag", "key", split[0])
		}
	}

	if len(addresses) == 0 {
		return nil, ErrAddressesEmpty
	}

	_, hasMyASN := bgpPeerSpec["myASN"]
	if !hasMyASN {
		return nil, ErrTagMissingASN
	}

	_, hasPeerAddress := bgpPeerSpec["peerAddress"]
	if !hasPeerAddress {
		return nil, ErrTagMissingPeer
	}

	ipAddressPool := map[string]any{
		"apiVersion": "metallb.io/v1beta1",
		"kind":       "IPAddressPool",
		"metadata": map[string]any{
			"name": network.Name,
		},
		"spec": map[string]any{
			"addresses": addresses,
		},
	}

	ipAddressPoolYAML, err := yaml.Marshal(ipAddressPool)
	if err != nil {
		return nil, err
	}

	bgpPeer := map[string]any{
		"apiVersion": "metallb.io/v1beta1",
		"kind":       "BGPPeer",
		"metadata": map[string]any{
			"name": network.Name,
		},
		"spec": bgpPeerSpec,
	}

	bgpPeerYAML, err := yaml.Marshal(bgpPeer)
	if err != nil {
		return nil, err
	}

	bgpAdvertisement := map[string]any{
		"apiVersion": "metallb.io/v1beta1",
		"kind":       "BGPAdvertisement",
		"metadata": map[string]any{
			"name": network.Name,
		},
		"spec": map[string]any{
			"ipAddressPools": []string{
				network.Name,
			},
		},
	}

	bgpAdvertisementYAML, err := yaml.Marshal(bgpAdvertisement)
	if err != nil {
		return nil, err
	}

	kustomization := map[string]any{
		"apiVersion": "kustomize.config.k8s.io/v1beta1",
		"kind":       "Kustomization",
		"resources": []string{
			"github.com/metallb/metallb/config/frr?ref=v0.13.11",
			"bgppeer.yaml",
			"ipaddresspool.yaml",
			"bgpadvertisement.yaml",
		},
		"patches": []map[string]any{
			{
				"patch": strings.Join([]string{
					"- op: add",
					"  path: /spec/template/spec/nodeSelector/node-role.dockyards.io~1load-balancer",
					"  value: \"\"",
					"- op: add",
					"  path: /spec/template/spec/tolerations/-",
					"  value:",
					"    effect: NoSchedule",
					"    key: node-role.dockyards.io/load-balancer",
					"    operator: Exists",
				}, "\n"),
				"target": map[string]any{
					"kind": "DaemonSet",
					"name": "speaker",
				},
			},
		},
	}

	kustomizationYAML, err := yaml.Marshal(kustomization)
	if err != nil {
		return nil, err
	}

	metalLBDeployment := v1.Deployment{
		Type:      v1.DeploymentTypeKustomize,
		ClusterID: cluster.ID,
		Name:      util.Ptr("metallb"),
		Namespace: util.Ptr("metallb-system"),
		Kustomize: &map[string][]byte{
			"kustomization.yaml":    kustomizationYAML,
			"ipaddresspool.yaml":    ipAddressPoolYAML,
			"bgppeer.yaml":          bgpPeerYAML,
			"bgpadvertisement.yaml": bgpAdvertisementYAML,
		},
	}

	return &metalLBDeployment, nil
}

func (s *openStackService) createIngressNginxDeployment(cluster *v1.Cluster) *v1.Deployment {
	ingressNginxDeployment := v1.Deployment{
		Type:           v1.DeploymentTypeHelm,
		ClusterID:      cluster.ID,
		Name:           util.Ptr("ingress-nginx"),
		Namespace:      util.Ptr("ingress-nginx"),
		HelmChart:      util.Ptr("ingress-nginx"),
		HelmRepository: util.Ptr("https://kubernetes.github.io/ingress-nginx"),
		HelmVersion:    util.Ptr("4.7.2"),
		HelmValues: &map[string]any{
			"controller": map[string]any{
				"kind": "DaemonSet",
				"hostPort": map[string]any{
					"enabled": true,
				},
				"ingressClassResource": map[string]any{
					"default": true,
				},
				"nodeSelector": map[string]any{
					"node-role.dockyards.io/load-balancer": "",
				},
				"tolerations": []map[string]any{
					{
						"key":      "node-role.dockyards.io/load-balancer",
						"operator": "Exists",
						"effect":   "NoSchedule",
					},
				},
			},
		},
	}

	return &ingressNginxDeployment
}

func (s *openStackService) GetClusterDeployments(organization *v1alpha1.Organization, cluster *v1.Cluster) (*[]v1.Deployment, error) {
	openstackProject, err := s.getOpenstackProject(organization)
	if err != nil {
		return nil, err
	}

	secret, err := s.getOpenstackSecret(organization)
	if err != nil {
		return nil, err
	}

	applicationCredentialID := secret.Data["applicationCredentialID"]
	applicationCredentialSecret := secret.Data["applicationCredentialSecret"]
	cloudConf := []string{
		"[Global]",
		"auth-url=" + s.authOptions.IdentityEndpoint,
		"application-credential-id=" + string(applicationCredentialID),
		"application-credential-secret=" + string(applicationCredentialSecret),
	}

	openStackCinderCSIDeployment := v1.Deployment{
		ClusterID:      cluster.ID,
		Name:           util.Ptr("openstack-cinder-csi"),
		Type:           v1.DeploymentTypeHelm,
		HelmChart:      util.Ptr("openstack-cinder-csi"),
		HelmRepository: util.Ptr("https://kubernetes.github.io/cloud-provider-openstack"),
		HelmVersion:    util.Ptr("2.28.0"),
		Namespace:      util.Ptr("kube-system"),
		HelmValues: util.Ptr(map[string]any{
			"secret": map[string]any{
				"enabled":  true,
				"create":   true,
				"filename": "cloud.conf",
				"name":     "cinder-csi-cloud-config",
				"data": map[string]interface{}{
					"cloud.conf": strings.Join(cloudConf, "\n"),
				},
			},
			"storageClass": map[string]any{
				"delete": map[string]any{
					"isDefault": true,
				},
			},
			"clusterID": cluster.Name,
		}),
	}

	clusterDeployments := []v1.Deployment{
		openStackCinderCSIDeployment,
	}

	needsLoadBalancer := false
	for _, nodePool := range cluster.NodePools {
		if nodePool.LoadBalancer != nil && *nodePool.LoadBalancer {
			s.logger.Debug("cluster has node pool load balancer", "id", nodePool.ID)

			needsLoadBalancer = true
			break
		}
	}

	if needsLoadBalancer {
		scopedClient, err := s.getScopedClient(openstackProject.Spec.ProjectID)
		if err != nil {
			return nil, err
		}

		networkv2, err := openstack.NewNetworkV2(scopedClient, s.endpointOpts)
		if err != nil {
			return nil, err
		}

		listOpts := networksv2.ListOpts{
			Name: "elasticip",
		}

		pager, err := networksv2.List(networkv2, &listOpts).AllPages()
		if err != nil {
			return nil, err
		}

		networks, err := networksv2.ExtractNetworks(pager)
		if len(networks) != 1 {
			return nil, errors.New("expected only 1 network")
		}

		metalLBDeployment, err := s.createMetalLBDeployment(&networks[0], cluster)
		if err != nil {
			return nil, err
		}

		ingressNginxDeployment := s.createIngressNginxDeployment(cluster)

		clusterDeployments = append(clusterDeployments, *metalLBDeployment, *ingressNginxDeployment)
	}

	return &clusterDeployments, nil
}
