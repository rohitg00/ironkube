# IronKube v0.0.1 Architecture Redesign

## Current State: What Exists (v0.0.0)

### Codebase: 49 files, ~3,480 LOC, 139 tests

| Package | LOC | What It Does | Verdict |
|---------|-----|-------------|---------|
| `pkg/ssh` | 350 | SSH client + parallel executor | **KEEP** — solid foundation |
| `pkg/engine` | 180 | Phase pipeline + state + cleanup | **KEEP + ENHANCE** — add state persistence, hooks |
| `pkg/config` | 550 | YAML loader + validator + defaults | **REWRITE** — needs provider/fleet/state config |
| `pkg/distro` | 350 | k3s/kubeadm shell script generation | **REWRITE** — becomes BootstrapProvider |
| `pkg/security` | 220 | minimal/CIS-hardened profiles | **KEEP + EXTEND** — add SOC2, HIPAA, PCI-DSS |
| `pkg/kubeconfig` | 260 | Fetch, rewrite, merge kubeconfig | **KEEP** |
| `pkg/health` | 130 | Health check types | **KEEP + EXTEND** — add client-go checks |
| `pkg/phases` | 650 | 7 pipeline phases | **REWRITE** — split into provider-driven phases |
| `pkg/addons` | 120 | Cilium/Calico/cert-manager/monitoring | **REWRITE** — proper Helm integration |
| `cmd/ironkube/cli` | 340 | cobra CLI (init/health/destroy) | **EXTEND** — add 15+ commands |

### Dependencies (go.mod)
- cobra, testify, golang.org/x/crypto, sigs.k8s.io/yaml
- **Missing**: client-go, helm SDK, oras-go, go-git, cloud SDKs

### Fundamental Problems
1. `distro.Plugin` returns **shell script strings** — not extensible, not testable
2. `engine.State` is `map[string]any` — ephemeral, no persistence, no type safety
3. No infrastructure provider concept — assumes SSH to existing machines
4. All K8s interaction via `kubectl` shell commands — no client-go
5. Config model is flat — no fleet, no infra provider, no state backend config
6. Addons are hardcoded `kubectl apply -f URL` — no Helm, no version management
7. No state tracking — "fire and forget", can't detect drift or resume

---

## New Architecture: CAPI-Inspired, No Management Cluster

### Core Idea
Steal CAPI's 3-provider abstraction (Infrastructure, Bootstrap, ControlPlane) but run from a standalone CLI with persistent state. No management cluster required.

```
ironkube apply cluster.yaml
  1. State.Load()                           # Load previous state (or init)
  2. InfraProvider.Reconcile()              # Create/validate machines
  3. BootstrapProvider.GenerateData()       # Distribution-specific install data
  4. Executor.Apply(bootstrapData)          # SSH execute on machines
  5. ControlPlaneProvider.WaitReady()       # client-go: wait for K8s API
  6. AddonProvider.Install()                # Helm SDK: install addons
  7. State.Save()                           # Persist to git/s3/local file
```

### Package Layout

