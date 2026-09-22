# dockyards-backend

The Dockyards Backend is the central Kubernetes controller-based backend for the Dockyards multi-tenant platform. It manages organizations, clusters, workloads, users, invitations, and members through a set of custom controllers that reconcile Dockyards CRDs against the desired cluster state.

## Purpose

Dockyards-backend provides the control plane for the Dockyards platform, enabling:

- **Multi-tenancy**: Organizations are created as Kubernetes namespaces with role-based access control reconciled automatically.
- **Cluster lifecycle management**: Clusters (Talos-backed) are provisioned, upgraded, and DNS zones are managed via the `ClusterReconciler`.
- **Workload deployment**: Workloads are deployed and tracked across clusters using the `WorkloadReconciler` and associated inventory.
- **User identity & onboarding**: User sign-up flows, verification requests, and OIDC integration are handled by the `UserReconciler`.
- **Invitation & membership**: The platform supports invitation-based org membership with automatic RBAC role binding reconciliation via the `InvitationReconciler` and `MemberReconciler`.

## Architecture

```
flowchart TB
    subgraph API["Dockyards Backend API"]
        public[Public HTTP\n:9000]
        private[Private HTTP\n:9001 /metrics + /healthz]
    end

    subgraph Controllers["Controllers (reconcilers)"]
        org[OrganizationReconciler]
        cluster[ClusterReconciler]
        workload[WorkloadReconciler]
        user[UserReconciler]
        invitation[InvitationReconciler]
        member[MemberReconciler]
    end

    subgraph CRDs["Custom Resource Definitions"]
        orgCR[Organization]
        clusterCR[Cluster]
        workloadCR[Workload]
        userCR[User]
        invitationCR[Invitation]
        memberCR[Member]
        nodepoolCR[NodePool, NodeClass]
        dnszoneCR[DNSZone, DNSZoneClaim]
    end

    subgraph Webhooks["Admission Webhooks"]
        orgWH[OrganizationWebhook]
        clusterWH[ClusterWebhook]
        invitationWH[InvitationWebhook]
        memberWH[MemberWebhook]
        nodepoolWH[NodePoolWebhook]
        userWH[UserWebhook]
    end

    public --> API
    private --> API
    API --> org
    API --> cluster
    API --> workload
    API --> user
    API --> invitation
    API --> member

    org --> orgCR
    cluster --> clusterCR
    workload --> workloadCR
    user --> userCR
    invitation --> invitationCR
    member --> memberCR

    orgCR -.-> orgWH
    clusterCR -.-> clusterWH
    invitationCR -.-> invitationWH
    memberCR -.-> memberWH
    nodepoolCR -.-> nodepoolWH
    userCR -.-> userWH
```

## Controllers

Each controller watches a specific Dockyards CRD and ensures the corresponding resources are in the desired state.

### OrganizationReconciler

Watches `dockyardsv1.Organization` resources. On create/update it reconciles RBAC role bindings for the organization's namespace. On delete, it cleans up all dependent objects owned by the organization via owner references.

### ClusterReconciler

Watches `dockyardsv1.Cluster` resources. Manages cluster lifecycle operations including:
- Provisioning and upgrading clusters (via `reconcileClusterUpgrades`)
- Creating and managing DNS zones per cluster (`reconcileDNSZones`)
- Mapping DNS zone changes back to their owning clusters via label selectors

### WorkloadReconciler

Watches `dockyardsv1.Workload` resources. Handles workload deployment lifecycle and correlates workloads with inventory objects via label-based indexing.

### UserReconciler

Watches `dockyardsv1.User` resources. Manages:
- Non-Dockyards-provided user reconciliation (external identity)
- User verification flows via `VerificationRequest` CRDs
- Sign-up request lifecycle (`sign-up-{username}` VerificationRequests)

### InvitationReconciler

Watches `dockyardsv1.Invitation` resources. Processes invitation objects to onboard users into organizations.

### MemberReconciler

Watches `dockyardsv1.Member` resources. Reconciles:
- Authorization state for each member (`reconcileAuthorization`)
- Member information relative to their organization (`reconcileInfo`)

## CRDs (api/v1alpha3)

The backend defines the following custom resource types in the `dockyards.dockyards.io/v1alpha3` API group:

| Category | CRDs |
|---|---|
| **Core** | `Cluster`, `Organization`, `Workload`, `User`, `Member`, `Invitation` |
| **Infrastructure** | `NodePool`, `NodeClass`, `DNSZone`, `DNSZoneClaim` |
| **Deployment** | `HelmDeployment`, `KustomizeDeployment`, `ContainerImageDeployment`, `Release`, `Deployment` |
| **Templates** | `WorkloadTemplate`, `ClusterTemplate`, `CredentialTemplate` |
| **Identity** | `IdentityProvider`, `OrganizationVoucher`, `VerificationRequest` |
| **Inventory** | `WorkloadInventory`, `WorkTree` |
| **Other** | `Feature`, `Conditions`, `Const` |

## Webhooks

Validation and defaulting admission webhooks are provided for:
- `Cluster`, `Invitation`, `Member`, `NodePool`, `Organization`, `User`

Webhooks are enabled via the `--enable-webhooks` flag with a list of allowed domains.

## Key Configuration

| Flag | Default | Description |
|---|---|---|
| `--dockyards-namespace` | `dockyards-system` | System namespace for Dockyards control plane |
| `--config-map` | `dockyards-system` | ConfigMap name for dockyards config |
| `--log-level` | `info` | Log level (debug, info, warn, error) |
| `--metrics-bind-address` | `0` | Metrics server bind address |
| `--collect-metrics-interval` | `30` | Collect metrics interval in seconds |
| `--enable-webhooks` | `false` | Enable admission webhooks |
| `--allow-origin` | `http://localhost, http://localhost:8000` | CORS allowed origins |
| `--allow-domain` | (none) | Allowed domains for webhook registration |

## Build & Run

```bash
# Build the backend binary
go build -o dockyards-backend ./main.go

# Install with dyctl (bootstrap cluster)
dyctl install --fqdn dockyards.localhost ...

# Seed resources
scripts/create-workload.sh
```

The backend runs two HTTP servers:
- **Public API** on `:9000` — serves the REST API with CORS support
- **Private metrics** on `:9001` — exposes `/metrics` (Prometheus) and `/healthz` endpoints

## Related Repositories

| Repo | Purpose |
|---|---|
| `dockyards-cluster-api` | Cluster API provider integration |
| `dockyards-talos` | Talos Linux integration layer |
| `dockyards-kubevirt` | KubeVirt integration for VM workloads |
| `sudoswedenab/dockyards-backend` | This repository |

## License

Dockyards-backend is licensed under the LICENSE file in this repository.
