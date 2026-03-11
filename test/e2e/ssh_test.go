package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/rohitg00/ironkube/pkg/config"
	"github.com/rohitg00/ironkube/pkg/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func skipIfNoDocker(t *testing.T) {
	t.Helper()
	if os.Getenv("IRONKUBE_E2E") != "1" {
		t.Skip("Set IRONKUBE_E2E=1 to run e2e tests")
	}
}

func testSSHConfig() ssh.Config {
	return ssh.Config{
		Host:    "127.0.0.1",
		Port:    2222,
		User:    "root",
		KeyPath: "keys/test_key",
		Timeout: 10 * time.Second,
	}
}

func TestSSHConnect(t *testing.T) {
	skipIfNoDocker(t)

	client, err := ssh.NewClient(testSSHConfig())
	require.NoError(t, err)
	defer client.Close()

	assert.Equal(t, "127.0.0.1", client.Host())
}

func TestSSHRunCommand(t *testing.T) {
	skipIfNoDocker(t)

	client, err := ssh.NewClient(testSSHConfig())
	require.NoError(t, err)
	defer client.Close()

	output, err := client.Run("hostname")
	require.NoError(t, err)
	assert.NotEmpty(t, output)
}

func TestSSHRunMultipleCommands(t *testing.T) {
	skipIfNoDocker(t)

	client, err := ssh.NewClient(testSSHConfig())
	require.NoError(t, err)
	defer client.Close()

	out1, err := client.Run("echo hello")
	require.NoError(t, err)
	assert.Contains(t, out1, "hello")

	out2, err := client.Run("uname -m")
	require.NoError(t, err)
	assert.NotEmpty(t, out2)

	out3, err := client.Run("cat /etc/os-release | head -1")
	require.NoError(t, err)
	assert.Contains(t, out3, "PRETTY_NAME")
}

func TestSSHExecutorParallel(t *testing.T) {
	skipIfNoDocker(t)

	exec := ssh.NewExecutor()
	defer exec.Close()

	err := exec.Connect(testSSHConfig())
	require.NoError(t, err)

	output, err := exec.RunOnHost("127.0.0.1", "echo works")
	require.NoError(t, err)
	assert.Contains(t, output, "works")

	results := exec.RunOnAll(context.Background(), "whoami")
	require.Len(t, results, 1)
	assert.NoError(t, results[0].Err)
	assert.Contains(t, results[0].Output, "root")
}

func TestConfigLoadAndValidate(t *testing.T) {
	skipIfNoDocker(t)

	cfg, err := config.Load("ironkube-test.yaml")
	require.NoError(t, err)
	require.NoError(t, config.Validate(cfg))

	assert.Equal(t, "e2e-test", cfg.Metadata.Name)
	assert.Equal(t, "k3s", cfg.Spec.Distro)
	assert.Equal(t, 2222, cfg.Spec.ControlPlane.Nodes[0].Port)
}

func TestK3sInstallScript(t *testing.T) {
	skipIfNoDocker(t)

	client, err := ssh.NewClient(testSSHConfig())
	require.NoError(t, err)
	defer client.Close()

	out, err := client.Run("curl -sfL https://get.k3s.io -o /dev/null -w '%{http_code}'")
	require.NoError(t, err)
	assert.Contains(t, out, "200")
}