```
ironkube/
  cmd/ironkube/
    cli/                          # cobra commands
      root.go                     # ironkube
      apply.go                    # ironkube apply (replaces init)
      destroy.go                  # ironkube destroy
      status.go                   # ironkube status (cluster health + state)
      upgrade.go                  # ironkube upgrade
      diff.go                     # ironkube diff (detect drift)
      backup.go                   # ironkube backup / restore
      certs.go                    # ironkube certs check / rotate
      etcd.go                     # ironkube etcd status / backup / defrag
      fleet.go                    # ironkube fleet list / apply / diff
      kubeconfig.go               # ironkube kubeconfig export / merge
      version.go
  pkg/
    provider/                     # Provider interfaces (the contracts)
      infra.go                    # InfraProvider interface
      bootstrap.go                # BootstrapProvider interface
      controlplane.go             # ControlPlaneProvider interface
      addon.go                    # AddonProvider interface
      types.go                    # Machine, BootstrapData, ClusterStatus
    providers/                    # Built-in implementations
      infra/
        ssh/                      # Bare metal (validates SSH, no machine creation)
        hetzner/                  # Hetzner Cloud API
        docker/                   # Docker containers (dev/testing)
      bootstrap/
        k3s/                      # k3s install data generation
        kubeadm/                  # kubeadm config generation
        k0s/                      # k0s config generation
        talos/                    # Talos machineconfig generation
      controlplane/
        universal/                # client-go based (works with any distro)
      addon/
        helm/                     # Helm SDK based addon installer
        manifest/                 # Raw manifest installer (kubectl apply)
    config/                       # Cluster configuration
      types.go                    # ClusterConfig with providers, fleet, state
      loader.go                   # YAML loading
      validate.go                 # Validation
      defaults.go                 # Defaults
    state/                        # Persistent state backend
      backend.go                  # StateBackend interface
      types.go                    # ClusterState, MachineState, AddonState
      local/                      # Local file (~/.ironkube/state/)
      git/                        # Git repo state
      s3/                         # S3 bucket state
    engine/                       # Execution engine
      phase.go                    # Phase interface (keep)
      pipeline.go                 # Pipeline (keep + enhance)
      reconciler.go               # Desired → actual state reconciliation
      hooks.go                    # Pre/post phase hooks
    k8s/                          # client-go integration
      client.go                   # K8s client factory from kubeconfig
      nodes.go                    # Node operations (ready, drain, uncordon)
      pods.go                     # Pod operations (wait, status)
      health.go                   # Cluster health via API
    lifecycle/                    # Day 2 operations
      certs/                      # Certificate expiry, rotation
      etcd/                       # Backup, restore, compaction, defrag
      upgrade/                    # Rolling upgrade with rollback
      backup/                     # Cluster backup/restore (etcd + manifests)
    security/                     # Security profiles (keep + extend)
      profile.go                  # Profile interface
      minimal.go                  # Minimal profile
      cis.go                      # CIS benchmarks
      soc2.go                     # SOC2 compliance
      pci.go                      # PCI-DSS compliance
    fleet/                        # Multi-cluster management
      inventory.go                # Cluster inventory
      targeting.go                # Cluster selectors/filters
      operations.go               # Fleet-wide operations
    airgap/                       # Airgap packaging
      bundle.go                   # OCI bundle creation
      registry.go                 # Local registry setup
    cost/                         # Cost estimation
      estimator.go                # Per-cloud pricing
      report.go                   # Cost reports
    ssh/                          # SSH client (keep)
    kubeconfig/                   # Kubeconfig management (keep)
```

---

## Provider Interfaces

### InfraProvider
```go
type InfraProvider interface {
    Name() string
    Validate(ctx context.Context, machines []MachineSpec) error
    Provision(ctx context.Context, machines []MachineSpec) ([]Machine, error)
    Status(ctx context.Context, machines []Machine) ([]MachineStatus, error)
    Destroy(ctx context.Context, machines []Machine) error
}
```
- `ssh` provider: Validate = test SSH connectivity, Provision = no-op (machines exist), Destroy = no-op
- `hetzner` provider: Validate = API key, Provision = create servers, Destroy = delete servers
- `docker` provider: Validate = docker running, Provision = create containers, Destroy = remove containers

### BootstrapProvider
```go
type BootstrapData struct {
    Scripts    []Script          // Shell commands to execute
    Files      []FileToWrite     // Files to place on machine
    EnvVars    map[string]string // Environment variables
    WaitCheck  WaitCondition     // How to know it succeeded
}

type BootstrapProvider interface {
    Name() string
    ValidateVersion(version string) error
    SupportedVersions() []string
    InitControlPlane(ctx context.Context, cluster *ClusterSpec, machine Machine, token string) (*BootstrapData, error)
    JoinControlPlane(ctx context.Context, cluster *ClusterSpec, machine Machine, joinEndpoint, token string) (*BootstrapData, error)
    JoinWorker(ctx context.Context, cluster *ClusterSpec, machine Machine, joinEndpoint, token string) (*BootstrapData, error)
    PreUpgrade(ctx context.Context, machine Machine, fromVersion, toVersion string) (*BootstrapData, error)
    PostUpgrade(ctx context.Context, machine Machine, version string) (*BootstrapData, error)
    Uninstall(ctx context.Context, machine Machine, role string) (*BootstrapData, error)
    SecurityArgs(profile SecurityProfile) []string
    KubeconfigPath() string
    TokenCommand() string
}
```

### ControlPlaneProvider
```go
type ControlPlaneProvider interface {
    NewClient(kubeconfig []byte) (kubernetes.Interface, error)
    WaitForAPI(ctx context.Context, client kubernetes.Interface, timeout time.Duration) error
    NodeReady(ctx context.Context, client kubernetes.Interface, nodeName string) (bool, error)
    WaitForNodes(ctx context.Context, client kubernetes.Interface, expected int, timeout time.Duration) error
    DrainNode(ctx context.Context, client kubernetes.Interface, nodeName string, opts DrainOptions) error
    UncordonNode(ctx context.Context, client kubernetes.Interface, nodeName string) error
    ClusterHealth(ctx context.Context, client kubernetes.Interface) (*HealthReport, error)
    ListNodes(ctx context.Context, client kubernetes.Interface) ([]NodeInfo, error)
    CertExpiry(ctx context.Context, exec Executor, machine Machine) ([]CertInfo, error)
    EtcdHealth(ctx context.Context, exec Executor, machine Machine) (*EtcdHealth, error)
    EtcdBackup(ctx context.Context, exec Executor, machine Machine, destPath string) error
    EtcdRestore(ctx context.Context, exec Executor, machine Machine, snapshotPath string) error
}
```

