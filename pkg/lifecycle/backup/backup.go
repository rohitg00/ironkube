package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rohitg00/ironkube/pkg/provider"
)

type EtcdBackupper interface {
	Backup(ctx context.Context, machine provider.Machine, outputPath string) error
	Restore(ctx context.Context, machine provider.Machine, snapshotPath, dataDir string) error
}

type StateExporter interface {
	ExportState(ctx context.Context, clusterName string) ([]byte, error)
}

type BackupOpts struct {
	IncludeState bool
	Prefix       string
}

type RestoreOpts struct {
	DataDir string
}

type BackupResult struct {
	EtcdSnapshotPath string
	StatePath        string
	Timestamp        time.Time
	Machine          provider.Machine
}

type Manager struct {
	etcd  EtcdBackupper
	state StateExporter
}

func New(etcd EtcdBackupper, state StateExporter) *Manager {
	return &Manager{etcd: etcd, state: state}
}

const defaultDataDir = "/var/lib/etcd"

func (m *Manager) Backup(ctx context.Context, machine provider.Machine, outputDir string, opts BackupOpts) (*BackupResult, error) {
	if outputDir == "" {
		return nil, fmt.Errorf("output directory is required")
	}

	timestamp := time.Now()
	prefix := opts.Prefix
	if prefix == "" {
		prefix = "etcd-snapshot"
	}

	snapshotName := fmt.Sprintf("%s-%s.db", prefix, timestamp.Format("20060102-150405"))
	snapshotPath := filepath.Join(outputDir, snapshotName)

	if err := m.etcd.Backup(ctx, machine, snapshotPath); err != nil {
		return nil, fmt.Errorf("creating etcd snapshot: %w", err)
	}

	result := &BackupResult{
		EtcdSnapshotPath: snapshotPath,
		Timestamp:        timestamp,
		Machine:          machine,
	}

	if opts.IncludeState {
		stateData, err := m.state.ExportState(ctx, machine.ID)
		if err != nil {
			return nil, fmt.Errorf("exporting state: %w", err)
		}

		stateName := fmt.Sprintf("%s-state-%s.json", prefix, timestamp.Format("20060102-150405"))
		statePath := filepath.Join(outputDir, stateName)
		if err := os.WriteFile(statePath, stateData, 0600); err != nil {
			return nil, fmt.Errorf("writing state backup: %w", err)
		}
		result.StatePath = statePath
	}

	return result, nil
}

func (m *Manager) Restore(ctx context.Context, machine provider.Machine, snapshotPath string, opts RestoreOpts) error {
	dataDir := opts.DataDir
	if dataDir == "" {
		dataDir = defaultDataDir
	}

	if err := m.etcd.Restore(ctx, machine, snapshotPath, dataDir); err != nil {
		return fmt.Errorf("restoring etcd snapshot: %w", err)
	}

	return nil
}
