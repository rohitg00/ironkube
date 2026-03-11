package engine

import (
	"testing"

	"github.com/rohitg00/ironkube/pkg/provider"
	"github.com/rohitg00/ironkube/pkg/state"
	"github.com/stretchr/testify/assert"
)

func TestReconcile_EmptyDesiredEmptyActual(t *testing.T) {
	result := Reconcile(DesiredState{}, ActualState{})

	assert.True(t, result.InSync)
	assert.Empty(t, result.Actions)
}

func TestReconcile_NewMachineNeeded(t *testing.T) {
	desired := DesiredState{
		Machines: []provider.MachineSpec{
			{Host: "10.0.0.1", Role: "control-plane", User: "root", Port: 22},
		},
	}
	actual := ActualState{}

	result := Reconcile(desired, actual)

	assert.False(t, result.InSync)
	assert.Len(t, result.Actions, 1)
	assert.Equal(t, ActionCreateMachine, result.Actions[0].Type)
	assert.Equal(t, "10.0.0.1", result.Actions[0].Machine.Host)
	assert.Equal(t, "control-plane", result.Actions[0].Machine.Role)
}

func TestReconcile_ExtraMachineExists(t *testing.T) {
	desired := DesiredState{}
	actual := ActualState{
		Machines: map[string]state.MachineState{
			"m1": {Host: "10.0.0.1", Role: "worker"},
		},
	}

	result := Reconcile(desired, actual)

	assert.False(t, result.InSync)
	assert.Len(t, result.Actions, 1)
	assert.Equal(t, ActionDeleteMachine, result.Actions[0].Type)
	assert.Equal(t, "10.0.0.1", result.Actions[0].Machine.Host)
}

func TestReconcile_MachinesMatch(t *testing.T) {
	desired := DesiredState{
		Machines: []provider.MachineSpec{
			{Host: "10.0.0.1", Role: "control-plane"},
		},
	}
	actual := ActualState{
		Machines: map[string]state.MachineState{
			"m1": {Host: "10.0.0.1", Role: "control-plane"},
		},
	}

	result := Reconcile(desired, actual)

	assert.True(t, result.InSync)
	assert.Empty(t, result.Actions)
}

func TestReconcile_MultipleMachineChanges(t *testing.T) {
	desired := DesiredState{
		Machines: []provider.MachineSpec{
			{Host: "10.0.0.1", Role: "control-plane"},
			{Host: "10.0.0.3", Role: "worker"},
		},
	}
	actual := ActualState{
		Machines: map[string]state.MachineState{
			"m1": {Host: "10.0.0.1", Role: "control-plane"},
			"m2": {Host: "10.0.0.2", Role: "worker"},
		},
	}

	result := Reconcile(desired, actual)

	assert.False(t, result.InSync)

	var creates, deletes int
	for _, a := range result.Actions {
		switch a.Type {
		case ActionCreateMachine:
			creates++
			assert.Equal(t, "10.0.0.3", a.Machine.Host)
		case ActionDeleteMachine:
			deletes++
			assert.Equal(t, "10.0.0.2", a.Machine.Host)
		}
	}
	assert.Equal(t, 1, creates)
	assert.Equal(t, 1, deletes)
}

func TestReconcile_VersionUpgradeNeeded(t *testing.T) {
	desired := DesiredState{
		Version: "1.30.0",
	}
	actual := ActualState{
		Version: "1.29.0",
		Machines: map[string]state.MachineState{
			"m1": {Host: "10.0.0.1", Role: "control-plane"},
			"m2": {Host: "10.0.0.2", Role: "worker"},
		},
	}

	result := Reconcile(desired, actual)

	assert.False(t, result.InSync)

	var upgrades int
	for _, a := range result.Actions {
		if a.Type == ActionUpgradeVersion {
			upgrades++
			assert.Equal(t, "1.29.0", a.FromVersion)
			assert.Equal(t, "1.30.0", a.ToVersion)
		}
	}
	assert.Equal(t, 2, upgrades)
}

