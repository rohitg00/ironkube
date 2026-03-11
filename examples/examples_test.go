package examples

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/rohitg00/ironkube/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExampleConfigs(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	examples := []struct {
		file    string
		name    string
		distro  string
		cpNodes int
	}{
		{"single-node-k3s.yaml", "dev-cluster", "k3s", 1},
		{"ha-kubeadm.yaml", "production-cluster", "kubeadm", 3},
	}

	for _, ex := range examples {
		t.Run(ex.file, func(t *testing.T) {
			cfg, err := config.Load(filepath.Join(dir, ex.file))
			require.NoError(t, err)
			require.NoError(t, config.Validate(cfg))
			assert.Equal(t, ex.name, cfg.Metadata.Name)
			assert.Equal(t, ex.distro, cfg.Spec.Distro)
			assert.Len(t, cfg.Spec.ControlPlane.Nodes, ex.cpNodes)
		})
	}
}
