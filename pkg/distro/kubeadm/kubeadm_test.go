package kubeadm

import (
	"testing"

	"github.com/rohitg00/ironkube/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestName(t *testing.T) {
	p := New()
	assert.Equal(t, "kubeadm", p.Name())
}

func TestValidateVersion(t *testing.T) {
	p := New()

	t.Run("valid version", func(t *testing.T) {
		assert.NoError(t, p.ValidateVersion("v1.28.2"))
	})

	t.Run("valid version with patch", func(t *testing.T) {
		assert.NoError(t, p.ValidateVersion("v1.29.0"))
	})

	t.Run("missing v prefix", func(t *testing.T) {
		err := p.ValidateVersion("1.28.2")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "start with 'v'")
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
			Version: "v1.28.2",
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
	assert.Contains(t, script, "kubeadm init")
	assert.Contains(t, script, "--pod-network-cidr=10.244.0.0/16")
	assert.Contains(t, script, "--service-cidr=10.96.0.0/12")
	assert.NotContains(t, script, "--control-plane-endpoint")
}

func TestServerInstallScriptHA(t *testing.T) {
	p := New()
	node := config.Node{Host: "10.0.0.1", User: "root", Port: 22}
	cfg := &config.ClusterConfig{
		Spec: config.ClusterSpec{
			Version: "v1.28.2",
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

	script := p.ServerInstallScript(node, cfg, "hatoken", true)
	assert.Contains(t, script, "--control-plane-endpoint=10.0.0.1:6443")
	assert.Contains(t, script, "--upload-certs")
}

func TestServerInstallScriptJoin(t *testing.T) {
	p := New()
	node := config.Node{Host: "10.0.0.2", User: "root", Port: 22}
	cfg := &config.ClusterConfig{
		Spec: config.ClusterSpec{
			Version: "v1.28.2",
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

	script := p.ServerInstallScript(node, cfg, "jointoken", false)
	assert.Contains(t, script, "kubeadm join")
	assert.Contains(t, script, "--control-plane")
	assert.Contains(t, script, "10.0.0.1:6443")
}

func TestAgentInstallScript(t *testing.T) {
	p := New()
	node := config.Node{Host: "10.0.0.10", User: "root", Port: 22}
	cfg := &config.ClusterConfig{
		Spec: config.ClusterSpec{
			Version: "v1.28.2",
		},
	}

	script := p.AgentInstallScript(node, cfg, "10.0.0.1:6443", "workertoken")
	assert.Contains(t, script, "kubeadm join")
	assert.Contains(t, script, "10.0.0.1:6443")
	assert.Contains(t, script, "workertoken")
	assert.NotContains(t, script, "--control-plane")
}

func TestKubeconfigPath(t *testing.T) {
	p := New()
	assert.Equal(t, "/etc/kubernetes/admin.conf", p.KubeconfigPath())
}

func TestUninstallCmd(t *testing.T) {
	p := New()

	t.Run("server", func(t *testing.T) {
		cmd := p.UninstallCmd("server")
		assert.Contains(t, cmd, "kubeadm reset")
	})

	t.Run("agent", func(t *testing.T) {
		cmd := p.UninstallCmd("agent")
		assert.Contains(t, cmd, "kubeadm reset")
	})
}

func TestSecurityArgs(t *testing.T) {
	p := New()

	t.Run("apiserver and etcd flags", func(t *testing.T) {
		result := p.SecurityArgs(
			[]string{"--audit-log-path=/var/log/audit.log", "--audit-log-maxage=30"},
			[]string{"--protect-kernel-defaults=true"},
			[]string{"--cipher-suites=TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"},
		)
		assert.Contains(t, result, "--apiserver-extra-args=audit-log-path=/var/log/audit.log")
		assert.Contains(t, result, "--apiserver-extra-args=audit-log-maxage=30")
		assert.Contains(t, result, "--etcd-extra-args=cipher-suites=TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256")
	})

	t.Run("kubelet flags ignored by kubeadm", func(t *testing.T) {
		result := p.SecurityArgs(nil, []string{"--protect-kernel-defaults=true"}, nil)
		assert.Empty(t, result)
	})

	t.Run("empty flags", func(t *testing.T) {
		result := p.SecurityArgs(nil, nil, nil)
		assert.Empty(t, result)
	})

	t.Run("parseFlag splits key=value", func(t *testing.T) {
		result := p.SecurityArgs([]string{"--key=value"}, nil, nil)
		assert.Contains(t, result, "--apiserver-extra-args=key=value")
	})

	t.Run("parseFlag without value", func(t *testing.T) {
		result := p.SecurityArgs([]string{"--enable-admission-plugins"}, nil, nil)
		assert.Contains(t, result, "--apiserver-extra-args=enable-admission-plugins=")
	})
}
