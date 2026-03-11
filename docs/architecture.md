# IronKube Architecture

## Design Principles

1. **No management cluster** — standalone CLI with state in Git/S3
2. **Distribution-agnostic** — plugin interface for k3s, kubeadm, Talos, k0s
3. **Security at bootstrap** — 52% of CIS Benchmark checks applied at creation time
4. **Declarative config** — single YAML file drives everything
5. **Phase-based execution** — ordered pipeline with rollback on failure
6. **Pull-based fleet** — agents connect outbound, no inbound connections

## Package Structure

```
cmd/ironkube/cli/     CLI commands (cobra)
pkg/config/           YAML config parsing, validation, defaults
pkg/engine/           Phase pipeline execution engine
pkg/ssh/              SSH client + parallel executor
pkg/distro/           Distribution plugin interface
pkg/distro/k3s/       K3s bootstrap/join/upgrade
pkg/distro/kubeadm/   Kubeadm bootstrap/join/upgrade
pkg/security/         Security profile presets
pkg/kubeconfig/       Kubeconfig fetch/rewrite/merge
pkg/health/           Cluster health check framework
pkg/phases/           Concrete phase implementations
```

## Plugin Interfaces

### DistroPlugin
Each Kubernetes distribution implements:
- `ServerInstallScript()` — generates bootstrap script for control plane
- `AgentInstallScript()` — generates join script for workers
- `ValidateVersion()` — validates distro-specific version format
- `UpgradeCmd()` — generates upgrade command

### SecurityProfile
Each profile provides flags applied at bootstrap:
- `APIServerFlags()` — kube-apiserver hardening flags
- `EtcdFlags()` — etcd TLS and auth flags
- `KubeletFlags()` — kubelet security flags
- `PSALabels()` — Pod Security Admission namespace labels

## Execution Flow

```
ironkube init --config ironkube.yaml
  │
  ├─ Load config → apply defaults → validate
  ├─ Resolve distro plugin (k3s/kubeadm)
  ├─ Resolve security profile (minimal/cis-hardened)
  ├─ Connect SSH to all nodes
  ├─ Bootstrap first control plane node
  ├─ Join additional control plane nodes (HA)
  ├─ Join worker nodes
  ├─ Fetch and rewrite kubeconfig
  └─ Run health checks
```

## Security Profiles

| Profile | API Server Flags | Etcd Flags | Kubelet Flags | PSA |
|---------|-----------------|------------|---------------|-----|
| minimal | 2 | 0 | 1 | baseline |
| cis-hardened | 11 | 4 | 7 | restricted |

CIS-hardened maps to 52% of CIS Kubernetes Benchmark checks that can be set at bootstrap time (Sections 1, 2, 4: file permissions, API server flags, etcd TLS, kubelet config).

## Roadmap

- v0.0.0 — Foundation (config, engine, SSH, k3s+kubeadm, security, health)
- v0.1.0 — Day 1 addons (CNI presets, observability, cert-manager, Gateway API)
- v0.2.0 — Upgrades (rolling upgrades, API deprecation pre-flight, rollback)
- v0.3.0 — Airgap (OCI packaging, registry bootstrap, proxy DHCP)
- v0.4.0 — Fleet (pull-based agent, Git/S3 state, per-cluster overrides)
- v0.5.0 — GPU/AI (GPU Operator preset, Kueue, cost estimation)
