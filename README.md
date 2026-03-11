# IronKube

Unified Kubernetes lifecycle management. Bootstrap production-hardened clusters across distributions with a single binary and one declarative config.

## Features

- **Distribution-agnostic** — k3s, kubeadm today; Talos, k0s planned
- **Security at bootstrap** — CIS-hardened profiles applied at creation time
- **No management cluster** — single Go binary, SSH-based
- **Phase-based engine** — validate, connect, bootstrap, kubeconfig fetch
- **Declarative config** — one YAML file describes your entire cluster

## Install

```bash
go install github.com/rohitg00/ironkube/cmd/ironkube@latest
```

## Usage

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
    profile: minimal
  networking:
    cni: flannel
```

Bootstrap:

```bash
ironkube init -c ironkube.yaml
```

Validate without bootstrapping:

```bash
ironkube init -c ironkube.yaml --dry-run
```

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
    podCIDR: 10.42.0.0/16
    serviceCIDR: 10.43.0.0/16
```

## Security Profiles

| Profile | Description |
|---------|-------------|
| `minimal` | RBAC + NodeRestriction, PSA baseline |
| `cis-hardened` | Anonymous auth disabled, audit logging, TLS 1.2+, etcd mutual TLS, kubelet webhook auth, PSA restricted |

## Examples

- [Single node k3s](examples/single-node-k3s.yaml)
- [HA kubeadm with CIS hardening](examples/ha-kubeadm.yaml)

## Development

```bash
make build
make test
make lint
```

## Architecture

```
ironkube init -c ironkube.yaml
  │
  ├─ ValidatePhase    → config + distro + security validation
  ├─ ConnectPhase     → SSH to all nodes (parallel)
  ├─ BootstrapPhase   → install distro, wait for Ready, join nodes
  └─ KubeconfigPhase  → fetch, rewrite server IP, merge locally
```

See [docs/architecture.md](docs/architecture.md) for details.

## Roadmap

- [x] v0.0.0 — Config, engine, SSH, k3s + kubeadm, security profiles, bootstrap pipeline
- [ ] v0.1.0 — Day 1 addons (CNI presets, observability, cert-manager)
- [ ] v0.2.0 — Rolling upgrades with API deprecation pre-flight
- [ ] v0.3.0 — Airgap packaging (OCI artifacts)
- [ ] v0.4.0 — Fleet management (no management cluster)
- [ ] v0.5.0 — GPU/AI workloads + cost estimation

## License

Apache-2.0
