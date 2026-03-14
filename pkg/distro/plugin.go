package distro

import "github.com/rohitg00/ironkube/pkg/config"

type Plugin interface {
	Name() string
	ValidateVersion(version string) error
	ServerInstallScript(node config.Node, cfg *config.ClusterConfig, token string, isInit bool, secFlags config.SecurityFlags, certKey string) string
	AgentInstallScript(node config.Node, cfg *config.ClusterConfig, serverURL string, token string) string
	GetKubeconfigCmd() string
	KubeconfigPath() string
	UninstallCmd(role string) string
	UpgradeCmd(version string) string
}
