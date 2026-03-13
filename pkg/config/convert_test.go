package config

import (
	"testing"

	"github.com/rohitg00/ironkube/pkg/config/v1alpha2"
)

func TestFromV1Alpha2_FullConfig(t *testing.T) {
	v2 := &v1alpha2.ClusterConfig{
		APIVersion: "ironkube.dev/v1alpha2",
		Kind:       "Cluster",
		Metadata: v1alpha2.Metadata{
			Name:   "prod-cluster",
			Labels: map[string]string{"env": "prod"},
		},
		Spec: v1alpha2.ClusterSpec{
			Distro:  "k3s",
			Version: "1.31.0",
			Infrastructure: v1alpha2.Infrastructure{
				Provider: "bare-metal",
				SSH:      &v1alpha2.SSHConfig{KeyPath: "/root/.ssh/id_ed25519"},
			},
			ControlPlane: v1alpha2.ControlPlane{
				Replicas: 3,
				Nodes: []v1alpha2.Node{
					{Host: "10.0.0.1", User: "ubuntu", Port: 22},
					{Host: "10.0.0.2", User: "ubuntu", Port: 22},
					{Host: "10.0.0.3", User: "ubuntu", Port: 22},
				},
			},
			Workers: []v1alpha2.NodePool{
				{
					Name: "gpu-pool",
					Nodes: []v1alpha2.Node{
						{Host: "10.0.1.1", User: "ubuntu", Port: 2222},
					},
				},
				{
					Name: "general",
					Nodes: []v1alpha2.Node{
						{Host: "10.0.2.1"},
						{Host: "10.0.2.2"},
					},
				},
			},
			Networking: v1alpha2.Networking{
				PodCIDR:     "10.244.0.0/16",
				ServiceCIDR: "10.96.0.0/12",
			},
			Security: v1alpha2.Security{
				Profile:          "cis-hardened",
				EncryptionAtRest: true,
				AuditLogging:     true,
			},
		},
	}

	got := FromV1Alpha2(v2)

	if got.APIVersion != "ironkube.dev/v1alpha1" {
		t.Errorf("APIVersion = %q, want %q", got.APIVersion, "ironkube.dev/v1alpha1")
	}
	if got.Kind != "Cluster" {
		t.Errorf("Kind = %q, want %q", got.Kind, "Cluster")
	}
	if got.Metadata.Name != "prod-cluster" {
		t.Errorf("Metadata.Name = %q, want %q", got.Metadata.Name, "prod-cluster")
	}
	if got.Spec.Distro != "k3s" {
		t.Errorf("Distro = %q, want %q", got.Spec.Distro, "k3s")
	}
	if got.Spec.Version != "1.31.0" {
		t.Errorf("Version = %q, want %q", got.Spec.Version, "1.31.0")
	}
	if got.Spec.Security.Profile != "cis-hardened" {
		t.Errorf("Security.Profile = %q, want %q", got.Spec.Security.Profile, "cis-hardened")
	}
	if got.Spec.Networking.PodCIDR != "10.244.0.0/16" {
		t.Errorf("PodCIDR = %q, want %q", got.Spec.Networking.PodCIDR, "10.244.0.0/16")
	}
	if got.Spec.Networking.SvcCIDR != "10.96.0.0/12" {
		t.Errorf("SvcCIDR = %q, want %q", got.Spec.Networking.SvcCIDR, "10.96.0.0/12")
	}
}

func TestFromV1Alpha2_ControlPlaneNodes(t *testing.T) {
	v2 := &v1alpha2.ClusterConfig{
		Spec: v1alpha2.ClusterSpec{
			Infrastructure: v1alpha2.Infrastructure{
				SSH: &v1alpha2.SSHConfig{KeyPath: "/global/key"},
			},
			ControlPlane: v1alpha2.ControlPlane{
				Replicas: 1,
				Nodes: []v1alpha2.Node{
					{Host: "cp1.example.com", User: "admin", Port: 2222},
				},
			},
		},
	}

	got := FromV1Alpha2(v2)

	if got.Spec.ControlPlane.Replicas != 1 {
		t.Errorf("Replicas = %d, want 1", got.Spec.ControlPlane.Replicas)
	}
	if len(got.Spec.ControlPlane.Nodes) != 1 {
		t.Fatalf("len(Nodes) = %d, want 1", len(got.Spec.ControlPlane.Nodes))
	}
	n := got.Spec.ControlPlane.Nodes[0]
	if n.Host != "cp1.example.com" {
		t.Errorf("Host = %q, want %q", n.Host, "cp1.example.com")
	}
	if n.User != "admin" {
		t.Errorf("User = %q, want %q", n.User, "admin")
	}
	if n.Port != 2222 {
		t.Errorf("Port = %d, want 2222", n.Port)
	}
	if n.KeyPath != "/global/key" {
		t.Errorf("KeyPath = %q, want %q", n.KeyPath, "/global/key")
	}
}

