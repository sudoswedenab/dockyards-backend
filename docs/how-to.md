## Table of Contents:

- [How to configure dockyards workload clusters to use OIDC for authentication](#how-to-configure-dockyards-workload-clusters-to-use-oidc)
- [How to configure dockyards management cluster to use OIDC for authentication](#how-to-configure-dockyards-management-clusters-to-use-oidc)
- [How to configure workload cluster permissions](#how-to-configure-workload-cluster-permissions)

## How to configure dockyards management cluster to use OIDC for authentication

To configure the management cluster to use OIDC you will need to create a
[`IdentityProvider`](identity-provider-crd).

It looks something like this:

```yaml

apiVersion: dockyards.io/v1alpha3
kind: IdentityProvider
metadata:
  name: midgard-lab # Unique name identifying the identity provider.
spec:
  displayName: midgard_lab # Human readable name
  oidc: # Location where we're storing a identity provider secret
    name: midgard-lab
    namespace: dockyards-system

```

The IdentityProvider secret should look something like this:

```yaml

apiVersion: v1
kind: Secret
metadata:
  name: midgard-lab
  namespace: dockyards-system
type: Opaque
stringData:
  clientConfig: |
    {
      "clientID": "midgard_lab", // OIDC client id.
      "clientSecret": "YOUR_OIDC_CLIENT_SECRET",
      "redirectURL": "https://{DOCKYARDS_DOMAIN}/api/backend/v1/callback-sso"
    }
  providerConfig: |
    {
      "issuer":  // "https://{IDENTITY_PROVIDER_DOMAIN}/realms/{REALM_NAME}",
      "authorization_endpoint": "https://{IDENTITY_PROVIDER_DOMAIN}/realms/{REALM_NAME}/protocol/openid-connect/auth",
      "token_endpoint": "https://{IDENTITY_PROVIDER_DOMAIN}/realms/{REALM_NAME}/protocol/openid-connect/token",
      "device_authorization_endpoint": "https://{IDENTITY_PROVIDER_DOMAIN}/realms/{REALM_NAME}/protocol/openid-connect/auth/device",
      "userinfo_endpoint": "https://{IDENTITY_PROVIDER_DOMAIN}/realms/{REALM_NAME}/protocol/openid-connect/userinfo",
      "jwks_uri": "https://{IDENTITY_PROVIDER_DOMAIN}/realms/{REALM_NAME}/protocol/openid-connect/certs",
      "id_token_signing_alg_values_supported": [
        "PS384",
        "RS384",
        "EdDSA",
        "ES384",
        "HS256",
        "HS512",
        "ES256",
        "RS256",
        "HS384",
        "ES512",
        "PS256",
        "PS512",
        "RS512"
      ]
    }

```

The values for the `providerConfig` can in the case of Keycloak be be found at `https://{IDENTITY_PROVIDER_DOMAIN}/realms/{REALM}/.well-known/openid-configuration`.

When you have created the above objects, you should be able to log in to
the platform using the `GET /v1/login-sso` API endpoint.

The user will be created lazily upon sign in and will not be a part of any
organization until it's either invited somewhere or added via to an
organization via [`dockyards-ldap`][dockyards-ldap].

[identity-provider-crd]: https://github.com/sudoswedenab/dockyards-backend/blob/main/config/crd/dockyards.io_identityproviders.yaml
[dockyards-ldap]: https://github.com/sudoswedenab/dockyards-ldap

## How to configure dockyards workload clusters to use OIDC for authentication

To configure dockyards workload clusters with OIDC, you will need to
provide the [`authenticationConfig`][auth-config] option when creating
a cluster via the `POST /v1/orgs/{organizationName}/clusters` endpoint.
This will ensure that a Kubernetes [AuthenticationConfiguration][k8s-auth-config]
is supplied to the `kube-apiserver` of the cluster.

If you want use an [`AuthenticationConfiguration`][k8s-auth-config] like
this:

```yaml

apiVersion: apiserver.config.k8s.io/v1
kind: AuthenticationConfiguration
jwt:
- issuer:
    url: "ISSUER_URL" # (e.g. https://{DOMAIN}/realms/{REALM} in the case of keycloak)
    audiences:
    - midgard # Whatever is in the aud field of the JWT token.
    audienceMatchPolicy: MatchAny
  claimMappings:
    username:
      claim: "preferred_username"
      prefix: "oidc:"
    groups:
      expression: "claims.roles.split(',')"
    uid:
      claim: "sub"
  userValidationRules:
  - expression: "!user.username.startsWith('system:')"
    message: "username cannot use reserved system: prefix"
  - expression: "user.groups.all(group, !group.startsWith('system:'))"
    message: "groups cannot use reserved system: prefix"

```

You will have to provide the following JSON value in the `authenticationConfig` field
of the POST request body:

```json

{
  "jwt": [
    {
      "issuer": "ISSUER_URL" // (e.g. https://{DOMAIN}/realms/{REALM} in the case of keycloak)
      "audiences": [
        "midgard" // Whatever is in the aud field of the JWT token.
      ],
      "audience_match_policy": "MatchAny",
      "claim_mappings": {
        "username": {
          "claim": "preferred_username",
          "prefix": "oidc:"
        },
        "groups": {
          "expression": "claims.roles.split(',')"
        },
        "uid": {
          "claim": "sub"
        }
      },
      "user_validation_rules": [
        {
          "expression": "!user.username.startsWith('system:')",
          "message": "username cannot use reserved system: prefix"
        },
        {
          "expression": "user.groups.all(group, !group.startsWith('system:'))",
          "message": "groups cannot use reserved system: prefix"
        }
      ]
    }
  ]
}

```

[auth-config]: https://github.com/sudoswedenab/dockyards-api/blob/265894d40de99945ec91f7e47bdc37599f4ca763/spec/types.yaml#L592-L625
[k8s-auth-config]: https://kubernetes.io/docs/reference/access-authn-authz/authentication/#using-authentication-configuration

## How to configure workload cluster permissions

To configure workload cluster permissions you will need to create a
[git repository workload][git-repo-workload] which contains the [Role][role]s
and [RoleBinding][role-binding]s you want to use, alternatively you may use
[ClusterRole][cluster-role] and [ClusterRoleBinding][cluster-role-binding]
depending on if you want to scope user permissions cluster wide or per
namespace within the cluster. See [kubernetes documentation][k8s-rbac-docs] for
more details.

Here is a couple of examples of `ClusterRole` and `ClusterRoleBinding`:

```yaml

apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: reader
rules:
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "watch", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: reader
subjects:
- kind: Group
  apiGroup: rbac.authorization.k8s.io
  name: APP000001234_MIDGARD_RO # This is whatever the corresponding OIDC group is called.
roleRef:
  kind: ClusterRole
  apiGroup: rbac.authorization.k8s.io
  name: reader # This is the role we want it to correspond to.

```

In case you want to do more fine grained access control, you may use `Role` and `RoleBinding` instead:

```yaml

apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: reader
  namespace: foobar # The namespace this pertains to.
rules:
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "watch", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: reader
  namespace: foobar # The namespace this pertains to.
subjects:
- kind: Group
  apiGroup: rbac.authorization.k8s.io
  name: APP000001234_MIDGARD_RO_FOOBAR # This is whatever the corresponding OIDC group is called.
roleRef:
  kind: Role
  apiGroup: rbac.authorization.k8s.io
  name: reader # This is the role we want it to correspond to.

```

To create a [git repository workload][git-repo-workload] you will need to
call the create cluster workload endpoint (`POST /v1/orgs/{organizationName}/clusters/{clusterName}/workloads`)
with something like the following body:

```json

{
  "workload_template_name": "git-repository",
  "name": "permissions", // Or whatever name you want
  "input": {
    "url": "URL TO GIT REPOSITORY",
    "interval": "5m",
    "path": ".",
    "ref": {
      "branch": "main"
    }
  }
}

```

[git-repo-workload]: https://github.com/sudoswedenab/dockyards-workload-templates/blob/main/templates/git-repository/git-repository.cue
[role]: https://kubernetes.io/docs/reference/access-authn-authz/rbac/#role-example
[role-binding]: https://kubernetes.io/docs/reference/access-authn-authz/rbac/#rolebinding-and-clusterrolebinding
[cluster-role]: https://kubernetes.io/docs/reference/access-authn-authz/rbac/#clusterrole-example
[cluster-role-binding]: https://kubernetes.io/docs/reference/access-authn-authz/rbac/#rolebinding-and-clusterrolebinding
[k8s-rbac-docs]: https://kubernetes.io/docs/reference/access-authn-authz/rbac
