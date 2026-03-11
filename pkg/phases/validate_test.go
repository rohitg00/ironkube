package phases

import (
	"context"
	"testing"

	"github.com/rohitg00/ironkube/pkg/config"
	"github.com/rohitg00/ironkube/pkg/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validTestConfig() *config.ClusterConfig {
	return &config.ClusterConfig{
		APIVersion: "ironkube.io/v1alpha1",
		Kind:       "Cluster",
		Metadata:   config.Metadata{Name: "test"},
		Spec: config.ClusterSpec{
			Distro:  "k3s",
			Version: "v1.34.3+k3s1",
			ControlPlane: config.ControlPlane{
				Replicas: 1,
				Nodes:    []config.Node{{Host: "10.0.0.1", User: "root", Port: 22}},
			},
			Security:   config.Security{Profile: "minimal"},
			Networking: config.Networking{CNI: "flannel", PodCIDR: "10.42.0.0/16", SvcCIDR: "10.43.0.0/16"},
		},
	}
}

func TestValidatePhaseSuccess(t *testing.T) {
	phase := &ValidatePhase{Config: validTestConfig()}
	state := engine.NewState()
	err := phase.Run(context.Background(), state)
	require.NoError(t, err)

	plugin, ok := state.Get("distro_plugin")
	assert.True(t, ok)
	assert.NotNil(t, plugin)
}

func TestValidatePhaseInvalidDistro(t *testing.T) {
	cfg := validTestConfig()
	cfg.Spec.Distro = "invalid"
	phase := &ValidatePhase{Config: cfg}
	err := phase.Run(context.Background(), engine.NewState())
	assert.Error(t, err)
}

func TestValidatePhaseBadVersion(t *testing.T) {
	cfg := validTestConfig()
	cfg.Spec.Version = "bad"
	phase := &ValidatePhase{Config: cfg}
	err := phase.Run(context.Background(), engine.NewState())
	assert.Error(t, err)
}

func TestValidatePhaseBadProfile(t *testing.T) {
	cfg := validTestConfig()
	cfg.Spec.Security.Profile = "ultra"
	phase := &ValidatePhase{Config: cfg}
	err := phase.Run(context.Background(), engine.NewState())
	assert.Error(t, err)
}

func TestValidatePhaseName(t *testing.T) {
	phase := &ValidatePhase{}
	assert.Equal(t, "validate", phase.Name())
}

func TestValidatePhaseKubeadm(t *testing.T) {
	cfg := validTestConfig()
	cfg.Spec.Distro = "kubeadm"
	cfg.Spec.Version = "v1.34.3"
	phase := &ValidatePhase{Config: cfg}
	err := phase.Run(context.Background(), engine.NewState())
	require.NoError(t, err)
}