func TestFromV1Alpha2_NodeKeyPathOverridesGlobal(t *testing.T) {
	v2 := &v1alpha2.ClusterConfig{
		Spec: v1alpha2.ClusterSpec{
			Infrastructure: v1alpha2.Infrastructure{
				SSH: &v1alpha2.SSHConfig{KeyPath: "/global/key"},
			},
			ControlPlane: v1alpha2.ControlPlane{
				Nodes: []v1alpha2.Node{
					{Host: "10.0.0.1", KeyPath: "/node/specific/key"},
				},
			},
		},
	}

	got := FromV1Alpha2(v2)

	if got.Spec.ControlPlane.Nodes[0].KeyPath != "/node/specific/key" {
		t.Errorf("KeyPath = %q, want %q (node-level should override global)",
			got.Spec.ControlPlane.Nodes[0].KeyPath, "/node/specific/key")
	}
}

func TestFromV1Alpha2_NoSSHConfig(t *testing.T) {
	v2 := &v1alpha2.ClusterConfig{
		Spec: v1alpha2.ClusterSpec{
			Infrastructure: v1alpha2.Infrastructure{
				Provider: "docker",
			},
			ControlPlane: v1alpha2.ControlPlane{
				Nodes: []v1alpha2.Node{
					{Host: "10.0.0.1"},
				},
			},
		},
	}

	got := FromV1Alpha2(v2)

	if got.Spec.ControlPlane.Nodes[0].KeyPath != "" {
		t.Errorf("KeyPath = %q, want empty when no SSH config", got.Spec.ControlPlane.Nodes[0].KeyPath)
	}
}

func TestFromV1Alpha2_WorkerPools(t *testing.T) {
	v2 := &v1alpha2.ClusterConfig{
		Spec: v1alpha2.ClusterSpec{
			Infrastructure: v1alpha2.Infrastructure{
				SSH: &v1alpha2.SSHConfig{KeyPath: "/default/key"},
			},
			Workers: []v1alpha2.NodePool{
				{
					Name: "pool-a",
					Nodes: []v1alpha2.Node{
						{Host: "w1", User: "deploy", Port: 22},
						{Host: "w2", User: "deploy", Port: 22, KeyPath: "/custom/key"},
					},
				},
				{
					Name: "pool-b",
					Nodes: []v1alpha2.Node{
						{Host: "w3"},
					},
				},
			},
		},
	}

	got := FromV1Alpha2(v2)

	if len(got.Spec.Workers) != 2 {
		t.Fatalf("len(Workers) = %d, want 2", len(got.Spec.Workers))
	}

	poolA := got.Spec.Workers[0]
	if poolA.Name != "pool-a" {
		t.Errorf("Workers[0].Name = %q, want %q", poolA.Name, "pool-a")
	}
	if len(poolA.Nodes) != 2 {
		t.Fatalf("Workers[0] node count = %d, want 2", len(poolA.Nodes))
	}
	if poolA.Nodes[0].KeyPath != "/default/key" {
		t.Errorf("Workers[0].Nodes[0].KeyPath = %q, want global default", poolA.Nodes[0].KeyPath)
	}
	if poolA.Nodes[1].KeyPath != "/custom/key" {
		t.Errorf("Workers[0].Nodes[1].KeyPath = %q, want node override", poolA.Nodes[1].KeyPath)
	}

	poolB := got.Spec.Workers[1]
	if poolB.Name != "pool-b" {
		t.Errorf("Workers[1].Name = %q, want %q", poolB.Name, "pool-b")
	}
	if poolB.Nodes[0].KeyPath != "/default/key" {
		t.Errorf("Workers[1].Nodes[0].KeyPath = %q, want global default", poolB.Nodes[0].KeyPath)
	}
}

