package v1alpha2

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

const fleetYAML = `
apiVersion: ironkube.dev/v1alpha2
kind: Fleet
metadata:
  name: production-fleet
  labels:
    env: production
spec:
  clusters:
    - path: clusters/eu-west.yaml
      labels:
        region: eu-west-1
    - path: clusters/us-east.yaml
      labels:
        region: us-east-1
    - path: clusters/ap-south.yaml
      labels:
        region: ap-south-1
  state:
    backend: s3
    s3:
      bucket: fleet-state
      region: us-east-1
      prefix: fleet/
  policies:
    upgradeWindow: "Sat 02:00-06:00 UTC"
    maxConcurrentUpgrades: 1
`

func TestFleetConfigParsing(t *testing.T) {
	var cfg FleetConfig
	err := yaml.Unmarshal([]byte(fleetYAML), &cfg)
	require.NoError(t, err)

	assert.Equal(t, "ironkube.dev/v1alpha2", cfg.APIVersion)
	assert.Equal(t, "Fleet", cfg.Kind)
	assert.Equal(t, "production-fleet", cfg.Metadata.Name)
	assert.Equal(t, "production", cfg.Metadata.Labels["env"])
	assert.Len(t, cfg.Spec.Clusters, 3)
	assert.Equal(t, "clusters/eu-west.yaml", cfg.Spec.Clusters[0].Path)
	assert.Equal(t, "eu-west-1", cfg.Spec.Clusters[0].Labels["region"])
	assert.Equal(t, "s3", cfg.Spec.State.Backend)
	require.NotNil(t, cfg.Spec.State.S3)
	assert.Equal(t, "fleet-state", cfg.Spec.State.S3.Bucket)
	assert.Equal(t, "Sat 02:00-06:00 UTC", cfg.Spec.Policies.UpgradeWindow)
	assert.Equal(t, 1, cfg.Spec.Policies.MaxConcurrentUpgrades)
}

func TestFleetConfigConstruction(t *testing.T) {
	cfg := FleetConfig{
		APIVersion: "ironkube.dev/v1alpha2",
		Kind:       "Fleet",
		Metadata:   Metadata{Name: "test-fleet"},
		Spec: FleetSpec{
			Clusters: []ClusterRef{
				{Path: "cluster-a.yaml"},
				{Path: "cluster-b.yaml"},
			},
			Policies: FleetPolicies{
				MaxConcurrentUpgrades: 2,
			},
		},
	}
	assert.Equal(t, "test-fleet", cfg.Metadata.Name)
	assert.Len(t, cfg.Spec.Clusters, 2)
	assert.Equal(t, 2, cfg.Spec.Policies.MaxConcurrentUpgrades)
}

func TestFleetConfigYAMLRoundTrip(t *testing.T) {
	original := FleetConfig{
		APIVersion: "ironkube.dev/v1alpha2",
		Kind:       "Fleet",
		Metadata:   Metadata{Name: "roundtrip"},
		Spec: FleetSpec{
			Clusters: []ClusterRef{
				{Path: "a.yaml", Labels: map[string]string{"tier": "prod"}},
			},
			State: StateConfig{Backend: "local"},
		},
	}

	data, err := yaml.Marshal(original)
	require.NoError(t, err)

	var parsed FleetConfig
	err = yaml.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, original.Metadata.Name, parsed.Metadata.Name)
	assert.Equal(t, original.Spec.Clusters[0].Path, parsed.Spec.Clusters[0].Path)
	assert.Equal(t, "prod", parsed.Spec.Clusters[0].Labels["tier"])
	assert.Equal(t, "local", parsed.Spec.State.Backend)
}

func TestFleetClusterRefValidation(t *testing.T) {
	tests := []struct {
		name    string
		refs    []ClusterRef
		wantErr bool
	}{
		{
			name:    "valid refs",
			refs:    []ClusterRef{{Path: "a.yaml"}, {Path: "b.yaml"}},
			wantErr: false,
		},
		{
			name:    "empty path",
			refs:    []ClusterRef{{Path: ""}},
			wantErr: true,
		},
		{
			name:    "mixed valid and empty",
			refs:    []ClusterRef{{Path: "valid.yaml"}, {Path: ""}},
			wantErr: true,
		},
		{
			name:    "empty cluster list",
			refs:    []ClusterRef{},
			wantErr: true,
		},
		{
			name:    "nil cluster list",
			refs:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateClusterRefs(tt.refs)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFleetEmptyPolicies(t *testing.T) {
	yamlStr := `
apiVersion: ironkube.dev/v1alpha2
kind: Fleet
metadata:
  name: minimal-fleet
spec:
  clusters:
    - path: cluster.yaml
`
	var cfg FleetConfig
	err := yaml.Unmarshal([]byte(yamlStr), &cfg)
	require.NoError(t, err)

	assert.Equal(t, "", cfg.Spec.Policies.UpgradeWindow)
	assert.Equal(t, 0, cfg.Spec.Policies.MaxConcurrentUpgrades)
}

func TestFleetClusterRefWithLabels(t *testing.T) {
	ref := ClusterRef{
		Path: "clusters/staging.yaml",
		Labels: map[string]string{
			"env":    "staging",
			"region": "us-west-2",
			"tier":   "non-prod",
		},
	}
	assert.Equal(t, "clusters/staging.yaml", ref.Path)
	assert.Len(t, ref.Labels, 3)
	assert.Equal(t, "staging", ref.Labels["env"])
}

func validateClusterRefs(refs []ClusterRef) error {
	if len(refs) == 0 {
		return fmt.Errorf("fleet must have at least one cluster reference")
	}
	for i, ref := range refs {
		if ref.Path == "" {
			return fmt.Errorf("clusters[%d].path is required", i)
		}
	}
	return nil
}
