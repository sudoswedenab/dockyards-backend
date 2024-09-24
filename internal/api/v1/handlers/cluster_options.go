package handlers

import (
	"encoding/json"
	"net/http"

	"bitbucket.org/sudosweden/dockyards-api/pkg/types"
	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1/middleware"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/apiutil"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/featurenames"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=dockyards.io,resources=clustertemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups=dockyards.io,resources=releases,verbs=get;list;watch

func (h *handler) GetClusterOptions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := middleware.LoggerFrom(ctx)

	objectKey := client.ObjectKey{
		Name:      dockyardsv1.ReleaseNameSupportedKubernetesVersions,
		Namespace: h.namespace,
	}

	var release dockyardsv1.Release
	err := h.Get(ctx, objectKey, &release)
	if err != nil {
		logger.Error("error getting release", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	options := types.Options{
		Version: release.Status.Versions,
	}

	featureEnabled, err := apiutil.IsFeatureEnabled(ctx, h.Client, featurenames.FeatureStorageRole, h.namespace)
	if err != nil {
		logger.Error("error verifying feature", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if featureEnabled {
		storageResourceTypes := []string{
			dockyardsv1.StorageResourceTypeHostPath,
		}

		options.StorageResourceTypes = &storageResourceTypes
	}

	b, err := json.Marshal(&options)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write(b)
	if err != nil {
		logger.Error("error writing response data", "err", err)
	}
}
