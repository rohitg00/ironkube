package state

import (
	"time"

	"github.com/rohitg00/ironkube/pkg/provider"
)

type StateMetadata struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Version   string    `json:"version"`
}

type OperationType string

const (
	OpApply      OperationType = "apply"
	OpDestroy    OperationType = "destroy"
	OpUpgrade    OperationType = "upgrade"
	OpBackup     OperationType = "backup"
	OpRestore    OperationType = "restore"
	OpCertRotate OperationType = "cert-rotate"
)

type OperationStatus string

const (
	OpStatusRunning OperationStatus = "running"
	OpStatusSuccess OperationStatus = "success"
	OpStatusFailed  OperationStatus = "failed"
)

type OperationRecord struct {
	Type       OperationType   `json:"type"`
	Status     OperationStatus `json:"status"`
	StartedAt  time.Time       `json:"startedAt"`
	FinishedAt time.Time       `json:"finishedAt,omitempty"`
	Message    string          `json:"message,omitempty"`
	Version    string          `json:"version,omitempty"`
}

type MachineState struct {
	ID           string    `json:"id"`
	ProviderID   string    `json:"providerId,omitempty"`
	Host         string    `json:"host"`
	Role         string    `json:"role"`
	K8sVersion   string    `json:"k8sVersion"`
	Status       string    `json:"status"`
	JoinedAt     time.Time `json:"joinedAt,omitempty"`
	LastHealthAt time.Time `json:"lastHealthAt,omitempty"`
}

type AddonState struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Installed bool   `json:"installed"`
	Ready     bool   `json:"ready"`
}

type CertState struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	ExpiresAt time.Time `json:"expiresAt"`
	RotatedAt time.Time `json:"rotatedAt,omitempty"`
}

type ClusterState struct {
	Metadata      StateMetadata           `json:"metadata"`
	DesiredSpec   any                     `json:"desiredSpec,omitempty"`
	Machines      map[string]MachineState `json:"machines,omitempty"`
	Addons        map[string]AddonState   `json:"addons,omitempty"`
	Certs         []CertState             `json:"certs,omitempty"`
	Kubeconfig    []byte                  `json:"kubeconfig,omitempty"`
	LastOperation OperationRecord         `json:"lastOperation"`
	History       []OperationRecord       `json:"history,omitempty"`
	EtcdHealth    *provider.EtcdHealth    `json:"etcdHealth,omitempty"`
}

func (s *ClusterState) RecordOperation(op OperationRecord) {
	s.LastOperation = op
	s.History = append(s.History, op)
	s.Metadata.UpdatedAt = time.Now()
}

func (s *ClusterState) SetMachine(id string, m MachineState) {
	if s.Machines == nil {
		s.Machines = make(map[string]MachineState)
	}
	s.Machines[id] = m
}

func (s *ClusterState) SetAddon(name string, a AddonState) {
	if s.Addons == nil {
		s.Addons = make(map[string]AddonState)
	}
	s.Addons[name] = a
}
