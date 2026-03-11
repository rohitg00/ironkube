package phases

import (
	"context"
	"testing"

	"github.com/rohitg00/ironkube/pkg/config"
	"github.com/rohitg00/ironkube/pkg/distro/k3s"
	"github.com/rohitg00/ironkube/pkg/engine"
	"github.com/rohitg00/ironkube/pkg/ssh"
	"github.com/stretchr/testify/assert"
)

func TestAddonsPhaseName(t *testing.T) {
	p := &AddonsPhase{Config: &config.ClusterConfig{}}
	assert.Equal(t, "addons", p.Name())
}

func TestAddonsPhaseMissingExecutor(t *testing.T) {
	p := &AddonsPhase{Config: validTestConfig()}
	state := engine.NewState()
	err := p.Run(context.Background(), state)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "executor not found")
}

func TestAddonsPhaseMissingDistroPlugin(t *testing.T) {
	p := &AddonsPhase{Config: validTestConfig()}
	state := engine.NewState()
	state.Set("executor", (*ssh.Executor)(nil))
	err := p.Run(context.Background(), state)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "distro_plugin not found")
}

func TestWithKubeconfigK3s(t *testing.T) {
	plugin := k3s.New()
	result := withKubeconfig("helm install foo", plugin)
	assert.Equal(t, "KUBECONFIG=/etc/rancher/k3s/k3s.yaml helm install foo", result)
}

func TestWithKubeconfigKubeadm(t *testing.T) {
	plugin := &fakePlugin{name: "kubeadm"}
	result := withKubeconfig("helm install foo", plugin)
	assert.Equal(t, "KUBECONFIG=/etc/kubernetes/admin.conf helm install foo", result)
}

type fakePlugin struct {
	name string
}

func (f *fakePlugin) Name() string { return f.name }
func (f *fakePlugin) ValidateVersion(string) error {
	return nil
}
func (f *fakePlugin) ServerInstallScript(config.Node, *config.ClusterConfig, string, bool) string {
	return ""
}
func (f *fakePlugin) AgentInstallScript(config.Node, *config.ClusterConfig, string, string) string {
	return ""
}
func (f *fakePlugin) GetKubeconfigCmd() string                                     { return "" }
func (f *fakePlugin) KubeconfigPath() string                                       { return "" }
func (f *fakePlugin) UninstallCmd(string) string                                   { return "" }
func (f *fakePlugin) UpgradeCmd(string) string                                     { return "" }
func (f *fakePlugin) SecurityArgs([]string, []string, []string) string             { return "" }
