package generate

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=hack/v1/types.cfg.yaml hack/v1/spec.yaml

//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen crd:generateEmbeddedObjectMeta=true object:headerFile="hack/header.txt" rbac:roleName=dockyards-backend webhook paths="./..."
