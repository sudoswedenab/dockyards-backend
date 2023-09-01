package sudo

import (
	"bytes"
	"context"

	"sigs.k8s.io/yaml"
)

func (a *sudoAPI) GetKubeconfig(ctx context.Context, req GetKubeconfigRequestObject) (GetKubeconfigResponseObject, error) {
	kubeconfig, err := a.clusterService.GetKubeconfig(req.ClusterID, 0)
	if err != nil {
		a.logger.Error("error getting kubeconfig", "err", err)

		return GetKubeconfig500Response{}, nil
	}

	b, err := yaml.Marshal(kubeconfig)
	if err != nil {
		a.logger.Error("error marshalling kubeconfig to yaml", "err", err)

		return GetKubeconfig500Response{}, nil
	}

	res := GetKubeconfig200TextplainCharsetUTF8Response{
		Body: bytes.NewBuffer(b),
	}
	return res, nil
}
