package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rohitg00/ironkube/cmd/ironkube/cli"
	"github.com/rohitg00/ironkube/pkg/airgap"
	"github.com/rohitg00/ironkube/pkg/config/v1alpha2"
	"github.com/rohitg00/ironkube/pkg/engine"
	"github.com/rohitg00/ironkube/pkg/fleet"
	"github.com/rohitg00/ironkube/pkg/provider"
	"github.com/rohitg00/ironkube/pkg/providers/addon/helm"
	"github.com/rohitg00/ironkube/pkg/providers/addon/manifest"
	"github.com/rohitg00/ironkube/pkg/providers/bootstrap/k0s"
	"github.com/rohitg00/ironkube/pkg/providers/bootstrap/k3s"
	"github.com/rohitg00/ironkube/pkg/providers/bootstrap/kubeadm"
	"github.com/rohitg00/ironkube/pkg/providers/bootstrap/talos"
	"github.com/rohitg00/ironkube/pkg/providers/infra/docker"
	"github.com/rohitg00/ironkube/pkg/providers/infra/hetzner"
	"github.com/rohitg00/ironkube/pkg/providers/infra/ssh"
	"github.com/rohitg00/ironkube/pkg/providers/controlplane/universal"
	"github.com/rohitg00/ironkube/pkg/security"
	"github.com/rohitg00/ironkube/pkg/state"
	"github.com/rohitg00/ironkube/pkg/state/local"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFullApplyWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "cluster.yaml")

	configYAML := `apiVersion: ironkube.dev/v1alpha2
kind: Cluster
metadata:
  name: integration-test
spec:
  distro: k3s
  version: v1.31.2+k3s1
  infrastructure:
    provider: ssh
  controlPlane:
    replicas: 1
    nodes:
      - host: 10.0.0.1
        user: root
        port: 22
  workers:
    - name: pool-1
      nodes:
        - host: 10.0.0.2
          user: root
          port: 22
  networking:
    podCIDR: 10.42.0.0/16
    serviceCIDR: 10.43.0.0/16
  security:
    profile: minimal
  addons:
    - name: metrics-server
      enabled: true
      version: "3.11.0"
      namespace: kube-system
`
	require.NoError(t, os.WriteFile(configPath, []byte(configYAML), 0644))

	cfg, err := v1alpha2.Load(configPath)
	require.NoError(t, err)
	assert.Equal(t, "integration-test", cfg.Metadata.Name)
	assert.Equal(t, "k3s", cfg.Spec.Distro)

	stateDir := filepath.Join(tmpDir, "state")
	backend := local.New(stateDir)

	desired := configToDesired(cfg)
	assert.Len(t, desired.Machines, 2)
	assert.Len(t, desired.Addons, 1)
	assert.Equal(t, "v1.31.2+k3s1", desired.Version)

	actual := engine.ActualState{
		Machines: make(map[string]state.MachineState),
		Addons:   make(map[string]state.AddonState),
	}

	result := engine.Reconcile(desired, actual)
	assert.False(t, result.InSync)

	hasCreateMachine := false
	hasInstallAddon := false
	for _, action := range result.Actions {
		if action.Type == engine.ActionCreateMachine {
			hasCreateMachine = true
		}
		if action.Type == engine.ActionInstallAddon {
			hasInstallAddon = true
		}
	}
	assert.True(t, hasCreateMachine)
	assert.True(t, hasInstallAddon)

	clusterState := &state.ClusterState{
		Metadata: state.StateMetadata{
			Name:      cfg.Metadata.Name,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Version:   cfg.Spec.Version,
		},
	}
	for _, m := range desired.Machines {
		clusterState.SetMachine(m.Host, state.MachineState{
			ID:         m.Host,
			Host:       m.Host,
			Role:       m.Role,
			K8sVersion: cfg.Spec.Version,
			Status:     "ready",
			JoinedAt:   time.Now(),
		})
	}
	for _, a := range desired.Addons {
		clusterState.SetAddon(a.Name, state.AddonState{
			Name:      a.Name,
			Version:   a.Version,
			Installed: true,
			Ready:     true,
		})
	}

	require.NoError(t, backend.Save(clusterState))

	loaded, err := backend.Load("integration-test")
	require.NoError(t, err)

	actualFromState := engine.ActualState{
		Machines: loaded.Machines,
		Addons:   loaded.Addons,
		Version:  loaded.Metadata.Version,
	}
	result2 := engine.Reconcile(desired, actualFromState)
	assert.True(t, result2.InSync)
}

