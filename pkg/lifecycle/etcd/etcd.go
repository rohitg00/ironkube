package etcd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rohitg00/ironkube/pkg/provider"
)

type SSHExecutor interface {
	Run(ctx context.Context, host, user string, port int, command string) (string, error)
}

type Manager struct {
	ssh SSHExecutor
}

func New(ssh SSHExecutor) *Manager {
	return &Manager{ssh: ssh}
}

const etcdEnv = "ETCDCTL_API=3 "

type endpointHealthEntry struct {
	Endpoint string `json:"endpoint"`
	Health   string `json:"health"`
}

type memberListResponse struct {
	Members []memberEntry `json:"members"`
}

type memberEntry struct {
	Name       string   `json:"name"`
	ClientURLs []string `json:"clientURLs"`
	ID         uint64   `json:"ID"`
}

type endpointStatusEntry struct {
	Endpoint string       `json:"Endpoint"`
	Status   statusDetail `json:"Status"`
}

type statusDetail struct {
	DbSize  int64  `json:"dbSize"`
	Version string `json:"version"`
	Leader  uint64 `json:"leader"`
	RaftID  uint64 `json:"header>raft_id"`
}

func (m *Manager) Health(ctx context.Context, machine provider.Machine) (*provider.EtcdHealth, error) {
	host := machine.Spec.Host
	user := machine.Spec.User
	port := machine.Spec.Port

	healthOut, err := m.ssh.Run(ctx, host, user, port, etcdEnv+"etcdctl endpoint health --cluster -w json")
	if err != nil {
		return nil, fmt.Errorf("checking etcd health: %w", err)
	}

	var healthEntries []endpointHealthEntry
	if err := json.Unmarshal([]byte(healthOut), &healthEntries); err != nil {
		return nil, fmt.Errorf("parsing health output: %w", err)
	}

	allHealthy := true
	for _, h := range healthEntries {
		if h.Health != "true" {
			allHealthy = false
			break
		}
	}

	memberOut, err := m.ssh.Run(ctx, host, user, port, etcdEnv+"etcdctl member list -w json")
	if err != nil {
		return nil, fmt.Errorf("listing etcd members: %w", err)
	}

	var memberResp memberListResponse
	if err := json.Unmarshal([]byte(memberOut), &memberResp); err != nil {
		return nil, fmt.Errorf("parsing member list: %w", err)
	}

	statusOut, err := m.ssh.Run(ctx, host, user, port, etcdEnv+"etcdctl endpoint status -w json")
	if err != nil {
		return nil, fmt.Errorf("getting etcd status: %w", err)
	}

	var statusEntries []endpointStatusEntry
	if err := json.Unmarshal([]byte(statusOut), &statusEntries); err != nil {
		return nil, fmt.Errorf("parsing status output: %w", err)
	}

	leaderID := uint64(0)
	var totalDbSize int64
	version := ""
	if len(statusEntries) > 0 {
		leaderID = statusEntries[0].Status.Leader
		version = statusEntries[0].Status.Version
		for _, s := range statusEntries {
			totalDbSize += s.Status.DbSize
		}
	}

	memberIDMap := make(map[uint64]bool)
	for _, s := range statusEntries {
		memberIDMap[s.Status.RaftID] = true
	}

	var members []provider.EtcdMember
	for _, mem := range memberResp.Members {
		endpoint := ""
		if len(mem.ClientURLs) > 0 {
			endpoint = mem.ClientURLs[0]
		}

		dbSize := int64(0)
		for _, s := range statusEntries {
			if s.Endpoint == endpoint {
				dbSize = s.Status.DbSize
				break
			}
		}

		members = append(members, provider.EtcdMember{
			Name:     mem.Name,
			Endpoint: endpoint,
			IsLeader: mem.ID == leaderID,
			DbSize:   dbSize,
		})
	}

	return &provider.EtcdHealth{
		Healthy: allHealthy,
		Members: members,
		DbSize:  totalDbSize,
		Version: version,
	}, nil
}

func (m *Manager) Backup(ctx context.Context, machine provider.Machine, outputPath string) error {
	cmd := fmt.Sprintf("%setcdctl snapshot save %s", etcdEnv, outputPath)
	_, err := m.ssh.Run(ctx, machine.Spec.Host, machine.Spec.User, machine.Spec.Port, cmd)
	if err != nil {
		return fmt.Errorf("etcd backup: %w", err)
	}
	return nil
}

func (m *Manager) Restore(ctx context.Context, machine provider.Machine, snapshotPath, dataDir string) error {
	cmd := fmt.Sprintf("%setcdctl snapshot restore %s --data-dir=%s", etcdEnv, snapshotPath, dataDir)
	_, err := m.ssh.Run(ctx, machine.Spec.Host, machine.Spec.User, machine.Spec.Port, cmd)
	if err != nil {
		return fmt.Errorf("etcd restore: %w", err)
	}
	return nil
}

func (m *Manager) Defrag(ctx context.Context, machine provider.Machine) error {
	_, err := m.ssh.Run(ctx, machine.Spec.Host, machine.Spec.User, machine.Spec.Port, etcdEnv+"etcdctl defrag --cluster")
	if err != nil {
		return fmt.Errorf("etcd defrag: %w", err)
	}
	return nil
}

func (m *Manager) Compact(ctx context.Context, machine provider.Machine, revision string) error {
	cmd := fmt.Sprintf("%setcdctl compact %s", etcdEnv, revision)
	_, err := m.ssh.Run(ctx, machine.Spec.Host, machine.Spec.User, machine.Spec.Port, cmd)
	if err != nil {
		return fmt.Errorf("etcd compact: %w", err)
	}
	return nil
}
