package rancher

import (
	"errors"

	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"github.com/rancher/norman/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *rancher) GetNodes(nodePool *v1alpha1.NodePool) (*v1alpha1.NodeList, error) {
	if nodePool.Status.ClusterServiceID == "" {
		return nil, errors.New("node pool has empty cluster service id")
	}

	listOpts := types.ListOpts{
		Filters: map[string]any{
			"nodePoolId": nodePool.Status.ClusterServiceID,
		},
	}

	nodes, err := r.managementClient.Node.ListAll(&listOpts)
	if err != nil {
		return nil, err
	}

	nodeList := v1alpha1.NodeList{
		Items: make([]v1alpha1.Node, len(nodes.Data)),
	}

	for i, node := range nodes.Data {
		nodeList.Items[i] = v1alpha1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: node.NodeName,
			},
			Status: v1alpha1.NodeStatus{
				ClusterServiceID: node.ID,
			},
		}
	}

	return &nodeList, nil
}
