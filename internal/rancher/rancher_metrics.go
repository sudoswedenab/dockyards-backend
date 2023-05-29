package rancher

import "github.com/prometheus/client_golang/prometheus"

func (r *rancher) CollectMetrics() error {
	r.logger.Debug("collecting prometheus metrics")

	clusters, err := r.GetAllClusters()
	if err != nil {
		return err
	}

	r.clusterMetric.Reset()

	for _, cluster := range *clusters {
		if cluster.Name == "local" {
			continue
		}

		labels := prometheus.Labels{
			"name":              cluster.Name,
			"organization_name": cluster.Organization,
			"state":             cluster.State,
		}
		r.clusterMetric.With(labels).Set(1)
	}

	return nil
}