func TestProviderRegistration(t *testing.T) {
	k3sProv := k3s.New()
	assert.Equal(t, "k3s", k3sProv.Name())

	kubeadmProv := kubeadm.New()
	assert.Equal(t, "kubeadm", kubeadmProv.Name())

	k0sProv := k0s.New()
	assert.Equal(t, "k0s", k0sProv.Name())

	talosProv := talos.New()
	assert.Equal(t, "talos", talosProv.Name())

	sshProv := ssh.New()
	assert.Equal(t, "ssh", sshProv.Name())

	hetznerProv := hetzner.New(hetzner.Config{
		Location:   "fsn1",
		ServerType: "cx31",
		Image:      "ubuntu-22.04",
		SSHKeyName: "test",
	}, &mockHetznerClient{})
	assert.Equal(t, "hetzner", hetznerProv.Name())

	dockerProv := docker.New(docker.Config{
		Image:   "kindest/node:v1.31.0",
		Network: "test-net",
	}, &mockDockerClient{})
	assert.Equal(t, "docker", dockerProv.Name())

	cpProv := universal.New("test-cluster")
	assert.NotNil(t, cpProv)

	helmProv := helm.New(&mockHelmClient{})
	assert.NotNil(t, helmProv)

	manifestProv := manifest.New(&mockManifestApplier{}, &mockManifestFetcher{})
	assert.NotNil(t, manifestProv)

	var _ provider.BootstrapProvider = k3sProv
	var _ provider.BootstrapProvider = kubeadmProv
	var _ provider.BootstrapProvider = k0sProv
	var _ provider.BootstrapProvider = talosProv
	var _ provider.InfraProvider = sshProv
	var _ provider.InfraProvider = hetznerProv
	var _ provider.InfraProvider = dockerProv
}

func TestSecurityProfileWorkflow(t *testing.T) {
	cisProfile, err := security.Get("cis-hardened")
	require.NoError(t, err)
	assert.Equal(t, "cis-hardened", cisProfile.Name())

	k3sProv := k3s.New()
	args := k3sProv.SecurityArgs(&cisSecurityAdapter{cisProfile})
	assert.NotEmpty(t, args)

	hasProtectKernel := false
	for _, arg := range args {
		if arg == "--protect-kernel-defaults" {
			hasProtectKernel = true
		}
	}
	assert.True(t, hasProtectKernel)

	minProfile, err := security.Get("minimal")
	require.NoError(t, err)
	assert.Equal(t, "minimal", minProfile.Name())
	assert.NotEmpty(t, minProfile.KubeletFlags())
}

