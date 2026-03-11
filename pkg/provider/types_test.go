package provider

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMachineSpecFields(t *testing.T) {
	spec := MachineSpec{
		Role:    "control-plane",
		Host:    "10.0.0.1",
		User:    "ubuntu",
		Port:    22,
		KeyPath: "/home/ubuntu/.ssh/id_rsa",
		Labels: map[string]string{
			"zone": "us-east-1a",
		},
		Taints: []Taint{
			{Key: "node-role.kubernetes.io/control-plane", Effect: "NoSchedule"},
		},
	}

	assert.Equal(t, "control-plane", spec.Role)
	assert.Equal(t, "10.0.0.1", spec.Host)
	assert.Equal(t, "ubuntu", spec.User)
	assert.Equal(t, 22, spec.Port)
	assert.Equal(t, "/home/ubuntu/.ssh/id_rsa", spec.KeyPath)
	assert.Equal(t, "us-east-1a", spec.Labels["zone"])
	require.Len(t, spec.Taints, 1)
	assert.Equal(t, "NoSchedule", spec.Taints[0].Effect)
	assert.Empty(t, spec.Taints[0].Value)
}

func TestMachineStatusConstants(t *testing.T) {
	assert.Equal(t, MachineStatus("provisioning"), MachineStatusProvisioning)
	assert.Equal(t, MachineStatus("ready"), MachineStatusReady)
	assert.Equal(t, MachineStatus("upgrading"), MachineStatusUpgrading)
	assert.Equal(t, MachineStatus("draining"), MachineStatusDraining)
	assert.Equal(t, MachineStatus("deleted"), MachineStatusDeleted)
}

func TestMachineConstruction(t *testing.T) {
	now := time.Now()
	m := Machine{
		ID:         "m-001",
		ProviderID: "aws://i-abc123",
		Spec: MachineSpec{
			Role: "worker",
			Host: "10.0.0.5",
			User: "ec2-user",
			Port: 22,
		},
		Status:    MachineStatusProvisioning,
		CreatedAt: now,
	}

	assert.Equal(t, "m-001", m.ID)
	assert.Equal(t, "aws://i-abc123", m.ProviderID)
	assert.Equal(t, "worker", m.Spec.Role)
	assert.Equal(t, MachineStatusProvisioning, m.Status)
	assert.Equal(t, now, m.CreatedAt)
}

func TestBootstrapDataConstruction(t *testing.T) {
	bd := BootstrapData{
		Scripts: []Script{
			{Name: "install-kubeadm", Content: "apt-get install -y kubeadm", RunAs: "root"},
			{Name: "init-cluster", Content: "kubeadm init"},
		},
		Files: []FileToWrite{
			{Path: "/etc/kubernetes/config.yaml", Content: "apiVersion: v1", Mode: 0644},
		},
		EnvVars: map[string]string{
			"KUBECONFIG": "/etc/kubernetes/admin.conf",
		},
		WaitCheck: &WaitCondition{
			Command:  "kubectl get nodes",
			Expected: "Ready",
			Interval: 5 * time.Second,
			Timeout:  2 * time.Minute,
		},
	}

	require.Len(t, bd.Scripts, 2)
	assert.Equal(t, "install-kubeadm", bd.Scripts[0].Name)
	assert.Equal(t, "root", bd.Scripts[0].RunAs)
	require.Len(t, bd.Files, 1)
	assert.Equal(t, uint32(0644), bd.Files[0].Mode)
	assert.Equal(t, "/etc/kubernetes/admin.conf", bd.EnvVars["KUBECONFIG"])
	require.NotNil(t, bd.WaitCheck)
	assert.Equal(t, 5*time.Second, bd.WaitCheck.Interval)
	assert.Equal(t, 2*time.Minute, bd.WaitCheck.Timeout)
}

func TestHealthReportAllHealthyTrue(t *testing.T) {
	r := HealthReport{
		ClusterName: "test-cluster",
		Nodes: []NodeHealth{
			{Name: "node-1", Ready: true, Version: "v1.30.0", Roles: "control-plane"},
			{Name: "node-2", Ready: true, Version: "v1.30.0", Roles: "worker"},
			{Name: "node-3", Ready: true, Version: "v1.30.0", Roles: "worker"},
		},
	}

	assert.True(t, r.AllHealthy())
}

func TestHealthReportAllHealthyFalse(t *testing.T) {
	r := HealthReport{
		ClusterName: "test-cluster",
		Nodes: []NodeHealth{
			{Name: "node-1", Ready: true, Version: "v1.30.0", Roles: "control-plane"},
			{Name: "node-2", Ready: false, Version: "v1.30.0", Roles: "worker"},
		},
	}

	assert.False(t, r.AllHealthy())
}

func TestHealthReportReadyCount(t *testing.T) {
	r := HealthReport{
		ClusterName: "mixed-cluster",
		Nodes: []NodeHealth{
			{Name: "node-1", Ready: true},
			{Name: "node-2", Ready: false},
			{Name: "node-3", Ready: true},
			{Name: "node-4", Ready: false},
			{Name: "node-5", Ready: true},
		},
	}

	assert.Equal(t, 3, r.ReadyCount())
}

func TestHealthReportTotalCount(t *testing.T) {
	r := HealthReport{
		ClusterName: "cluster",
		Nodes: []NodeHealth{
			{Name: "a"},
			{Name: "b"},
			{Name: "c"},
		},
	}

	assert.Equal(t, 3, r.TotalCount())
}

