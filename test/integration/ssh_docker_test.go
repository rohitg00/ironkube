//go:build integration

package integration

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rohitg00/ironkube/pkg/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func startSSHContainer(t *testing.T) (containerName string, port int, keyPath string) {
	t.Helper()

	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available, skipping integration test")
	}

	tmpDir := t.TempDir()
	keyPath = filepath.Join(tmpDir, "id_ed25519")

	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-f", keyPath, "-N", "", "-q")
	require.NoError(t, cmd.Run(), "failed to generate SSH key pair")

	pubKey, err := os.ReadFile(keyPath + ".pub")
	require.NoError(t, err, "failed to read public key")

	containerName = fmt.Sprintf("ironkube-ssh-test-%d", time.Now().UnixNano()%100000)

	cleanup := func() {
		exec.Command("docker", "rm", "-f", containerName).Run()
	}
	cleanup()
	t.Cleanup(cleanup)

	startCmd := exec.Command("docker", "run", "-d",
		"--name", containerName,
		"-p", "0:22",
		"ubuntu:22.04",
		"bash", "-c",
		fmt.Sprintf(
			`apt-get update -qq && `+
				`apt-get install -y -qq openssh-server > /dev/null 2>&1 && `+
				`mkdir -p /run/sshd /root/.ssh && `+
				`echo '%s' > /root/.ssh/authorized_keys && `+
				`chmod 600 /root/.ssh/authorized_keys && `+
				`chmod 700 /root/.ssh && `+
				`sed -i 's/#PermitRootLogin prohibit-password/PermitRootLogin yes/' /etc/ssh/sshd_config && `+
				`sed -i 's/#PubkeyAuthentication yes/PubkeyAuthentication yes/' /etc/ssh/sshd_config && `+
				`/usr/sbin/sshd -D`,
			strings.TrimSpace(string(pubKey)),
		),
	)
	out, err := startCmd.CombinedOutput()
	if err != nil {
		t.Skipf("failed to start Docker container: %v\n%s", err, out)
	}

	portCmd := exec.Command("docker", "port", containerName, "22")
	portOut, err := portCmd.Output()
	require.NoError(t, err, "failed to get mapped port")

	portLines := strings.Split(strings.TrimSpace(string(portOut)), "\n")
	portStr := strings.TrimSpace(portLines[0])
	_, portPart, err := net.SplitHostPort(portStr)
	require.NoError(t, err, "failed to parse host:port from %q", portStr)

	port, err = strconv.Atoi(portPart)
	require.NoError(t, err, "failed to convert port %q to int", portPart)

	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		conn, dialErr := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 2*time.Second)
		if dialErr == nil {
			conn.Close()
			break
		}
		time.Sleep(2 * time.Second)
	}

	sshReady := false
	for time.Now().Before(deadline) {
		client, dialErr := ssh.NewClient(ssh.Config{
			Host:    "127.0.0.1",
			Port:    port,
			User:    "root",
			KeyPath: keyPath,
			Timeout: 5 * time.Second,
		})
		if dialErr == nil {
			client.Close()
			sshReady = true
			break
		}
		time.Sleep(2 * time.Second)
	}
	if !sshReady {
		t.Fatal("SSH server did not become ready within 90 seconds")
	}

	return containerName, port, keyPath
}

func TestSSH_DockerConnection(t *testing.T) {
	_, port, keyPath := startSSHContainer(t)

	client, err := ssh.NewClient(ssh.Config{
		Host:    "127.0.0.1",
		Port:    port,
		User:    "root",
		KeyPath: keyPath,
		Timeout: 10 * time.Second,
	})
	require.NoError(t, err)
	defer client.Close()

	assert.Equal(t, "127.0.0.1", client.Host())

	result, err := client.Run("echo hello-ironkube")
	require.NoError(t, err)
	assert.Contains(t, result, "hello-ironkube")
}

func TestSSH_DockerMultipleCommands(t *testing.T) {
	_, port, keyPath := startSSHContainer(t)

	client, err := ssh.NewClient(ssh.Config{
		Host:    "127.0.0.1",
		Port:    port,
		User:    "root",
		KeyPath: keyPath,
		Timeout: 10 * time.Second,
	})
	require.NoError(t, err)
	defer client.Close()

	out1, err := client.Run("whoami")
	require.NoError(t, err)
	assert.Contains(t, out1, "root")

	out2, err := client.Run("uname -s")
	require.NoError(t, err)
	assert.Contains(t, out2, "Linux")

	out3, err := client.Run("cat /etc/os-release | head -1")
	require.NoError(t, err)
	assert.Contains(t, out3, "PRETTY_NAME")
}

func TestSSH_DockerRunStream(t *testing.T) {
	_, port, keyPath := startSSHContainer(t)

	client, err := ssh.NewClient(ssh.Config{
		Host:    "127.0.0.1",
		Port:    port,
		User:    "root",
		KeyPath: keyPath,
		Timeout: 10 * time.Second,
	})
	require.NoError(t, err)
	defer client.Close()

	var stdout, stderr strings.Builder
	err = client.RunStream("echo stream-test && echo err-test >&2", &stdout, &stderr)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "stream-test")
	assert.Contains(t, stderr.String(), "err-test")
}

func TestSSH_ExecutorDockerConnection(t *testing.T) {
	_, port, keyPath := startSSHContainer(t)

	executor := ssh.NewExecutor()
	defer executor.Close()

	err := executor.Connect(ssh.Config{
		Host:    "127.0.0.1",
		Port:    port,
		User:    "root",
		KeyPath: keyPath,
		Timeout: 10 * time.Second,
	})
	require.NoError(t, err)

	output, err := executor.RunOnHost("127.0.0.1", "echo executor-works")
	require.NoError(t, err)
	assert.Contains(t, output, "executor-works")

	results := executor.RunOnAll(context.Background(), "hostname")
	require.Len(t, results, 1)
	assert.NoError(t, results[0].Err)
	assert.NotEmpty(t, results[0].Output)
	assert.Equal(t, "127.0.0.1", results[0].Host)
}

func TestSSH_ExecutorRunOnUnknownHost(t *testing.T) {
	_, port, keyPath := startSSHContainer(t)

	executor := ssh.NewExecutor()
	defer executor.Close()

	err := executor.Connect(ssh.Config{
		Host:    "127.0.0.1",
		Port:    port,
		User:    "root",
		KeyPath: keyPath,
		Timeout: 10 * time.Second,
	})
	require.NoError(t, err)

	_, err = executor.RunOnHost("10.99.99.99", "echo should-fail")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no connection for host")
}

func TestSSH_DockerCommandFailure(t *testing.T) {
	_, port, keyPath := startSSHContainer(t)

	client, err := ssh.NewClient(ssh.Config{
		Host:    "127.0.0.1",
		Port:    port,
		User:    "root",
		KeyPath: keyPath,
		Timeout: 10 * time.Second,
	})
	require.NoError(t, err)
	defer client.Close()

	_, err = client.Run("false")
	assert.Error(t, err)
}