### AddonProvider
```go
type AddonSpec struct {
    Name       string
    Type       string            // "helm" | "manifest"
    Repository string            // Helm repo URL or manifest URL
    Chart      string            // Helm chart name
    Version    string            // Chart/manifest version
    Namespace  string
    Values     map[string]any    // Helm values
    WaitReady  bool
}

type AddonProvider interface {
    Install(ctx context.Context, client kubernetes.Interface, kubeconfig []byte, addon AddonSpec) error
    Upgrade(ctx context.Context, client kubernetes.Interface, kubeconfig []byte, addon AddonSpec) error
    Uninstall(ctx context.Context, client kubernetes.Interface, kubeconfig []byte, addon AddonSpec) error
    Status(ctx context.Context, client kubernetes.Interface, addon AddonSpec) (*AddonStatus, error)
}
```

---

## State Backend

### State Model
```go
type ClusterState struct {
    Metadata      StateMetadata
    Spec          ClusterSpec              // Desired state (from config)
    Status        ClusterStatus            // Actual state
    Machines      map[string]MachineState  // Per-machine state
    Addons        map[string]AddonState    // Per-addon state
    Certificates  []CertState              // Cert expiry tracking
    LastOperation OperationRecord          // Last apply/upgrade/destroy
    History       []OperationRecord        // Operation history
}

type MachineState struct {
    ID           string
    ProviderID   string          // Cloud provider machine ID
    Host         string
    Role         string          // "control-plane" | "worker"
    K8sVersion   string
    Status       string          // "provisioning" | "ready" | "upgrading" | "draining" | "deleted"
    JoinedAt     time.Time
    LastHealthAt time.Time
}
```

### StateBackend Interface
```go
type StateBackend interface {
    Load(clusterName string) (*ClusterState, error)
    Save(state *ClusterState) error
    Lock(clusterName string) (UnlockFunc, error)  // Prevent concurrent operations
    List() ([]string, error)                        // List all clusters
    Delete(clusterName string) error
}
```

---

## Config Schema (v2)

```yaml
apiVersion: ironkube.io/v1alpha2
kind: Cluster
metadata:
  name: production
  labels:
    env: production
    region: eu-west-1
spec:
  distro: k3s
  version: v1.34.3+k3s1

  infrastructure:
    provider: ssh                # ssh | hetzner | docker | aws
    ssh:                         # provider-specific config
      keyPath: ~/.ssh/id_ed25519

  controlPlane:
    replicas: 3
    nodes:
      - host: 10.0.0.1
      - host: 10.0.0.2
      - host: 10.0.0.3
    resources:                   # NEW: resource constraints
      cpu: 4
      memory: 8Gi

  workers:
    - name: general
      nodes:
        - host: 10.0.0.10
        - host: 10.0.0.11
    - name: gpu                  # NEW: GPU node pools
      labels:
        accelerator: nvidia-a100
      taints:
        - key: nvidia.com/gpu
          effect: NoSchedule
      nodes:
        - host: 10.0.0.20

  networking:
    podCIDR: 10.42.0.0/16
    serviceCIDR: 10.43.0.0/16

  security:
    profile: cis-hardened        # minimal | cis-hardened | soc2 | pci-dss
    encryptionAtRest: true       # NEW: etcd encryption
    auditLogging: true           # NEW: API audit log

  addons:
    cni:
      name: cilium
      version: v1.17.4
      values:                    # NEW: Helm values
        hubble:
          enabled: true
    certManager:
      enabled: true
      version: v1.17.2
    monitoring:
      enabled: true
      values:
        retention: 30d

  lifecycle:                     # NEW: Day 2 config
    certs:
      autoRotate: true
      rotateBeforeExpiry: 30d
    etcd:
      backupSchedule: "0 */6 * * *"
      backupRetention: 7
      compactionInterval: 1h
    upgrades:
      strategy: rolling          # rolling | blue-green
      maxUnavailable: 1

  state:                         # NEW: state backend
    backend: local               # local | git | s3
    # git:
    #   repo: git@github.com:org/infra-state.git
    #   branch: main
    # s3:
    #   bucket: ironkube-state
    #   region: eu-west-1
```

