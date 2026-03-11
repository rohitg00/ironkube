package phases

import (
	"context"
	"testing"

	"github.com/rohitg00/ironkube/pkg/config"
	"github.com/rohitg00/ironkube/pkg/engine"
	"github.com/rohitg00/ironkube/pkg/ssh"
	"github.com/stretchr/testify/assert"
)

func TestHealthCheckPhaseName(t *testing.T) {
	p := &HealthCheckPhase{Config: &config.ClusterConfig{}}
	assert.Equal(t, "health-check", p.Name())
}

func TestHealthCheckPhaseMissingExecutor(t *testing.T) {
	p := &HealthCheckPhase{Config: validTestConfig()}
	state := engine.NewState()
	err := p.Run(context.Background(), state)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "executor not found")
}

func TestHealthCheckPhaseMissingDistroPlugin(t *testing.T) {
	p := &HealthCheckPhase{Config: validTestConfig()}
	state := engine.NewState()
	state.Set("executor", (*ssh.Executor)(nil))
	err := p.Run(context.Background(), state)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "distro_plugin not found")
}