func TestConfigToStateRoundtrip(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "cluster.yaml")

	configYAML := `apiVersion: ironkube.dev/v1alpha2
kind: Cluster
metadata:
  name: roundtrip-test
spec:
  distro: kubeadm
  version: v1.31.2
  infrastructure:
    provider: ssh
  controlPlane:
    replicas: 3
    nodes:
      - host: 10.0.0.1
      - host: 10.0.0.2
      - host: 10.0.0.3
  workers:
    - name: pool-1
      nodes:
        - host: 10.0.1.1
        - host: 10.0.1.2
  networking:
    podCIDR: 10.244.0.0/16
    serviceCIDR: 10.96.0.0/12
  security:
    profile: minimal
  addons:
    - name: flannel
      enabled: true
      version: "0.22.0"
      namespace: kube-flannel
    - name: metrics-server
      enabled: true
      version: "3.11.0"
      namespace: kube-system
    - name: disabled-addon
      enabled: false
      version: "1.0.0"
      namespace: default
`
	require.NoError(t, os.WriteFile(configPath, []byte(configYAML), 0644))

	cfg, err := v1alpha2.Load(configPath)
	require.NoError(t, err)

	desired := configToDesired(cfg)
	assert.Len(t, desired.Machines, 5)
	assert.Len(t, desired.Addons, 2)

	cpCount := 0
	workerCount := 0
	for _, m := range desired.Machines {
		if m.Role == "control-plane" {
			cpCount++
		}
		if m.Role == "worker" {
			workerCount++
		}
	}
	assert.Equal(t, 3, cpCount)
	assert.Equal(t, 2, workerCount)

	actual := engine.ActualState{
		Machines: make(map[string]state.MachineState),
		Addons:   make(map[string]state.AddonState),
	}
	result := engine.Reconcile(desired, actual)
	assert.False(t, result.InSync)
	assert.GreaterOrEqual(t, len(result.Actions), 5+2)

	for _, m := range desired.Machines {
		actual.Machines[m.Host] = state.MachineState{
			ID:         m.Host,
			Host:       m.Host,
			Role:       m.Role,
			K8sVersion: cfg.Spec.Version,
			Status:     "ready",
		}
	}
	for _, a := range desired.Addons {
		actual.Addons[a.Name] = state.AddonState{
			Name:      a.Name,
			Version:   a.Version,
			Installed: true,
			Ready:     true,
		}
	}
	actual.Version = cfg.Spec.Version

	result2 := engine.Reconcile(desired, actual)
	assert.True(t, result2.InSync)
}

func TestFleetWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "state")
	backend := local.New(stateDir)

	cluster1State := &state.ClusterState{
		Metadata: state.StateMetadata{
			Name:      "cluster-a",
			CreatedAt: time.Now(),
			Version:   "v1.31.0",
		},
		Machines: map[string]state.MachineState{
			"10.0.0.1": {
				ID:         "10.0.0.1",
				Host:       "10.0.0.1",
				Role:       "control-plane",
				K8sVersion: "v1.31.0",
				Status:     "ready",
			},
		},
	}
	require.NoError(t, backend.Save(cluster1State))

	fleetConfigPath := filepath.Join(tmpDir, "fleet.yaml")
	fleetYAML := `apiVersion: ironkube.dev/v1alpha2
kind: Fleet
metadata:
  name: test-fleet
spec:
  clusters:
    - path: cluster-a.yaml
      labels:
        env: staging
    - path: cluster-b.yaml
      labels:
        env: production
`
	require.NoError(t, os.WriteFile(fleetConfigPath, []byte(fleetYAML), 0644))

	fleetCfg, err := fleet.LoadFleetConfig(fleetConfigPath)
	require.NoError(t, err)
	assert.Equal(t, "test-fleet", fleetCfg.Metadata.Name)
	assert.Len(t, fleetCfg.Spec.Clusters, 2)

	inv := fleet.NewInventory(backend)
	summaries, err := inv.Collect(context.Background(), fleetCfg)
	require.NoError(t, err)
	assert.Len(t, summaries, 2)

	var clusterA, clusterB fleet.ClusterSummary
	for _, s := range summaries {
		if s.Name == "cluster-a" {
			clusterA = s
		}
		if s.Name == "cluster-b" {
			clusterB = s
		}
	}

	assert.True(t, clusterA.Available)
	assert.NotNil(t, clusterA.State)
	assert.False(t, clusterB.Available)

	staging := fleet.FilterByLabel(summaries, "env", "staging")
	assert.Len(t, staging, 1)
	assert.Equal(t, "cluster-a", staging[0].Name)

	summary := fleet.Summary(summaries)
	assert.Equal(t, 2, summary.TotalClusters)
	assert.Equal(t, 1, summary.AvailableClusters)
	assert.Equal(t, 1, summary.TotalMachines)
	assert.Equal(t, 1, summary.ReadyMachines)
}

func TestBundleCreateAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	bundleDir := filepath.Join(tmpDir, "bundle")

	binFetcher := &mockBinaryFetcher{}
	imgExporter := &mockImageExporter{}
	chartFetcher := &mockChartFetcher{}

	creator := airgap.NewCreator(binFetcher, imgExporter, chartFetcher)

	spec := airgap.BundleSpec{
		Name:    "test-bundle",
		Version: "v1.31.0",
		Distro:  "k3s",
		Arch:    "amd64",
		Binaries: []airgap.BinaryRef{
			{Name: "k3s", Version: "v1.31.0+k3s1"},
			{Name: "kubectl", Version: "v1.31.0"},
		},
		Images: []string{"rancher/k3s:v1.31.0"},
		Charts: []airgap.ChartRef{
			{Repository: "https://charts.example.com", Chart: "metrics-server", Version: "3.11.0"},
		},
	}

	manifest, err := creator.Create(context.Background(), spec, bundleDir)
	require.NoError(t, err)
	assert.Equal(t, "test-bundle", manifest.Name)
	assert.Equal(t, "v1.31.0", manifest.Version)
	assert.Equal(t, "k3s", manifest.Distro)
	assert.Len(t, manifest.Binaries, 2)
	assert.Len(t, manifest.Images, 1)
	assert.Len(t, manifest.Charts, 1)

	loader := airgap.NewLoader(nil, nil)
	loadedManifest, err := loader.LoadManifest(bundleDir)
	require.NoError(t, err)
	assert.Equal(t, manifest.Name, loadedManifest.Name)
	assert.Equal(t, manifest.Version, loadedManifest.Version)
	assert.Equal(t, manifest.Distro, loadedManifest.Distro)
	assert.Equal(t, manifest.Binaries, loadedManifest.Binaries)
	assert.Equal(t, manifest.Charts, loadedManifest.Charts)
}

func TestReconcilerWithUpgrade(t *testing.T) {
	desired := engine.DesiredState{
		Version: "v1.29.0",
		Machines: []provider.MachineSpec{
			{Host: "10.0.0.1", Role: "control-plane", User: "root", Port: 22},
			{Host: "10.0.0.2", Role: "control-plane", User: "root", Port: 22},
			{Host: "10.0.0.3", Role: "control-plane", User: "root", Port: 22},
		},
	}

	actual := engine.ActualState{
		Version: "v1.28.0",
		Machines: map[string]state.MachineState{
			"10.0.0.1": {Host: "10.0.0.1", Role: "control-plane", K8sVersion: "v1.28.0", Status: "ready"},
			"10.0.0.2": {Host: "10.0.0.2", Role: "control-plane", K8sVersion: "v1.28.0", Status: "ready"},
			"10.0.0.3": {Host: "10.0.0.3", Role: "control-plane", K8sVersion: "v1.28.0", Status: "ready"},
		},
		Addons: make(map[string]state.AddonState),
	}

	result := engine.Reconcile(desired, actual)
	assert.False(t, result.InSync)

	upgradeCount := 0
	for _, action := range result.Actions {
		if action.Type == engine.ActionUpgradeVersion {
			upgradeCount++
			assert.Equal(t, "v1.28.0", action.FromVersion)
			assert.Equal(t, "v1.29.0", action.ToVersion)
		}
	}
	assert.Equal(t, 3, upgradeCount)
}

func TestCLICommandsRegistered(t *testing.T) {
	root := cli.NewRootCmd()
	assert.Equal(t, "ironkube", root.Use)

	expectedCommands := []string{
		"apply",
		"destroy",
		"status",
		"diff",
		"certs",
		"etcd",
		"upgrade",
		"backup",
		"fleet",
		"airgap",
		"security",
		"config",
		"state",
		"version",
		"init",
		"health",
	}

	registeredNames := make(map[string]bool)
	for _, cmd := range root.Commands() {
		registeredNames[cmd.Name()] = true
	}

	for _, expected := range expectedCommands {
		assert.True(t, registeredNames[expected], "command %q not registered on root", expected)
	}
}

