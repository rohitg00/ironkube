package phases

import (
	"context"
	"testing"

	"github.com/rohitg00/ironkube/pkg/config"
	"github.com/rohitg00/ironkube/pkg/engine"
	"github.com/rohitg00/ironkube/pkg/ssh"
	"github.com/stretchr/testify/assert"
)

func TestDestroyPhaseName(t *testing.T) {
	p := &DestroyPhase{Config: &config.ClusterConfig{}}
	assert.Equal(t, "destroy", p.Name())
}

func TestDestroyPhaseMissingExecutor(t *testing.T) {
	p := &DestroyPhase{Config: validTestConfig()}
	state := engine.NewState()
	err := p.Run(context.Background(), state)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "executor not found")
}

func TestDestroyPhaseMissingDistroPlugin(t *testing.T) {
	p := &DestroyPhase{Config: validTestConfig()}
	state := engine.NewState()
	state.Set("executor", (*ssh.Executor)(nil))
	err := p.Run(context.Background(), state)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "distro_plugin not found")
}
