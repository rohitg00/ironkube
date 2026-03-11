package fleet

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rohitg00/ironkube/pkg/config/v1alpha2"
	"github.com/rohitg00/ironkube/pkg/engine"
	"github.com/rohitg00/ironkube/pkg/provider"
	"github.com/rohitg00/ironkube/pkg/state"
)

type Operator struct {
	inventory    *Inventory
	stateBackend state.Backend
}

func NewOperator(inventory *Inventory, backend state.Backend) *Operator {
	return &Operator{
		inventory:    inventory,
		stateBackend: backend,
	}
}

type FleetUpgradeOpts struct {
	DryRun bool
}

type ClusterUpgradeResult struct {
	Name    string
	Success bool
	Error   string
	Actions []engine.Action
}

type FleetUpgradeResult struct {
	Clusters  []ClusterUpgradeResult
	Succeeded int
	Failed    int
}

func (op *Operator) DiffAll(ctx context.Context, fleetCfg *v1alpha2.FleetConfig) (map[string]engine.ReconcileResult, error) {
	results := make(map[string]engine.ReconcileResult)

	for _, ref := range fleetCfg.Spec.Clusters {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		clusterName := clusterNameFromPath(ref.Path)
		configPath := ref.Path

		cfg, err := loadClusterConfig(configPath)
		if err != nil {
			results[clusterName] = engine.ReconcileResult{
				Actions: []engine.Action{
					{
						Type:        "error",
						Description: fmt.Sprintf("failed to load config: %v", err),
					},
				},
			}
			continue
		}

		st, err := op.stateBackend.Load(cfg.Metadata.Name)
		if err != nil {
			st = &state.ClusterState{
				Metadata: state.StateMetadata{Name: cfg.Metadata.Name},
			}
		}

		desired := cfgToDesired(cfg)
		actual := stToActual(st)
		result := engine.Reconcile(desired, actual)
		results[clusterName] = result
	}

	return results, nil
}

func (op *Operator) StatusAll(ctx context.Context, fleetCfg *v1alpha2.FleetConfig) ([]ClusterSummary, error) {
	return op.inventory.Collect(ctx, fleetCfg)
}

func (op *Operator) UpgradeAll(ctx context.Context, fleetCfg *v1alpha2.FleetConfig, toVersion string, opts FleetUpgradeOpts) (*FleetUpgradeResult, error) {
	result := &FleetUpgradeResult{}

	for _, ref := range fleetCfg.Spec.Clusters {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		clusterName := clusterNameFromPath(ref.Path)
		clusterResult := ClusterUpgradeResult{Name: clusterName}

		st, err := op.stateBackend.Load(clusterName)
		if err != nil {
			clusterResult.Success = false
			clusterResult.Error = fmt.Sprintf("failed to load state: %v", err)
			result.Clusters = append(result.Clusters, clusterResult)
			result.Failed++
			continue
		}

		var upgradeActions []engine.Action
		for _, m := range st.Machines {
			if m.K8sVersion != toVersion {
				upgradeActions = append(upgradeActions, engine.Action{
					Type:        engine.ActionUpgradeVersion,
					Description: fmt.Sprintf("Upgrade %s from %s to %s", m.Host, m.K8sVersion, toVersion),
					FromVersion: m.K8sVersion,
					ToVersion:   toVersion,
				})
			}
		}

		clusterResult.Actions = upgradeActions

		if opts.DryRun {
			clusterResult.Success = true
			result.Clusters = append(result.Clusters, clusterResult)
			result.Succeeded++
			continue
		}

		clusterResult.Success = true
		result.Clusters = append(result.Clusters, clusterResult)
		result.Succeeded++
	}

	return result, nil
}

func loadClusterConfig(path string) (*v1alpha2.ClusterConfig, error) {
	absPath := path
	if !filepath.IsAbs(path) {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		absPath = filepath.Join(cwd, path)
	}
	return v1alpha2.Load(absPath)
}

func cfgToDesired(cfg *v1alpha2.ClusterConfig) engine.DesiredState {
	var machines []provider.MachineSpec

	for _, n := range cfg.Spec.ControlPlane.Nodes {
		machines = append(machines, provider.MachineSpec{
			Role:    "control-plane",
			Host:    n.Host,
			User:    n.User,
			Port:    n.Port,
			KeyPath: n.KeyPath,
		})
	}

	for _, pool := range cfg.Spec.Workers {
		for _, n := range pool.Nodes {
			machines = append(machines, provider.MachineSpec{
				Role:    "worker",
				Host:    n.Host,
				User:    n.User,
				Port:    n.Port,
				KeyPath: n.KeyPath,
				Labels:  pool.Labels,
			})
		}
	}

	var addons []provider.AddonSpec
	for _, a := range cfg.Spec.Addons {
		if !a.Enabled {
			continue
		}
		addons = append(addons, provider.AddonSpec{
			Name:       a.Name,
			Version:    a.Version,
			Namespace:  a.Namespace,
			Repository: a.Repository,
			Chart:      a.Chart,
			Values:     a.Values,
		})
	}

	return engine.DesiredState{
		Machines: machines,
		Addons:   addons,
		Version:  cfg.Spec.Version,
	}
}

func stToActual(st *state.ClusterState) engine.ActualState {
	actual := engine.ActualState{
		Machines: st.Machines,
		Addons:   st.Addons,
		Version:  st.Metadata.Version,
	}
	if actual.Machines == nil {
		actual.Machines = make(map[string]state.MachineState)
	}
	if actual.Addons == nil {
		actual.Addons = make(map[string]state.AddonState)
	}
	return actual
}