func TestBootstrapProviderInitProducesBootstrapData(t *testing.T) {
	providers := []provider.BootstrapProvider{
		k3s.New(),
		kubeadm.New(),
		k0s.New(),
		talos.New(),
	}

	cluster := &provider.ClusterSpec{
		Name:        "test-cluster",
		Distro:      "k3s",
		Version:     "v1.31.0",
		PodCIDR:     "10.42.0.0/16",
		ServiceCIDR: "10.43.0.0/16",
		HAEnabled:   false,
	}

	machine := provider.Machine{
		ID: "test-0",
		Spec: provider.MachineSpec{
			Host: "10.0.0.1",
			User: "root",
			Port: 22,
		},
		Status: provider.MachineStatusReady,
	}

	for _, prov := range providers {
		t.Run(prov.Name(), func(t *testing.T) {
			data, err := prov.InitControlPlane(context.Background(), cluster, machine, "test-token")
			require.NoError(t, err)
			require.NotNil(t, data)
			assert.NotEmpty(t, data.Scripts)

			for _, script := range data.Scripts {
				assert.NotEmpty(t, script.Name)
				assert.NotEmpty(t, script.Content)
			}
		})
	}
}

func TestStateLockingWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	backend := local.New(tmpDir)

	clusterState := &state.ClusterState{
		Metadata: state.StateMetadata{
			Name:      "lock-test",
			CreatedAt: time.Now(),
		},
	}
	require.NoError(t, backend.Save(clusterState))

	unlock, err := backend.Lock("lock-test")
	require.NoError(t, err)
	require.NotNil(t, unlock)

	_, err = backend.Lock("lock-test")
	assert.Error(t, err)

	unlock()

	unlock2, err := backend.Lock("lock-test")
	require.NoError(t, err)
	defer unlock2()

	loaded, err := backend.Load("lock-test")
	require.NoError(t, err)
	assert.Equal(t, "lock-test", loaded.Metadata.Name)

	clusters, err := backend.List()
	require.NoError(t, err)
	assert.Contains(t, clusters, "lock-test")
}