func TestReconcile_SameVersionNoUpgrade(t *testing.T) {
	desired := DesiredState{
		Version: "1.30.0",
		Machines: []provider.MachineSpec{
			{Host: "10.0.0.1", Role: "control-plane"},
		},
	}
	actual := ActualState{
		Version: "1.30.0",
		Machines: map[string]state.MachineState{
			"m1": {Host: "10.0.0.1", Role: "control-plane"},
		},
	}

	result := Reconcile(desired, actual)

	assert.True(t, result.InSync)
	assert.Empty(t, result.Actions)
}

func TestReconcile_NewAddonNeeded(t *testing.T) {
	desired := DesiredState{
		Addons: []provider.AddonSpec{
			{Name: "cilium", Version: "1.15.0", Namespace: "kube-system"},
		},
	}
	actual := ActualState{}

	result := Reconcile(desired, actual)

	assert.False(t, result.InSync)
	assert.Len(t, result.Actions, 1)
	assert.Equal(t, ActionInstallAddon, result.Actions[0].Type)
	assert.Equal(t, "cilium", result.Actions[0].Addon.Name)
	assert.Equal(t, "1.15.0", result.Actions[0].Addon.Version)
}

func TestReconcile_AddonRemoved(t *testing.T) {
	desired := DesiredState{}
	actual := ActualState{
		Addons: map[string]state.AddonState{
			"flannel": {Name: "flannel", Version: "0.24.0", Installed: true},
		},
	}

	result := Reconcile(desired, actual)

	assert.False(t, result.InSync)
	assert.Len(t, result.Actions, 1)
	assert.Equal(t, ActionRemoveAddon, result.Actions[0].Type)
	assert.Equal(t, "flannel", result.Actions[0].Addon.Name)
}

func TestReconcile_AddonVersionChanged(t *testing.T) {
	desired := DesiredState{
		Addons: []provider.AddonSpec{
			{Name: "cilium", Version: "1.16.0", Namespace: "kube-system"},
		},
	}
	actual := ActualState{
		Addons: map[string]state.AddonState{
			"cilium": {Name: "cilium", Version: "1.15.0", Installed: true},
		},
	}

	result := Reconcile(desired, actual)

	assert.False(t, result.InSync)
	assert.Len(t, result.Actions, 1)
	assert.Equal(t, ActionUpgradeAddon, result.Actions[0].Type)
	assert.Equal(t, "1.15.0", result.Actions[0].FromVersion)
	assert.Equal(t, "1.16.0", result.Actions[0].ToVersion)
}

func TestReconcile_AddonSameVersion(t *testing.T) {
	desired := DesiredState{
		Addons: []provider.AddonSpec{
			{Name: "cilium", Version: "1.15.0", Namespace: "kube-system"},
		},
	}
	actual := ActualState{
		Addons: map[string]state.AddonState{
			"cilium": {Name: "cilium", Version: "1.15.0", Installed: true},
		},
	}

	result := Reconcile(desired, actual)

	assert.True(t, result.InSync)
	assert.Empty(t, result.Actions)
}

func TestReconcile_MixedChanges(t *testing.T) {
	desired := DesiredState{
		Version: "1.30.0",
		Machines: []provider.MachineSpec{
			{Host: "10.0.0.1", Role: "control-plane"},
			{Host: "10.0.0.3", Role: "worker"},
		},
		Addons: []provider.AddonSpec{
			{Name: "cilium", Version: "1.16.0", Namespace: "kube-system"},
			{Name: "metrics-server", Version: "0.7.0", Namespace: "kube-system"},
		},
	}
	actual := ActualState{
		Version: "1.29.0",
		Machines: map[string]state.MachineState{
			"m1": {Host: "10.0.0.1", Role: "control-plane"},
			"m2": {Host: "10.0.0.2", Role: "worker"},
		},
		Addons: map[string]state.AddonState{
			"cilium":  {Name: "cilium", Version: "1.15.0", Installed: true},
			"flannel": {Name: "flannel", Version: "0.24.0", Installed: true},
		},
	}

	result := Reconcile(desired, actual)

	assert.False(t, result.InSync)

	actionTypes := make(map[ActionType]int)
	for _, a := range result.Actions {
		actionTypes[a.Type]++
	}

	assert.Equal(t, 1, actionTypes[ActionCreateMachine])
	assert.Equal(t, 1, actionTypes[ActionDeleteMachine])
	assert.Equal(t, 2, actionTypes[ActionUpgradeVersion])
	assert.Equal(t, 1, actionTypes[ActionInstallAddon])
	assert.Equal(t, 1, actionTypes[ActionUpgradeAddon])
	assert.Equal(t, 1, actionTypes[ActionRemoveAddon])
}

