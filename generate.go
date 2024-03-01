package generate

//go:generate oapi-codegen -config hack/v1/types.cfg.yaml hack/v1/spec.yaml

//go:generate controller-gen crd:generateEmbeddedObjectMeta=true object:headerFile="hack/header.txt" rbac:roleName=dockyards-backend paths="./..."