### Fleet Config (separate file)
```yaml
apiVersion: ironkube.io/v1alpha2
kind: Fleet
metadata:
  name: my-fleet
spec:
  clusters:
    - path: clusters/production.yaml
    - path: clusters/staging.yaml
    - path: clusters/dev.yaml
  state:
    backend: git
    git:
      repo: git@github.com:org/fleet-state.git
  policies:
    upgradeWindow: "Sat 02:00-06:00 UTC"
    maxConcurrentUpgrades: 1
```

---

## CLI Commands (all v0.0.1)

### Cluster Lifecycle
```
ironkube apply -f cluster.yaml                             # Create or update cluster
ironkube destroy -f cluster.yaml                           # Tear down + cleanup
ironkube status -f cluster.yaml                            # Health + state + certs + etcd
ironkube diff -f cluster.yaml                              # Drift detection
ironkube kubeconfig export -f cluster.yaml                 # Export/merge kubeconfig
```

### Upgrades
```
ironkube upgrade -f cluster.yaml --version v1.35.0+k3s1   # Rolling upgrade with rollback
```

### Certificates
```
ironkube certs check -f cluster.yaml                       # Expiry report
ironkube certs rotate -f cluster.yaml                      # Rotate expiring certs
```

### etcd Operations
```
ironkube etcd status -f cluster.yaml                       # Health/size/members
ironkube etcd backup -f cluster.yaml -o backup.db          # Snapshot
ironkube etcd restore -f cluster.yaml -i backup.db         # Restore
```

### Backup & Restore
```
ironkube backup create -f cluster.yaml                     # Full cluster backup
ironkube backup restore -f cluster.yaml --from latest      # Restore
```

### Fleet Management
```
ironkube fleet apply -f fleet.yaml                         # Apply to all clusters
ironkube fleet status -f fleet.yaml                        # Fleet-wide health
ironkube fleet diff -f fleet.yaml                          # Drift across fleet
ironkube fleet upgrade -f fleet.yaml --version ...         # Rolling fleet upgrade
```

### Airgap
```
ironkube bundle create -f cluster.yaml -o bundle.tar       # Create airgap bundle
ironkube bundle apply -f cluster.yaml --bundle bundle.tar  # Bootstrap from bundle
```

---

## What Gets Reused vs Rewritten

### KEEP (move into new structure)
- `pkg/ssh/client.go` + `executor.go` → `pkg/ssh/` (as-is)
- `pkg/kubeconfig/kubeconfig.go` → `pkg/kubeconfig/` (as-is)
- `pkg/health/checker.go` → `pkg/k8s/health.go` (types reused)
- `pkg/engine/phase.go` → `pkg/engine/phase.go` (interface unchanged)
- `pkg/engine/pipeline.go` → `pkg/engine/pipeline.go` (add hooks)
- `pkg/engine/state.go` → `pkg/engine/state.go` (add persistence)
- `pkg/security/profile.go` + profiles → `pkg/security/` (extend)

### REWRITE
- `pkg/distro/plugin.go` → `pkg/provider/bootstrap.go` (new interface)
- `pkg/distro/k3s/` → `pkg/providers/bootstrap/k3s/` (returns BootstrapData, not strings)
- `pkg/distro/kubeadm/` → `pkg/providers/bootstrap/kubeadm/` (returns BootstrapData)
- `pkg/config/types.go` → new schema with providers, lifecycle, state, fleet
- `pkg/phases/bootstrap.go` → provider-driven reconciliation
- `pkg/addons/` → `pkg/providers/addon/helm/` (Helm SDK)

### NEW
- `pkg/provider/` → all 4 provider interfaces
- `pkg/providers/infra/ssh/` → bare metal infra provider
- `pkg/providers/infra/hetzner/` → Hetzner Cloud provider
- `pkg/providers/controlplane/universal/` → client-go based
- `pkg/state/` → persistent state backend (local/git/s3)
- `pkg/k8s/` → client-go integration (nodes, pods, health)
- `pkg/lifecycle/` → certs, etcd, upgrade, backup
- `pkg/fleet/` → multi-cluster management
- `pkg/airgap/` → OCI bundle packaging
- `pkg/cost/` → cost estimation

---

## v0.0.1 Scope — Everything Together

**Goal**: Ship a unified K8s lifecycle tool that does Day 0 through Day 2, single and multi-cluster, with airgap support. Nothing like this exists.