func TestEmptyHealthReport(t *testing.T) {
	r := HealthReport{
		ClusterName: "empty",
	}

	assert.True(t, r.AllHealthy())
	assert.Equal(t, 0, r.ReadyCount())
	assert.Equal(t, 0, r.TotalCount())
	assert.Nil(t, r.Etcd)
	assert.Empty(t, r.Certs)
	assert.Empty(t, r.Addons)
}

func TestEtcdHealth(t *testing.T) {
	eh := EtcdHealth{
		Healthy: true,
		Members: []EtcdMember{
			{Name: "etcd-0", Endpoint: "https://10.0.0.1:2379", IsLeader: true, DbSize: 1024 * 1024 * 50},
			{Name: "etcd-1", Endpoint: "https://10.0.0.2:2379", IsLeader: false, DbSize: 1024 * 1024 * 48},
			{Name: "etcd-2", Endpoint: "https://10.0.0.3:2379", IsLeader: false, DbSize: 1024 * 1024 * 49},
		},
		DbSize:  1024 * 1024 * 50,
		Version: "3.5.12",
	}

	assert.True(t, eh.Healthy)
	require.Len(t, eh.Members, 3)
	assert.True(t, eh.Members[0].IsLeader)
	assert.False(t, eh.Members[1].IsLeader)
	assert.Equal(t, int64(1024*1024*50), eh.DbSize)
	assert.Equal(t, "3.5.12", eh.Version)
}

func TestAddonSpec(t *testing.T) {
	addon := AddonSpec{
		Name:       "cilium",
		Type:       "helm",
		Repository: "https://helm.cilium.io",
		Chart:      "cilium",
		Version:    "1.15.0",
		Namespace:  "kube-system",
		Values: map[string]any{
			"hubble": map[string]any{
				"enabled": true,
			},
		},
		WaitReady: true,
	}

	assert.Equal(t, "cilium", addon.Name)
	assert.Equal(t, "helm", addon.Type)
	assert.Equal(t, "https://helm.cilium.io", addon.Repository)
	assert.Equal(t, "1.15.0", addon.Version)
	assert.Equal(t, "kube-system", addon.Namespace)
	assert.True(t, addon.WaitReady)
	hubble, ok := addon.Values["hubble"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, hubble["enabled"])
}

func TestDrainOptions(t *testing.T) {
	opts := DrainOptions{
		GracePeriod:      30,
		Timeout:          5 * time.Minute,
		IgnoreDaemonSets: true,
		DeleteLocalData:  false,
		Force:            false,
	}

	assert.Equal(t, 30, opts.GracePeriod)
	assert.Equal(t, 5*time.Minute, opts.Timeout)
	assert.True(t, opts.IgnoreDaemonSets)
	assert.False(t, opts.DeleteLocalData)
	assert.False(t, opts.Force)
}

func TestDrainOptionsZeroValues(t *testing.T) {
	opts := DrainOptions{}

	assert.Equal(t, 0, opts.GracePeriod)
	assert.Equal(t, time.Duration(0), opts.Timeout)
	assert.False(t, opts.IgnoreDaemonSets)
	assert.False(t, opts.DeleteLocalData)
	assert.False(t, opts.Force)
}

func TestClusterSpec(t *testing.T) {
	cs := ClusterSpec{
		Name:        "prod",
		Distro:      "kubeadm",
		Version:     "1.30.0",
		PodCIDR:     "10.244.0.0/16",
		ServiceCIDR: "10.96.0.0/12",
		HAEnabled:   true,
		FirstCPHost: "10.0.0.1",
	}

	assert.Equal(t, "prod", cs.Name)
	assert.Equal(t, "kubeadm", cs.Distro)
	assert.True(t, cs.HAEnabled)
	assert.Equal(t, "10.0.0.1", cs.FirstCPHost)
}

func TestCertInfo(t *testing.T) {
	expires := time.Date(2027, 3, 11, 0, 0, 0, 0, time.UTC)
	ci := CertInfo{
		Name:      "apiserver",
		Path:      "/etc/kubernetes/pki/apiserver.crt",
		ExpiresAt: expires,
		Issuer:    "kubernetes-ca",
	}

	assert.Equal(t, "apiserver", ci.Name)
	assert.Equal(t, "/etc/kubernetes/pki/apiserver.crt", ci.Path)
	assert.Equal(t, expires, ci.ExpiresAt)
	assert.Equal(t, "kubernetes-ca", ci.Issuer)
}

func TestHealthReportWithFullData(t *testing.T) {
	r := HealthReport{
		ClusterName: "full-cluster",
		Nodes: []NodeHealth{
			{Name: "cp-1", Ready: true, Version: "v1.30.0", Roles: "control-plane"},
			{Name: "w-1", Ready: true, Version: "v1.30.0", Roles: "worker"},
		},
		Certs: []CertInfo{
			{Name: "apiserver", Path: "/etc/kubernetes/pki/apiserver.crt"},
		},
		Etcd: &EtcdHealth{
			Healthy: true,
			Members: []EtcdMember{
				{Name: "etcd-0", Endpoint: "https://10.0.0.1:2379", IsLeader: true},
			},
		},
		Addons: []AddonStatus{
			{Name: "cilium", Installed: true, Version: "1.15.0", Ready: true},
			{Name: "metrics-server", Installed: true, Version: "0.7.0", Ready: false},
		},
	}

	assert.True(t, r.AllHealthy())
	assert.Equal(t, 2, r.ReadyCount())
	assert.Equal(t, 2, r.TotalCount())
	require.Len(t, r.Certs, 1)
	require.NotNil(t, r.Etcd)
	assert.True(t, r.Etcd.Healthy)
	require.Len(t, r.Addons, 2)
	assert.True(t, r.Addons[0].Ready)
	assert.False(t, r.Addons[1].Ready)
}
