package middleware

import (
	"bitbucket.org/sudosweden/dockyards-api/pkg/types"
)

#_objectName: =~"^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"

#clusterOptions: types.#ClusterOptions
#login:          types.#Login
#workload:       types.#Workload
#workload: namespace!:              #_objectName
#workload: workload_template_name!: #_objectName