### Architecture Foundation
1. Provider interfaces (Infra, Bootstrap, ControlPlane, Addon)
2. Persistent state backend (local/git/S3) with locking
3. New config schema (v1alpha2 with providers, lifecycle, fleet, state)
4. client-go for all K8s interaction
5. Helm SDK for addon management
6. Engine enhancements (hooks, reconciliation)

### Infrastructure Providers
7. SSH bare metal provider (validate + execute, no machine creation)
8. Hetzner Cloud provider (create/destroy VMs via API)
9. Docker provider (for dev/testing)

### Bootstrap Providers
10. k3s (rewrite: BootstrapData structs, not shell strings)
11. kubeadm (rewrite: same)
12. k0s
13. Talos (machineconfig generation)

### Day 2 Lifecycle
14. Certificate expiry detection + auto-rotation
15. etcd health monitoring, backup, restore, compaction, defrag
16. Rolling upgrades with drain/uncordon/rollback
17. Pre-upgrade API deprecation scanning
18. Full backup/restore (etcd snapshot + manifest export)

### Fleet Management
19. Fleet config (multi-cluster targeting)
20. Fleet operations (apply, diff, upgrade across clusters)
21. Drift detection + reporting

### Security & Compliance
22. Security profiles: minimal, CIS-hardened, SOC2, PCI-DSS
23. Encryption at rest setup
24. Audit logging configuration

### Addons (Helm-based)
25. CNI (Cilium, Calico) with version + values
26. cert-manager
27. Monitoring (kube-prometheus stack)
28. Cost estimation per cloud provider

### Airgap
29. OCI bundle creation (binaries + images + charts)
30. Bootstrap from bundle
31. Local registry setup

### CLI Commands
32. apply, destroy, status, diff, kubeconfig
33. upgrade
34. certs check/rotate
35. etcd status/backup/restore
36. backup create/restore
37. fleet apply/status/diff/upgrade
38. bundle create/apply

### New Dependencies
- k8s.io/client-go (K8s API)
- helm.sh/helm/v3 (Helm SDK)
- go.etcd.io/etcd/client/v3 (etcd client)
- github.com/hetznercloud/hcloud-go/v2 (Hetzner)
- github.com/docker/docker (Docker provider)
- github.com/go-git/go-git/v5 (git state backend)
- github.com/aws/aws-sdk-go-v2 (S3 state backend)
- oras.land/oras-go/v2 (OCI artifacts for airgap)
- filippo.io/age (encryption for airgap bundles)

### Test Target
- 500+ tests
- e2e: Lima VM (k3s, kubeadm), Docker provider
- CI: cross-platform builds (linux/darwin x amd64/arm64)

---

## Key Architectural Decisions

1. **No management cluster** — CLI-driven, state in git/s3/local file
2. **Provider model** — CAPI contracts without CAPI complexity
3. **client-go for all K8s interaction** — no more shelling out to kubectl
4. **Helm SDK for addons** — version management, values, rollback
5. **Persistent state** — enables diff, drift detection, resume, fleet ops
6. **BootstrapData struct** — scripts + files + env, not raw shell strings
7. **Go single binary** — no runtime deps, cross-platform
8. **Locking** — state backend locking prevents concurrent operations on same cluster

---

## Success Criteria (v0.0.1 ships when ALL pass)

### Core
- `ironkube apply` bootstraps k3s/kubeadm/k0s/Talos cluster with client-go verification
- `ironkube status` shows health, cert expiry, etcd status, node info via K8s API
- `ironkube diff` detects version drift, missing nodes, addon changes
- `ironkube destroy` tears down cleanly with state cleanup + cloud resource cleanup
- State persists across sessions (local/git/S3)

### Day 2
- `ironkube upgrade` does rolling K8s version upgrade with rollback on failure
- `ironkube certs check/rotate` detects and rotates expiring certificates
- `ironkube etcd status/backup/restore` does backup, restore, defrag
- `ironkube backup create/restore` does full cluster backup + restore

### Fleet
- `ironkube fleet apply/status/diff/upgrade` manages 10+ clusters from one config
- Drift detection across fleet works

### Infrastructure
- SSH provider works for bare metal
- Hetzner provider creates VMs from scratch
- Docker provider works for dev/testing

### Airgap
- `ironkube bundle create` packages binaries + images + charts into OCI bundle
- `ironkube bundle apply` bootstraps cluster from bundle without internet

### Quality
- 500+ tests, CI green, cross-platform builds (linux/darwin x amd64/arm64)
- e2e tested on Lima VM (k3s + kubeadm)