func TestReconcile_AllInSync(t *testing.T) {
	desired := DesiredState{
		Version: "1.30.0",
		Machines: []provider.MachineSpec{
			{Host: "10.0.0.1", Role: "control-plane"},
			{Host: "10.0.0.2", Role: "worker"},
		},
		Addons: []provider.AddonSpec{
			{Name: "cilium", Version: "1.15.0", Namespace: "kube-system"},
		},
	}
	actual := ActualState{
		Version: "1.30.0",
		Machines: map[string]state.MachineState{
			"m1": {Host: "10.0.0.1", Role: "control-plane"},
			"m2": {Host: "10.0.0.2", Role: "worker"},
		},
		Addons: map[string]state.AddonState{
			"cilium": {Name: "cilium", Version: "1.15.0", Installed: true},
		},
	}

	result := Reconcile(desired, actual)

	assert.True(t, result.InSync)
	assert.Empty(t, result.Actions)
}

func TestReconcile_AddonNotInstalledTreatedAsMissing(t *testing.T) {
	desired := DesiredState{
		Addons: []provider.AddonSpec{
			{Name: "cilium", Version: "1.15.0", Namespace: "kube-system"},
		},
	}
	actual := ActualState{
		Addons: map[string]state.AddonState{
			"cilium": {Name: "cilium", Version: "1.15.0", Installed: false},
		},
	}

	result := Reconcile(desired, actual)

	assert.False(t, result.InSync)
	assert.Len(t, result.Actions, 1)
	assert.Equal(t, ActionInstallAddon, result.Actions[0].Type)
}

func TestReconcile_EmptyDesiredVersionSkipsUpgrade(t *testing.T) {
	desired := DesiredState{
		Machines: []provider.MachineSpec{
			{Host: "10.0.0.1", Role: "control-plane"},
		},
	}
	actual := ActualState{
		Version: "1.29.0",
		Machines: map[string]state.MachineState{
			"m1": {Host: "10.0.0.1", Role: "control-plane"},
		},
	}

	result := Reconcile(desired, actual)

	assert.True(t, result.InSync)
	for _, a := range result.Actions {
		assert.NotEqual(t, ActionUpgradeVersion, a.Type)
	}
}

func TestReconcile_VersionUpgradePerMachine(t *testing.T) {
	desired := DesiredState{
		Version: "1.31.0",
	}
	actual := ActualState{
		Version: "1.30.0",
		Machines: map[string]state.MachineState{
			"m1": {Host: "10.0.0.1", Role: "control-plane"},
			"m2": {Host: "10.0.0.2", Role: "worker"},
			"m3": {Host: "10.0.0.3", Role: "worker"},
		},
	}

	result := Reconcile(desired, actual)

	var upgradeActions []Action
	for _, a := range result.Actions {
		if a.Type == ActionUpgradeVersion {
			upgradeActions = append(upgradeActions, a)
		}
	}
	assert.Len(t, upgradeActions, 3)
}

func TestReconcile_CreateMachineHasCorrectDescription(t *testing.T) {
	desired := DesiredState{
		Machines: []provider.MachineSpec{
			{Host: "10.0.0.1", Role: "control-plane"},
		},
	}

	result := Reconcile(desired, ActualState{})

	assert.Contains(t, result.Actions[0].Description, "10.0.0.1")
	assert.Contains(t, result.Actions[0].Description, "control-plane")
}

func TestReconcile_MultipleAddonsInstalled(t *testing.T) {
	desired := DesiredState{
		Addons: []provider.AddonSpec{
			{Name: "cilium", Version: "1.15.0"},
			{Name: "metrics-server", Version: "0.7.0"},
			{Name: "cert-manager", Version: "1.14.0"},
		},
	}
	actual := ActualState{}

	result := Reconcile(desired, actual)

	assert.Len(t, result.Actions, 3)
	for _, a := range result.Actions {
		assert.Equal(t, ActionInstallAddon, a.Type)
	}
}
