package rancher

import (
	"errors"

	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"github.com/rancher/norman/clientbase"
	"github.com/rancher/norman/types"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
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
				Name: node.RequestedHostname,
			},
			Status: v1alpha1.NodeStatus{
				ClusterServiceID: node.ID,
			},
		}
	}

	return &nodeList, nil
}

func (r *rancher) getNodeCondition(nodeConditions []managementv3.NodeCondition, conditionType string) *managementv3.NodeCondition {
	for i, nodeCondition := range nodeConditions {
		if nodeCondition.Type == conditionType {
			return &nodeConditions[i]
		}
	}

	return nil
}

func (r *rancher) ignoreNotFound(err error) error {
	if clientbase.IsNotFound(err) {
		return nil
	}

	return err
}

func (r *rancher) GetNode(nodeID string) (*v1alpha1.NodeStatus, error) {
	if nodeID == "" {
		return nil, errors.New("node id must not be empty")
	}

	node, err := r.managementClient.Node.ByID(nodeID)
	if r.ignoreNotFound(err) != nil {
		return nil, err
	}

	if clientbase.IsNotFound(err) {
		return nil, nil
	}

	nodeStatus := v1alpha1.NodeStatus{
		ClusterServiceID: node.ID,
		CloudServiceID:   node.ProviderId,
	}

	provisionedCondition := r.getNodeCondition(node.Conditions, "Provisioned")
	if provisionedCondition != nil {
		condition := metav1.Condition{
			Type:    v1alpha1.ProvisionedCondition,
			Status:  metav1.ConditionStatus(provisionedCondition.Status),
			Reason:  v1alpha1.NodeProvisionedReason,
			Message: provisionedCondition.Message,
		}

		nodeStatus.Conditions = append(nodeStatus.Conditions, condition)
	}

	readyCondition := r.getNodeCondition(node.Conditions, "Ready")
	if readyCondition != nil {
		condition := metav1.Condition{
			Type:    v1alpha1.ReadyCondition,
			Status:  metav1.ConditionStatus(readyCondition.Status),
			Reason:  v1alpha1.NodeReadyReason,
			Message: readyCondition.Message,
		}

		nodeStatus.Conditions = append(nodeStatus.Conditions, condition)
	}

	return &nodeStatus, nil
}
