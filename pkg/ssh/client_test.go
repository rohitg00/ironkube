package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gossh "golang.org/x/crypto/ssh"
)

func TestConfigDefaults(t *testing.T) {
	cfg := Config{
		Host: "10.0.0.1",
	}
	assert.Equal(t, "10.0.0.1", cfg.Host)
	assert.Equal(t, 0, cfg.Port)
	assert.Equal(t, "", cfg.User)
	assert.Equal(t, "", cfg.KeyPath)
	assert.Equal(t, time.Duration(0), cfg.Timeout)
}

func TestConfigFields(t *testing.T) {
	cfg := Config{
		Host:    "192.168.1.100",
		Port:    2222,
		User:    "ubuntu",
		KeyPath: "/home/ubuntu/.ssh/id_ed25519",
		Timeout: 10 * time.Second,
	}
	assert.Equal(t, "192.168.1.100", cfg.Host)
	assert.Equal(t, 2222, cfg.Port)
	assert.Equal(t, "ubuntu", cfg.User)
	assert.Equal(t, "/home/ubuntu/.ssh/id_ed25519", cfg.KeyPath)
	assert.Equal(t, 10*time.Second, cfg.Timeout)
}

func TestNewClientNonexistentKey(t *testing.T) {
	cfg := Config{
		Host:    "127.0.0.1",
		Port:    22,
		User:    "test",
		KeyPath: "/tmp/nonexistent_key_ironkube_test_xyz",
	}

	client, err := NewClient(cfg)
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "failed to read key file")
}

func TestNewClientInvalidKey(t *testing.T) {
	keyPath := t.TempDir() + "/bad_key"
	require.NoError(t, os.WriteFile(keyPath, []byte("not a real key"), 0600))

	cfg := Config{
		Host:    "127.0.0.1",
		Port:    22,
		User:    "test",
		KeyPath: keyPath,
	}

	client, err := NewClient(cfg)
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "failed to parse private key")
}

func TestNewClientUnreachableHost(t *testing.T) {
	keyPath := generateTestKey(t)

	cfg := Config{
		Host:    "192.0.2.1",
		Port:    22,
		User:    "test",
		KeyPath: keyPath,
		Timeout: 1 * time.Second,
	}

	client, err := NewClient(cfg)
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "failed to connect")
}

func TestNewClientConnectionRefused(t *testing.T) {
	keyPath := generateTestKey(t)

	cfg := Config{
		Host:    "127.0.0.1",
		Port:    59999,
		User:    "test",
		KeyPath: keyPath,
		Timeout: 2 * time.Second,
	}

	client, err := NewClient(cfg)
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "failed to connect")
}

func TestHostMethod(t *testing.T) {
	c := &Client{host: "myhost.example.com"}
	assert.Equal(t, "myhost.example.com", c.Host())
}

func TestCloseNilConn(t *testing.T) {
	c := &Client{host: "test"}
	assert.NoError(t, c.Close())
}

func generateTestKey(t *testing.T) string {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	pemBlock, err := gossh.MarshalPrivateKey(priv, "")
	require.NoError(t, err)

	pemData := pem.EncodeToMemory(pemBlock)
	require.NotEmpty(t, pemData)

	keyPath := t.TempDir() + "/test_key"
	require.NoError(t, os.WriteFile(keyPath, pemData, 0600))
	return keyPath
}
