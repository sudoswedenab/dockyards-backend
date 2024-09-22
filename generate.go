package generate

//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen crd:generateEmbeddedObjectMeta=true object:headerFile="hack/header.txt" rbac:roleName=dockyards-backend webhook paths="./..."
