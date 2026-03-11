# IronKube

Unified Kubernetes lifecycle management. Bootstrap production-hardened clusters across distributions with a single declarative config.

## Why IronKube

Every K8s tool solves 1-3 problems. k3sup is locked to k3s. k0sctl is locked to k0s. kubeadm has no fleet management. CAPI needs a management cluster. Nobody does distribution-agnostic bootstrapping with security built in from day zero.

IronKube fixes this:
- **One config, any distro** — k3s, kubeadm (Talos, k0s coming)
- **Security at bootstrap** — CIS-hardened profiles applied at creation, not bolted on after
- **No management cluster** — standalone Go binary, state in Git/S3
- **Full lifecycle** — bootstrap, upgrade, health checks, fleet (planned)

## Quick Start

```bash
go install github.com/rohitg00/ironkube/cmd/ironkube@latest
```

Create `ironkube.yaml`:

```yaml
apiVersion: ironkube.io/v1alpha1
kind: Cluster
metadata:
  name: my-cluster
spec:
  distro: k3s
  version: v1.34.3+k3s1
  controlPlane:
    replicas: 1
    nodes:
      - host: 10.0.0.1
        user: root
  security:
    profile: cis-hardened
  networking:
    cni: cilium
    podCIDR: 10.42.0.0/16
    serviceCIDR: 10.43.0.0/16
```

Bootstrap:

```bash
ironkube init --config ironkube.yaml

ironkube init --config ironkube.yaml --dry-run
```

## Supported Distributions

| Distro | Bootstrap | Join | HA | Upgrade |
|--------|-----------|------|----|---------|
| k3s | Yes | Yes | Yes (embedded etcd) | Planned |
| kubeadm | Yes | Yes | Yes (stacked etcd) | Planned |
| Talos | Planned | | | |
| k0s | Planned | | | |

## Security Profiles

| Profile | Description | CIS Checks |
|---------|-------------|-----------|
| `minimal` | Basic RBAC + NodeRestriction, PSA baseline | ~10% |
| `cis-hardened` | Full CIS Benchmark hardening at bootstrap | ~52% |

CIS-hardened applies: anonymous auth disabled, RBAC+Node authorization, audit logging, TLS 1.2 minimum, etcd mutual TLS, kubelet webhook auth, kernel defaults protection, Pod Security restricted, and more.

## Config Reference

```yaml
apiVersion: ironkube.io/v1alpha1
kind: Cluster
metadata:
  name: string                    # cluster name (required)
spec:
  distro: k3s | kubeadm          # distribution (required)
  version: string                 # K8s version (required)
  controlPlane:
    replicas: 1 | 3 | 5          # must be odd for HA
    nodes:
      - host: string             # IP or hostname (required)
        user: root               # SSH user (default: root)
        port: 22                 # SSH port (default: 22)
        keyPath: ~/.ssh/id_rsa   # SSH key (default: ~/.ssh/id_rsa)
  workers:
    - name: string               # pool name
      nodes:
        - host: string
          user: root
  security:
    profile: minimal | cis-hardened  # (default: minimal)
  networking:
    cni: flannel | cilium | calico   # (default: flannel)
    podCIDR: 10.42.0.0/16           # (default: 10.42.0.0/16)
    serviceCIDR: 10.43.0.0/16       # (default: 10.43.0.0/16)
```

## Examples

- [Single node k3s](examples/single-node-k3s.yaml) — simplest setup
- [HA kubeadm with CIS hardening](examples/ha-kubeadm.yaml) — production-grade

## Architecture

See [docs/architecture.md](docs/architecture.md) for full design.

```
ironkube init --config ironkube.yaml
  │
  ├─ Load config → defaults → validate
  ├─ Resolve distro plugin + security profile
  ├─ SSH connect to all nodes
  ├─ Bootstrap control plane
  ├─ Join workers
  ├─ Fetch kubeconfig
  └─ Health checks
```

## Development

```bash
make build        # build binary
make test         # run all tests (119 tests)
make lint         # golangci-lint
make clean        # remove build artifacts
```

## Roadmap

- [x] **v0.0.0** — Config, engine, SSH, k3s+kubeadm, security profiles, health checks
- [ ] **v0.1.0** — Day 1 addons (CNI presets, observability, cert-manager, Gateway API)
- [ ] **v0.2.0** — Rolling upgrades with API deprecation pre-flight
- [ ] **v0.3.0** — Airgap packaging (OCI artifacts, registry bootstrap)
- [ ] **v0.4.0** — Fleet management (pull-based agent, no management cluster)
- [ ] **v0.5.0** — GPU/AI workloads + cost estimation

## License

Apache-2.0
