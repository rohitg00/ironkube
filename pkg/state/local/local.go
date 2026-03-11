package local

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rohitg00/ironkube/pkg/state"
)

type LocalBackend struct {
	dir   string
	mu    sync.Mutex
	locks map[string]bool
}

func New(dir string) *LocalBackend {
	return &LocalBackend{
		dir:   dir,
		locks: make(map[string]bool),
	}
}

func (b *LocalBackend) filePath(name string) string {
	return filepath.Join(b.dir, name+".json")
}

func (b *LocalBackend) Save(s *state.ClusterState) error {
	if err := os.MkdirAll(b.dir, 0700); err != nil {
		return fmt.Errorf("creating state directory: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	if err := os.WriteFile(b.filePath(s.Metadata.Name), data, 0600); err != nil {
		return fmt.Errorf("writing state file: %w", err)
	}

	return nil
}

func (b *LocalBackend) Load(clusterName string) (*state.ClusterState, error) {
	data, err := os.ReadFile(b.filePath(clusterName))
	if err != nil {
		return nil, fmt.Errorf("reading state file: %w", err)
	}

	var s state.ClusterState
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("unmarshaling state: %w", err)
	}

	return &s, nil
}

func (b *LocalBackend) Lock(clusterName string) (state.UnlockFunc, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.locks[clusterName] {
		return nil, fmt.Errorf("cluster %q is already locked", clusterName)
	}

	b.locks[clusterName] = true

	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		delete(b.locks, clusterName)
	}, nil
}

func (b *LocalBackend) List() ([]string, error) {
	entries, err := os.ReadDir(b.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading state directory: %w", err)
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".json") {
			names = append(names, strings.TrimSuffix(name, ".json"))
		}
	}

	return names, nil
}

func (b *LocalBackend) Delete(clusterName string) error {
	if err := os.Remove(b.filePath(clusterName)); err != nil {
		return fmt.Errorf("deleting state file: %w", err)
	}
	return nil
}
