package provider

import "context"

type BootstrapProvider interface {
	Name() string
	ValidateVersion(version string) error
	SupportedVersions() []string
	InitControlPlane(ctx context.Context, cluster *ClusterSpec, machine Machine, token string) (*BootstrapData, error)
	JoinControlPlane(ctx context.Context, cluster *ClusterSpec, machine Machine, joinEndpoint, token string) (*BootstrapData, error)
	JoinWorker(ctx context.Context, cluster *ClusterSpec, machine Machine, joinEndpoint, token string) (*BootstrapData, error)
	Uninstall(ctx context.Context, machine Machine, role string) (*BootstrapData, error)
	Upgrade(ctx context.Context, machine Machine, toVersion string) (*BootstrapData, error)
	SecurityArgs(profile SecurityProfile) []string
	KubeconfigPath() string
	TokenCommand() string
}
