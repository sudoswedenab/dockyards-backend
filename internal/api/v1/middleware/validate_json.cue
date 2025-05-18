package middleware

import (
	"encoding/base64"

	"bitbucket.org/sudosweden/dockyards-api/pkg/types"
)

#_objectName: =~"^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"

#clusterOptions: types.#ClusterOptions
#login:          types.#Login

#workloadOptions: types.#WorkloadOptions
#workloadOptions: name!:                   #_objectName
#workloadOptions: namespace?:              #_objectName
#workloadOptions: workload_template_name!: #_objectName

#nodePoolOptions: types.#NodePoolOptions
#nodePoolOptions: name!:    #_objectName
#nodePoolOptions: quantity: >=0
#nodePoolOptions: storage_resources: [
	{name!: #_objectName},
]

#_base64Bytes: b={
	string
	#valid: base64.Decode(null, b)
}

#createCredential: name!:                     #_objectName
#createCredential: credential_template_name?: #_objectName
#createCredential: data: null | {[string]: null | #_base64Bytes}

#updateCredential: name?:                     _|_
#updateCredential: credential_template_name?: _|_
#updateCredential: data: null | {[string]: null | #_base64Bytes}

#updateOrganization: types.#OrganizationOptions
#updateOrganization: voucher_code?: _|_

#createInvitation: types.#InvitationOptions
#createInvitation: role!: "SuperUser" | "User" | "Reader"
