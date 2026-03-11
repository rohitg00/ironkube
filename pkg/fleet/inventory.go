package fleet

import (
	"context"
	"fmt"
	"os"

	"github.com/rohitg00/ironkube/pkg/config/v1alpha2"
	"github.com/rohitg00/ironkube/pkg/state"
	"sigs.k8s.io/yaml"
)

type Inventory struct {
	stateBackend state.Backend
}

func NewInventory(backend state.Backend) *Inventory {
	return &Inventory{stateBackend: backend}
}

type ClusterSummary struct {
	Name      string
	Labels    map[string]string
	State     *state.ClusterState
	Available bool
	Error     string
}

type FleetSummary struct {
	TotalClusters     int
	AvailableClusters int
	TotalMachines     int
	ReadyMachines     int
	Versions          map[string]int
}

func LoadFleetConfig(path string) (*v1alpha2.FleetConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading fleet config: %w", err)
	}

	var cfg v1alpha2.FleetConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing fleet config YAML: %w", err)
	}

	if cfg.Kind != "Fleet" {
		return nil, fmt.Errorf("expected kind Fleet, got %q", cfg.Kind)
	}

	if cfg.Metadata.Name == "" {
		return nil, fmt.Errorf("fleet metadata.name is required")
	}

	if len(cfg.Spec.Clusters) == 0 {
		return nil, fmt.Errorf("fleet must have at least one cluster reference")
	}

	for i, ref := range cfg.Spec.Clusters {
		if ref.Path == "" {
			return nil, fmt.Errorf("clusters[%d].path is required", i)
		}
	}

	return &cfg, nil
}

func (inv *Inventory) Collect(ctx context.Context, fleetCfg *v1alpha2.FleetConfig) ([]ClusterSummary, error) {
	var summaries []ClusterSummary

	for _, ref := range fleetCfg.Spec.Clusters {
		select {
		case <-ctx.Done():
			return summaries, ctx.Err()
		default:
		}

		clusterName := clusterNameFromPath(ref.Path)
		summary := ClusterSummary{
			Name:   clusterName,
			Labels: ref.Labels,
		}

		st, err := inv.stateBackend.Load(clusterName)
		if err != nil {
			summary.Available = false
			summary.Error = fmt.Sprintf("failed to load state: %v", err)
		} else {
			summary.State = st
			summary.Available = true
		}

		summaries = append(summaries, summary)
	}

	return summaries, nil
}

func FilterByLabel(summaries []ClusterSummary, key, value string) []ClusterSummary {
	var filtered []ClusterSummary
	for _, s := range summaries {
		if s.Labels != nil && s.Labels[key] == value {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

func Summary(summaries []ClusterSummary) FleetSummary {
	fs := FleetSummary{
		TotalClusters: len(summaries),
		Versions:      make(map[string]int),
	}

	for _, s := range summaries {
		if !s.Available {
			continue
		}
		fs.AvailableClusters++

		if s.State == nil {
			continue
		}

		for _, m := range s.State.Machines {
			fs.TotalMachines++
			if m.Status == "ready" {
				fs.ReadyMachines++
			}
		}

		if s.State.Metadata.Version != "" {
			fs.Versions[s.State.Metadata.Version]++
		}
	}

	return fs
}

func clusterNameFromPath(path string) string {
	base := path
	for i := len(base) - 1; i >= 0; i-- {
		if base[i] == '/' || base[i] == '\\' {
			base = base[i+1:]
			break
		}
	}

	if len(base) > 5 && base[len(base)-5:] == ".yaml" {
		return base[:len(base)-5]
	}
	if len(base) > 4 && base[len(base)-4:] == ".yml" {
		return base[:len(base)-4]
	}
	return base
}
