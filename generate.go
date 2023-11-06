package generate

//go:generate oapi-codegen -config haxx/v1/types.cfg.yaml haxx/v1/spec.yaml

//go:generate controller-gen crd object:headerFile="haxx/header.txt" rbac:roleName=dockyards-backend paths="./..."
