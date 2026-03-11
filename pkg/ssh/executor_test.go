package ssh

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewExecutor(t *testing.T) {
	e := NewExecutor()
	require.NotNil(t, e)
	assert.NotNil(t, e.clients)
	assert.Empty(t, e.clients)
}

func TestRunOnHostNotConnected(t *testing.T) {
	e := NewExecutor()

	output, err := e.RunOnHost("unknown-host", "echo hello")
	assert.Empty(t, output)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no connection")
}

func TestRunOnAllEmpty(t *testing.T) {
	e := NewExecutor()
	results := e.RunOnAll(context.Background(), "echo hello")
	assert.Empty(t, results)
}

func TestRunOnHostsWithCancelledContext(t *testing.T) {
	e := NewExecutor()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	hosts := []string{"host1", "host2", "host3"}
	results := e.RunOnHosts(ctx, hosts, "echo hello")

	assert.Len(t, results, 3)
	for _, r := range results {
		require.Error(t, r.Err)
	}
}

func TestRunOnHostsEmptyHosts(t *testing.T) {
	e := NewExecutor()
	results := e.RunOnHosts(context.Background(), []string{}, "echo hello")
	assert.Empty(t, results)
}

func TestCloseEmpty(t *testing.T) {
	e := NewExecutor()
	assert.NotPanics(t, func() {
		e.Close()
	})
}

func TestCloseMultipleTimes(t *testing.T) {
	e := NewExecutor()
	assert.NotPanics(t, func() {
		e.Close()
		e.Close()
	})
}

func TestConnectWithBadConfig(t *testing.T) {
	e := NewExecutor()
	err := e.Connect(Config{
		Host:    "127.0.0.1",
		Port:    22,
		User:    "test",
		KeyPath: "/tmp/nonexistent_ironkube_key_abc",
	})
	require.Error(t, err)
}
