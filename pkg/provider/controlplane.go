package provider

import (
	"context"
	"time"
)

type ControlPlaneProvider interface {
	WaitForAPI(ctx context.Context, kubeconfig []byte, timeout time.Duration) error
	NodeReady(ctx context.Context, kubeconfig []byte, nodeName string) (bool, error)
	WaitForNodes(ctx context.Context, kubeconfig []byte, expected int, timeout time.Duration) error
	DrainNode(ctx context.Context, kubeconfig []byte, nodeName string, opts DrainOptions) error
	UncordonNode(ctx context.Context, kubeconfig []byte, nodeName string) error
	Health(ctx context.Context, kubeconfig []byte) (*HealthReport, error)
	ListNodes(ctx context.Context, kubeconfig []byte) ([]NodeHealth, error)
	CertExpiry(ctx context.Context, kubeconfig []byte) ([]CertInfo, error)
	EtcdHealth(ctx context.Context, kubeconfig []byte) (*EtcdHealth, error)
}
