package kustomization

_name: !=""
_name: string @tag(name)
_tag:  !=""
_tag:  string @tag(tag)

apiVersion: "kustomize.config.k8s.io/v1beta1"
configurations: [
	"kustomizeconfig.yaml",
]
images: [
	{
		name:    "dockyards-backend"
		newName: _name
		newTag:  _tag
	},
]
kind: "Kustomization"
patches: [
	{
		path: "patches/conversion.yaml"
		target: {
			kind: "CustomResourceDefinition"
			name: "organizations.dockyards.io"
		}
	},
]
resources: [
	"base",
	"crd",
	"rbac",
	"webhook",
]
