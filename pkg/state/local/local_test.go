package local

import (
	"path/filepath"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/rohitg00/ironkube/pkg/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestState(name string) *state.ClusterState {
	now := time.Now().Truncate(time.Second)
	return &state.ClusterState{
		Metadata: state.StateMetadata{
			Name:      name,
			CreatedAt: now,
			UpdatedAt: now,
			Version:   "v0.0.1",
		},
		LastOperation: state.OperationRecord{
			Type:      state.OpApply,
			Status:    state.OpStatusSuccess,
			StartedAt: now,
		},
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	b := New(dir)

	original := newTestState("cluster-1")
	original.SetMachine("node-1", state.MachineState{
		ID:   "node-1",
		Host: "10.0.0.1",
		Role: "control-plane",
	})
	original.SetAddon("cilium", state.AddonState{
		Name:      "cilium",
		Version:   "1.15.0",
		Installed: true,
		Ready:     true,
	})

	err := b.Save(original)
	require.NoError(t, err)

	loaded, err := b.Load("cluster-1")
	require.NoError(t, err)

	assert.Equal(t, original.Metadata.Name, loaded.Metadata.Name)
	assert.Equal(t, original.Metadata.Version, loaded.Metadata.Version)
	assert.Equal(t, original.LastOperation.Type, loaded.LastOperation.Type)
	assert.Equal(t, original.Machines["node-1"].Host, loaded.Machines["node-1"].Host)
	assert.Equal(t, original.Addons["cilium"].Version, loaded.Addons["cilium"].Version)
}

func TestLoadNotFound(t *testing.T) {
	dir := t.TempDir()
	b := New(dir)

	_, err := b.Load("nonexistent")
	require.Error(t, err)
}

func TestList(t *testing.T) {
	dir := t.TempDir()
	b := New(dir)

	require.NoError(t, b.Save(newTestState("alpha")))
	require.NoError(t, b.Save(newTestState("beta")))

	names, err := b.List()
	require.NoError(t, err)

	sort.Strings(names)
	assert.Equal(t, []string{"alpha", "beta"}, names)
}

func TestDelete(t *testing.T) {
	dir := t.TempDir()
	b := New(dir)

	require.NoError(t, b.Save(newTestState("doomed")))

	_, err := b.Load("doomed")
	require.NoError(t, err)

	require.NoError(t, b.Delete("doomed"))

	_, err = b.Load("doomed")
	require.Error(t, err)
}

func TestLocking(t *testing.T) {
	dir := t.TempDir()
	b := New(dir)

	unlock, err := b.Lock("test-cluster")
	require.NoError(t, err)
	require.NotNil(t, unlock)

	_, err = b.Lock("test-cluster")
	require.Error(t, err)

	unlock()

	unlock2, err := b.Lock("test-cluster")
	require.NoError(t, err)
	require.NotNil(t, unlock2)
	unlock2()
}

func TestCreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "subdir")
	b := New(dir)

	err := b.Save(newTestState("auto-dir"))
	require.NoError(t, err)

	loaded, err := b.Load("auto-dir")
	require.NoError(t, err)
	assert.Equal(t, "auto-dir", loaded.Metadata.Name)
}

func TestSaveOverwrite(t *testing.T) {
	dir := t.TempDir()
	b := New(dir)

	s := newTestState("overwrite-me")
	s.Metadata.Version = "v1"
	require.NoError(t, b.Save(s))

	s.Metadata.Version = "v2"
	require.NoError(t, b.Save(s))

	loaded, err := b.Load("overwrite-me")
	require.NoError(t, err)
	assert.Equal(t, "v2", loaded.Metadata.Version)
}

func TestConcurrentAccess(t *testing.T) {
	dir := t.TempDir()
	b := New(dir)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s := newTestState("concurrent")
			_ = b.Save(s)
			_, _ = b.Load("concurrent")
		}()
	}
	wg.Wait()

	loaded, err := b.Load("concurrent")
	require.NoError(t, err)
	assert.Equal(t, "concurrent", loaded.Metadata.Name)
}
