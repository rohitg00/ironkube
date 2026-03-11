package addons

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	tests := []struct {
		name  string
		found bool
	}{
		{"cilium", true},
		{"calico", true},
		{"cert-manager", true},
		{"monitoring", true},
		{"flannel", false},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addon, ok := Get(tt.name)
			assert.Equal(t, tt.found, ok)
			if ok {
				assert.Equal(t, tt.name, addon.Name())
				assert.NotEmpty(t, addon.InstallCmd())
				assert.NotEmpty(t, addon.WaitCmd())
			}
		})
	}
}

func TestCilium(t *testing.T) {
	c := &Cilium{}
	assert.Equal(t, "cilium", c.Name())
	assert.Contains(t, c.InstallCmd(), "cilium")
	assert.Contains(t, c.WaitCmd(), "cilium-agent")
}

func TestCalico(t *testing.T) {
	c := &Calico{}
	assert.Equal(t, "calico", c.Name())
	assert.Contains(t, c.InstallCmd(), "calico")
	assert.Contains(t, c.WaitCmd(), "calico-node")
}

func TestCertManager(t *testing.T) {
	c := &CertManager{}
	assert.Equal(t, "cert-manager", c.Name())
	assert.Contains(t, c.InstallCmd(), "cert-manager")
	assert.Contains(t, c.WaitCmd(), "cert-manager")
}

func TestMonitoring(t *testing.T) {
	m := &Monitoring{}
	assert.Equal(t, "monitoring", m.Name())
	assert.Contains(t, m.InstallCmd(), "prometheus")
	assert.Contains(t, m.WaitCmd(), "prometheus")
}