func TestFromV1Alpha2_EmptyWorkers(t *testing.T) {
	v2 := &v1alpha2.ClusterConfig{
		Spec: v1alpha2.ClusterSpec{
			ControlPlane: v1alpha2.ControlPlane{
				Replicas: 1,
				Nodes:    []v1alpha2.Node{{Host: "10.0.0.1"}},
			},
		},
	}

	got := FromV1Alpha2(v2)

	if got.Spec.Workers == nil {
		t.Fatal("Workers should not be nil (want empty slice)")
	}
	if len(got.Spec.Workers) != 0 {
		t.Errorf("len(Workers) = %d, want 0", len(got.Spec.Workers))
	}
}

func TestFromV1Alpha2_ServiceCIDRMapping(t *testing.T) {
	v2 := &v1alpha2.ClusterConfig{
		Spec: v1alpha2.ClusterSpec{
			Networking: v1alpha2.Networking{
				ServiceCIDR: "10.96.0.0/12",
			},
		},
	}

	got := FromV1Alpha2(v2)

	if got.Spec.Networking.SvcCIDR != "10.96.0.0/12" {
		t.Errorf("SvcCIDR = %q, want %q (ServiceCIDR should map to SvcCIDR)",
			got.Spec.Networking.SvcCIDR, "10.96.0.0/12")
	}
}

func TestFromV1Alpha2_MinimalConfig(t *testing.T) {
	v2 := &v1alpha2.ClusterConfig{
		Metadata: v1alpha2.Metadata{Name: "minimal"},
		Spec: v1alpha2.ClusterSpec{
			Distro:  "kubeadm",
			Version: "1.30.0",
		},
	}

	got := FromV1Alpha2(v2)

	if got.Metadata.Name != "minimal" {
		t.Errorf("Name = %q, want %q", got.Metadata.Name, "minimal")
	}
	if got.Spec.Distro != "kubeadm" {
		t.Errorf("Distro = %q, want %q", got.Spec.Distro, "kubeadm")
	}
	if got.Spec.Workers == nil {
		t.Error("Workers should be empty slice, not nil")
	}
}

func TestFromV1Alpha2_GlobalKeyPathAppliedToAllNodes(t *testing.T) {
	v2 := &v1alpha2.ClusterConfig{
		Spec: v1alpha2.ClusterSpec{
			Infrastructure: v1alpha2.Infrastructure{
				SSH: &v1alpha2.SSHConfig{KeyPath: "/shared/key"},
			},
			ControlPlane: v1alpha2.ControlPlane{
				Nodes: []v1alpha2.Node{
					{Host: "cp1"},
					{Host: "cp2"},
				},
			},
			Workers: []v1alpha2.NodePool{
				{
					Name:  "workers",
					Nodes: []v1alpha2.Node{{Host: "w1"}, {Host: "w2"}},
				},
			},
		},
	}

	got := FromV1Alpha2(v2)

	for i, n := range got.Spec.ControlPlane.Nodes {
		if n.KeyPath != "/shared/key" {
			t.Errorf("ControlPlane.Nodes[%d].KeyPath = %q, want /shared/key", i, n.KeyPath)
		}
	}
	for i, n := range got.Spec.Workers[0].Nodes {
		if n.KeyPath != "/shared/key" {
			t.Errorf("Workers[0].Nodes[%d].KeyPath = %q, want /shared/key", i, n.KeyPath)
		}
	}
}

func TestFromV1Alpha2_EmptyNodesSlice(t *testing.T) {
	v2 := &v1alpha2.ClusterConfig{
		Spec: v1alpha2.ClusterSpec{
			ControlPlane: v1alpha2.ControlPlane{
				Replicas: 3,
			},
		},
	}

	got := FromV1Alpha2(v2)

	if got.Spec.ControlPlane.Nodes == nil {
		t.Fatal("ControlPlane.Nodes should be empty slice, not nil")
	}
	if len(got.Spec.ControlPlane.Nodes) != 0 {
		t.Errorf("len(ControlPlane.Nodes) = %d, want 0", len(got.Spec.ControlPlane.Nodes))
	}
}
