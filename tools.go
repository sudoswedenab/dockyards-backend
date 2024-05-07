//go:build tools
// +build tools

package generate

import (
	_ "github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen"
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
)
