package rancher

import (
	"time"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/util/name"
	"github.com/rancher/norman/types"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *rancher) clusterToModel(cluster *managementv3.Cluster) v1.Cluster {
	createdAt, _ := time.Parse(time.RFC3339, cluster.Created)
	organization, name := name.DecodeName(cluster.Name)

	c := v1.Cluster{
		Organization: organization,
		Name:         name,
		State:        cluster.State,
		NodeCount:    int(cluster.NodeCount),
		CreatedAt:    createdAt,
		ID:           cluster.ID,
	}

	if cluster.RancherKubernetesEngineConfig != nil {
		c.Version = cluster.RancherKubernetesEngineConfig.Version
	}

	return c
}

func (r *rancher) GetAllClusters() (*[]v1.Cluster, error) {
	clusterCollection, err := r.managementClient.Cluster.ListAll(&types.ListOpts{})
	if err != nil {
		return nil, err
	}

	clusters := []v1.Cluster{}
	for _, cluster := range clusterCollection.Data {
		c := r.clusterToModel(&cluster)
		clusters = append(clusters, c)
	}

	return &clusters, nil
}

func (r *rancher) getClusterCondition(clusterConditions []managementv3.ClusterCondition, conditionType string) *managementv3.ClusterCondition {
	for i, clusterCondition := range clusterConditions {
		if clusterCondition.Type == conditionType {
			return &clusterConditions[i]
		}
	}

	return nil
}

func (r *rancher) GetCluster(id string) (*v1alpha1.ClusterStatus, error) {
	cluster, err := r.managementClient.Cluster.ByID(id)
	if err != nil {
		return nil, err
	}

	clusterStatus := v1alpha1.ClusterStatus{
		ClusterServiceID: cluster.ID,
	}

	readyCondition := r.getClusterCondition(cluster.Conditions, "Ready")
	if readyCondition != nil {
		condition := metav1.Condition{
			Type:    v1alpha1.ReadyCondition,
			Status:  metav1.ConditionStatus(readyCondition.Status),
			Reason:  v1alpha1.ClusterReadyReason,
			Message: cluster.State,
		}

		clusterStatus.Conditions = []metav1.Condition{
			condition,
		}
	}

	if cluster.RancherKubernetesEngineConfig != nil {
		clusterStatus.Version = cluster.RancherKubernetesEngineConfig.Version
	}

	return &clusterStatus, nil
}
