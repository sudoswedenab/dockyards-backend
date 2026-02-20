# How to configure workload cluster permissions

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

[git-repo-workload]: https://github.com/sudoswedenab/dockyards-workload-templates/blob/main/templates/git-repository/git-repository.cue
[role]: https://kubernetes.io/docs/reference/access-authn-authz/rbac/#role-example
[role-binding]: https://kubernetes.io/docs/reference/access-authn-authz/rbac/#rolebinding-and-clusterrolebinding
[cluster-role]: https://kubernetes.io/docs/reference/access-authn-authz/rbac/#clusterrole-example
[cluster-role-binding]: https://kubernetes.io/docs/reference/access-authn-authz/rbac/#rolebinding-and-clusterrolebinding
[k8s-rbac-docs]: https://kubernetes.io/docs/reference/access-authn-authz/rbac
