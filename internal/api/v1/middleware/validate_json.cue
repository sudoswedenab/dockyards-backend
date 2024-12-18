package middleware

import (
	"bitbucket.org/sudosweden/dockyards-api/pkg/types"
)

#_objectName: =~"^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"

#clusterOptions: types.#ClusterOptions
#login:          types.#Login

#workload: types.#Workload
#workload: name!:                   #_objectName
#workload: namespace?:              #_objectName
#workload: workload_template_name!: #_objectName

#nodePoolOptions: types.#NodePoolOptions
#nodePoolOptions: name!:    #_objectName
#nodePoolOptions: quantity: >=0
#nodePoolOptions: storage_resources: [
	{name!: #_objectName},
]

#credential: types.#Credential
#credential: name!: #_objectName
