package k3s

import (
	"strings"
	"testing"

	"github.com/rohitg00/ironkube/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestName(t *testing.T) {
	p := New()
	assert.Equal(t, "k3s", p.Name())
}

func TestValidateVersion(t *testing.T) {
	p := New()

	t.Run("valid version", func(t *testing.T) {
		assert.NoError(t, p.ValidateVersion("v1.28.2+k3s1"))
	})

	t.Run("valid version with k3s2", func(t *testing.T) {
		assert.NoError(t, p.ValidateVersion("v1.29.0+k3s2"))
	})

	t.Run("missing v prefix", func(t *testing.T) {
		err := p.ValidateVersion("1.28.2+k3s1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "start with 'v'")
	})

	t.Run("missing k3s suffix", func(t *testing.T) {
		err := p.ValidateVersion("v1.28.2")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "+k3s")
	})

	t.Run("empty version", func(t *testing.T) {
		err := p.ValidateVersion("")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})
}

func TestServerInstallScriptInit(t *testing.T) {
	p := New()
	node := config.Node{Host: "10.0.0.1", User: "root", Port: 22}
	cfg := &config.ClusterConfig{
		Spec: config.ClusterSpec{
			Version: "v1.28.2+k3s1",
			ControlPlane: config.ControlPlane{
				Replicas: 1,
				Nodes:    []config.Node{node},
			},
			Networking: config.Networking{
				PodCIDR: "10.244.0.0/16",
				SvcCIDR: "10.96.0.0/12",
			},
		},
	}

	script := p.ServerInstallScript(node, cfg, "", true)
	assert.Contains(t, script, "INSTALL_K3S_VERSION=v1.28.2+k3s1")
	assert.Contains(t, script, "--cluster-cidr=10.244.0.0/16")
	assert.Contains(t, script, "--service-cidr=10.96.0.0/12")
	assert.NotContains(t, script, "--cluster-init")
}

func TestServerInstallScriptHAInit(t *testing.T) {
	p := New()
	node := config.Node{Host: "10.0.0.1", User: "root", Port: 22}
	cfg := &config.ClusterConfig{
		Spec: config.ClusterSpec{
			Version: "v1.28.2+k3s1",
			ControlPlane: config.ControlPlane{
				Replicas: 3,
				Nodes: []config.Node{
					{Host: "10.0.0.1", User: "root", Port: 22},
					{Host: "10.0.0.2", User: "root", Port: 22},
					{Host: "10.0.0.3", User: "root", Port: 22},
				},
			},
			Networking: config.Networking{
				PodCIDR: "10.244.0.0/16",
				SvcCIDR: "10.96.0.0/12",
			},
		},
	}

	script := p.ServerInstallScript(node, cfg, "mysecrettoken", true)
	assert.Contains(t, script, "--cluster-init")
	assert.Contains(t, script, "K3S_TOKEN=mysecrettoken")
}

func TestServerInstallScriptHAJoin(t *testing.T) {
	p := New()
	node := config.Node{Host: "10.0.0.2", User: "root", Port: 22}
	cfg := &config.ClusterConfig{
		Spec: config.ClusterSpec{
			Version: "v1.28.2+k3s1",
			ControlPlane: config.ControlPlane{
				Replicas: 3,
				Nodes: []config.Node{
					{Host: "10.0.0.1", User: "root", Port: 22},
					{Host: "10.0.0.2", User: "root", Port: 22},
					{Host: "10.0.0.3", User: "root", Port: 22},
				},
			},
			Networking: config.Networking{
				PodCIDR: "10.244.0.0/16",
			},
		},
	}

	script := p.ServerInstallScript(node, cfg, "mysecrettoken", false)
	assert.Contains(t, script, "K3S_URL=https://10.0.0.1:6443")
	assert.Contains(t, script, "K3S_TOKEN=mysecrettoken")
	assert.NotContains(t, script, "--cluster-init")
}

func TestAgentInstallScript(t *testing.T) {
	p := New()
	node := config.Node{Host: "10.0.0.10", User: "root", Port: 22}
	cfg := &config.ClusterConfig{
		Spec: config.ClusterSpec{
			Version: "v1.28.2+k3s1",
		},
	}

	script := p.AgentInstallScript(node, cfg, "https://10.0.0.1:6443", "agenttoken")
	assert.Contains(t, script, "K3S_URL=https://10.0.0.1:6443")
	assert.Contains(t, script, "K3S_TOKEN=agenttoken")
	assert.Contains(t, script, "agent")
	assert.False(t, strings.Contains(script, "sh -s - server"), "agent script should not contain 'server' mode")
}

func TestKubeconfigPath(t *testing.T) {
	p := New()
	assert.Equal(t, "/etc/rancher/k3s/k3s.yaml", p.KubeconfigPath())
}

func TestUninstallCmd(t *testing.T) {
	p := New()

	t.Run("server", func(t *testing.T) {
		cmd := p.UninstallCmd("server")
		assert.Contains(t, cmd, "k3s-uninstall.sh")
		assert.NotContains(t, cmd, "agent")
	})

	t.Run("agent", func(t *testing.T) {
		cmd := p.UninstallCmd("agent")
		assert.Contains(t, cmd, "k3s-agent-uninstall.sh")
	})
}