func TestConfigValidationIntegration(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{
			name: "valid k3s config",
			yaml: `apiVersion: ironkube.dev/v1alpha2
kind: Cluster
metadata:
  name: valid-k3s
spec:
  distro: k3s
  version: v1.31.0+k3s1
  infrastructure:
    provider: ssh
  controlPlane:
    replicas: 1
    nodes:
      - host: 10.0.0.1
  security:
    profile: minimal`,
			wantErr: false,
		},
		{
			name: "missing name",
			yaml: `apiVersion: ironkube.dev/v1alpha2
kind: Cluster
metadata:
  name: ""
spec:
  distro: k3s
  version: v1.31.0
  controlPlane:
    replicas: 1
  security:
    profile: minimal`,
			wantErr: true,
		},
		{
			name: "invalid distro",
			yaml: `apiVersion: ironkube.dev/v1alpha2
kind: Cluster
metadata:
  name: bad-distro
spec:
  distro: rke
  version: v1.31.0
  controlPlane:
    replicas: 1
  security:
    profile: minimal`,
			wantErr: true,
		},
		{
			name: "even HA replicas",
			yaml: `apiVersion: ironkube.dev/v1alpha2
kind: Cluster
metadata:
  name: bad-ha
spec:
  distro: k3s
  version: v1.31.0
  controlPlane:
    replicas: 2
  security:
    profile: minimal`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := v1alpha2.LoadBytes([]byte(tc.yaml))
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func configToDesired(cfg *v1alpha2.ClusterConfig) engine.DesiredState {
	var machines []provider.MachineSpec

	for _, n := range cfg.Spec.ControlPlane.Nodes {
		machines = append(machines, provider.MachineSpec{
			Role:    "control-plane",
			Host:    n.Host,
			User:    n.User,
			Port:    n.Port,
			KeyPath: n.KeyPath,
		})
	}

	for _, pool := range cfg.Spec.Workers {
		for _, n := range pool.Nodes {
			machines = append(machines, provider.MachineSpec{
				Role:    "worker",
				Host:    n.Host,
				User:    n.User,
				Port:    n.Port,
				KeyPath: n.KeyPath,
				Labels:  pool.Labels,
			})
		}
	}

	var addons []provider.AddonSpec
	for _, a := range cfg.Spec.Addons {
		if !a.Enabled {
			continue
		}
		addons = append(addons, provider.AddonSpec{
			Name:       a.Name,
			Version:    a.Version,
			Namespace:  a.Namespace,
			Repository: a.Repository,
			Chart:      a.Chart,
			Values:     a.Values,
		})
	}

	return engine.DesiredState{
		Machines: machines,
		Addons:   addons,
		Version:  cfg.Spec.Version,
	}
}

type cisSecurityAdapter struct {
	profile security.Profile
}

func (a *cisSecurityAdapter) Name() string             { return "cis" }
func (a *cisSecurityAdapter) APIServerFlags() []string { return a.profile.APIServerFlags() }
func (a *cisSecurityAdapter) KubeletFlags() []string   { return a.profile.KubeletFlags() }
func (a *cisSecurityAdapter) EtcdFlags() []string      { return a.profile.EtcdFlags() }

type mockHetznerClient struct{}

func (m *mockHetznerClient) CreateServer(_ context.Context, name string, _ hetzner.ServerCreateOpts) (*hetzner.ServerResult, error) {
	return &hetzner.ServerResult{ID: "srv-1", Name: name, Status: "running", PublicIP: "1.2.3.4"}, nil
}
func (m *mockHetznerClient) GetServer(_ context.Context, _ string) (*hetzner.ServerResult, error) {
	return &hetzner.ServerResult{ID: "srv-1", Status: "running"}, nil
}
func (m *mockHetznerClient) DeleteServer(_ context.Context, _ string) error { return nil }

type mockDockerClient struct{}

func (m *mockDockerClient) CreateContainer(_ context.Context, _ string, _ docker.ContainerCreateOpts) (string, error) {
	return "container-1", nil
}
func (m *mockDockerClient) StartContainer(_ context.Context, _ string) error    { return nil }
func (m *mockDockerClient) StopContainer(_ context.Context, _ string) error     { return nil }
func (m *mockDockerClient) RemoveContainer(_ context.Context, _ string) error   { return nil }
func (m *mockDockerClient) InspectContainer(_ context.Context, _ string) (*docker.ContainerInfo, error) {
	return &docker.ContainerInfo{ID: "container-1", Status: "running", IP: "172.17.0.2"}, nil
}

type mockHelmClient struct{}

func (m *mockHelmClient) Install(_ context.Context, _ helm.InstallOpts) error   { return nil }
func (m *mockHelmClient) Upgrade(_ context.Context, _ helm.UpgradeOpts) error   { return nil }
func (m *mockHelmClient) Uninstall(_ context.Context, _ helm.UninstallOpts) error { return nil }
func (m *mockHelmClient) Status(_ context.Context, _, _ string) (*helm.ReleaseStatus, error) {
	return &helm.ReleaseStatus{Status: "deployed"}, nil
}
func (m *mockHelmClient) AddRepo(_ context.Context, _, _ string) error { return nil }

type mockManifestApplier struct{}

func (m *mockManifestApplier) Apply(_ context.Context, _ []byte, _ string, _ []byte) error  { return nil }
func (m *mockManifestApplier) Delete(_ context.Context, _ []byte, _ string, _ []byte) error { return nil }

type mockManifestFetcher struct{}

func (m *mockManifestFetcher) Fetch(_ context.Context, _ string) ([]byte, error) {
	return []byte("kind: Deployment"), nil
}

type mockBinaryFetcher struct{}

func (m *mockBinaryFetcher) Fetch(_ context.Context, name, _, _ string) ([]byte, error) {
	return []byte("binary-" + name), nil
}

type mockImageExporter struct{}

func (m *mockImageExporter) Export(_ context.Context, _ []string, _ string) error { return nil }

type mockChartFetcher struct{}

func (m *mockChartFetcher) Fetch(_ context.Context, _, chart, _ string) ([]byte, error) {
	return []byte("chart-" + chart), nil
}
