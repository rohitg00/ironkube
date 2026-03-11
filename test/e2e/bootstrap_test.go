package e2e

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rohitg00/ironkube/pkg/config"
	"github.com/rohitg00/ironkube/pkg/engine"
	"github.com/rohitg00/ironkube/pkg/phases"
	"github.com/rohitg00/ironkube/pkg/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func skipIfNoVM(t *testing.T) {
	t.Helper()
	if os.Getenv("IRONKUBE_VM_E2E") != "1" {
		t.Skip("Set IRONKUBE_VM_E2E=1 to run VM-based e2e tests")
	}
}

func vmSSHConfig() ssh.Config {
	return ssh.Config{
		Host:    "127.0.0.1",
		Port:    62872,
		User:    "root",
		KeyPath: "keys/test_key",
		Timeout: 30 * time.Second,
	}
}

func TestVMSSHConnect(t *testing.T) {
	skipIfNoVM(t)

	client, err := ssh.NewClient(vmSSHConfig())
	require.NoError(t, err)
	defer client.Close()

	out, err := client.Run("systemctl is-system-running")
	require.NoError(t, err)
	assert.Contains(t, out, "running")
}

func TestVMSystemdAvailable(t *testing.T) {
	skipIfNoVM(t)

	client, err := ssh.NewClient(vmSSHConfig())
	require.NoError(t, err)
	defer client.Close()

	out, err := client.Run("which systemctl && uname -a")
	require.NoError(t, err)
	assert.Contains(t, out, "systemctl")
	assert.Contains(t, out, "Linux")
}

func TestFullK3sBootstrap(t *testing.T) {
	skipIfNoVM(t)

	cfg, err := config.Load("ironkube-vm-test.yaml")
	require.NoError(t, err)
	require.NoError(t, config.Validate(cfg))

	t.Cleanup(func() {
		client, err := ssh.NewClient(vmSSHConfig())
		if err != nil {
			return
		}
		defer client.Close()
		client.Run("/usr/local/bin/k3s-uninstall.sh")
	})

	tmpKubeconfig := t.TempDir() + "/kubeconfig"

	pipeline := engine.NewPipeline(
		&phases.ValidatePhase{Config: cfg},
		&phases.ConnectPhase{Config: cfg},
		&phases.BootstrapPhase{Config: cfg},
		&phases.FetchKubeconfigPhase{Config: cfg, OutputPath: tmpKubeconfig},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err = pipeline.Execute(ctx)
	require.NoError(t, err, "bootstrap pipeline should succeed")

	completedPhases := pipeline.CompletedPhases()
	assert.Contains(t, completedPhases, "validate")
	assert.Contains(t, completedPhases, "connect")
	assert.Contains(t, completedPhases, "bootstrap")
	assert.Contains(t, completedPhases, "kubeconfig")

	kubeconfigData, err := os.ReadFile(tmpKubeconfig)
	require.NoError(t, err)
	assert.Contains(t, string(kubeconfigData), "e2e-vm-test")
	assert.Contains(t, string(kubeconfigData), "server:")

	exec := ssh.NewExecutor()
	defer exec.Close()
	err = exec.Connect(vmSSHConfig())
	require.NoError(t, err)

	out, err := exec.RunOnHost("127.0.0.1", "k3s kubectl get nodes --no-headers")
	require.NoError(t, err)
	assert.Contains(t, out, "Ready")

	out, err = exec.RunOnHost("127.0.0.1", "k3s kubectl get pods -A --no-headers")
	require.NoError(t, err)
	trimmed := strings.TrimSpace(out)
	require.NotEmpty(t, trimmed, "should have system pods running")
	lines := strings.Split(trimmed, "\n")
	assert.Greater(t, len(lines), 0, "should have at least one system pod")
}

func TestValidateAndConnectPhases(t *testing.T) {
	skipIfNoVM(t)

	cfg, err := config.Load("ironkube-vm-test.yaml")
	require.NoError(t, err)

	pipeline := engine.NewPipeline(
		&phases.ValidatePhase{Config: cfg},
		&phases.ConnectPhase{Config: cfg},
	)

	err = pipeline.Execute(context.Background())
	require.NoError(t, err)

	v, ok := pipeline.State().Get("executor")
	require.True(t, ok)
	exec := v.(*ssh.Executor)
	defer exec.Close()

	out, err := exec.RunOnHost("127.0.0.1", "hostname")
	require.NoError(t, err)
	assert.NotEmpty(t, out)
}
