package engine

import (
	"fmt"

	"github.com/rohitg00/ironkube/pkg/provider"
	"github.com/rohitg00/ironkube/pkg/state"
)

type ActionType string

const (
	ActionCreateMachine ActionType = "create-machine"
	ActionDeleteMachine ActionType = "delete-machine"
	ActionUpgradeVersion ActionType = "upgrade-version"
	ActionInstallAddon  ActionType = "install-addon"
	ActionUpgradeAddon  ActionType = "upgrade-addon"
	ActionRemoveAddon   ActionType = "remove-addon"
)

type Action struct {
	Type        ActionType
	Description string
	Machine     *provider.MachineSpec
	Addon       *provider.AddonSpec
	FromVersion string
	ToVersion   string
}

type ReconcileResult struct {
	Actions []Action
	InSync  bool
}

type DesiredState struct {
	Machines []provider.MachineSpec
	Addons   []provider.AddonSpec
	Version  string
}

type ActualState struct {
	Machines map[string]state.MachineState
	Addons   map[string]state.AddonState
	Version  string
}

func Reconcile(desired DesiredState, actual ActualState) ReconcileResult {
	var actions []Action

	actions = append(actions, reconcileMachines(desired, actual)...)
	actions = append(actions, reconcileVersion(desired, actual)...)
	actions = append(actions, reconcileAddons(desired, actual)...)

	return ReconcileResult{
		Actions: actions,
		InSync:  len(actions) == 0,
	}
}

func reconcileMachines(desired DesiredState, actual ActualState) []Action {
	var actions []Action

	desiredHosts := make(map[string]provider.MachineSpec, len(desired.Machines))
	for _, m := range desired.Machines {
		desiredHosts[m.Host] = m
	}

	actualHosts := make(map[string]state.MachineState, len(actual.Machines))
	for _, m := range actual.Machines {
		actualHosts[m.Host] = m
	}

	for host, spec := range desiredHosts {
		if _, exists := actualHosts[host]; !exists {
			s := spec
			actions = append(actions, Action{
				Type:        ActionCreateMachine,
				Description: fmt.Sprintf("Create %s machine %s", spec.Role, host),
				Machine:     &s,
			})
		}
	}

	for host, m := range actualHosts {
		if _, exists := desiredHosts[host]; !exists {
			actions = append(actions, Action{
				Type:        ActionDeleteMachine,
				Description: fmt.Sprintf("Delete %s machine %s", m.Role, host),
				Machine: &provider.MachineSpec{
					Host: host,
					Role: m.Role,
				},
			})
		}
	}

	return actions
}

func reconcileVersion(desired DesiredState, actual ActualState) []Action {
	var actions []Action

	if desired.Version == "" || desired.Version == actual.Version {
		return actions
	}

	for _, m := range actual.Machines {
		actions = append(actions, Action{
			Type:        ActionUpgradeVersion,
			Description: fmt.Sprintf("Upgrade machine %s from %s to %s", m.Host, actual.Version, desired.Version),
			Machine: &provider.MachineSpec{
				Host: m.Host,
				Role: m.Role,
			},
			FromVersion: actual.Version,
			ToVersion:   desired.Version,
		})
	}

	return actions
}

func reconcileAddons(desired DesiredState, actual ActualState) []Action {
	var actions []Action

	desiredAddons := make(map[string]provider.AddonSpec, len(desired.Addons))
	for _, a := range desired.Addons {
		desiredAddons[a.Name] = a
	}

	for name, spec := range desiredAddons {
		actualAddon, exists := actual.Addons[name]
		if !exists || !actualAddon.Installed {
			s := spec
			actions = append(actions, Action{
				Type:        ActionInstallAddon,
				Description: fmt.Sprintf("Install addon %s %s", name, spec.Version),
				Addon:       &s,
			})
			continue
		}
		if actualAddon.Version != spec.Version {
			s := spec
			actions = append(actions, Action{
				Type:        ActionUpgradeAddon,
				Description: fmt.Sprintf("Upgrade addon %s from %s to %s", name, actualAddon.Version, spec.Version),
				Addon:       &s,
				FromVersion: actualAddon.Version,
				ToVersion:   spec.Version,
			})
		}
	}

	for name := range actual.Addons {
		if _, exists := desiredAddons[name]; !exists {
			actions = append(actions, Action{
				Type:        ActionRemoveAddon,
				Description: fmt.Sprintf("Remove addon %s", name),
				Addon: &provider.AddonSpec{
					Name: name,
				},
			})
		}
	}

	return actions
}
