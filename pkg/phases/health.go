package phases

import (
	"context"
	"fmt"
	"strings"

	"github.com/rohitg00/ironkube/pkg/config"
	"github.com/rohitg00/ironkube/pkg/distro"
	"github.com/rohitg00/ironkube/pkg/engine"
	"github.com/rohitg00/ironkube/pkg/health"
	"github.com/rohitg00/ironkube/pkg/ssh"
)

type HealthCheckPhase struct {
	Config *config.ClusterConfig
}

func (p *HealthCheckPhase) Name() string { return "health-check" }

func (p *HealthCheckPhase) Run(ctx context.Context, state *engine.State) error {
	v, ok := state.Get("executor")
	if !ok {
		return fmt.Errorf("executor not found in state")
	}
	exec := v.(*ssh.Executor)

	v, ok = state.Get("distro_plugin")
	if !ok {
		return fmt.Errorf("distro_plugin not found in state")
	}
	plugin := v.(distro.Plugin)

	firstNode := p.Config.Spec.ControlPlane.Nodes[0]
	kubectl := kubectlCmd(plugin)

	clusterHealth := &health.ClusterHealth{
		ClusterName: p.Config.Metadata.Name,
	}

	clusterHealth.Results = append(clusterHealth.Results, checkNodes(exec, firstNode.Host, kubectl)...)
	clusterHealth.Results = append(clusterHealth.Results, checkSystemPods(exec, firstNode.Host, kubectl)...)
	clusterHealth.Results = append(clusterHealth.Results, checkAPIServer(exec, firstNode.Host, kubectl))

	for _, node := range p.Config.Spec.ControlPlane.Nodes {
		clusterHealth.Results = append(clusterHealth.Results, checkSSH(exec, node.Host))
	}

	state.Set("cluster_health", clusterHealth)
	return nil
}

func checkNodes(exec *ssh.Executor, host, kubectl string) []health.CheckResult {
	out, err := exec.RunOnHost(host, fmt.Sprintf("%s get nodes --no-headers 2>/dev/null", kubectl))
	if err != nil {
		return []health.CheckResult{{
			Name:    "nodes",
			Status:  health.StatusUnhealthy,
			Message: fmt.Sprintf("failed to get nodes: %v", err),
		}}
	}

	var results []health.CheckResult
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		nodeName := fields[0]
		status := health.StatusHealthy
		if fields[1] != "Ready" {
			status = health.StatusUnhealthy
		}
		results = append(results, health.CheckResult{
			Name:    fmt.Sprintf("node/%s", nodeName),
			Status:  status,
			Message: fields[1],
		})
	}
	return results
}

func checkSystemPods(exec *ssh.Executor, host, kubectl string) []health.CheckResult {
	out, err := exec.RunOnHost(host, fmt.Sprintf("%s get pods -n kube-system --no-headers 2>/dev/null", kubectl))
	if err != nil {
		return []health.CheckResult{{
			Name:    "system-pods",
			Status:  health.StatusUnhealthy,
			Message: fmt.Sprintf("failed to get system pods: %v", err),
		}}
	}

	running := 0
	notRunning := 0
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		if fields[2] == "Running" || fields[2] == "Completed" {
			running++
		} else {
			notRunning++
		}
	}

	status := health.StatusHealthy
	if notRunning > 0 {
		status = health.StatusUnhealthy
	}
	return []health.CheckResult{{
		Name:    "system-pods",
		Status:  status,
		Message: fmt.Sprintf("%d running, %d not running", running, notRunning),
	}}
}

func checkAPIServer(exec *ssh.Executor, host, kubectl string) health.CheckResult {
	_, err := exec.RunOnHost(host, fmt.Sprintf("%s cluster-info 2>/dev/null", kubectl))
	if err != nil {
		return health.CheckResult{
			Name:    "api-server",
			Status:  health.StatusUnhealthy,
			Message: "API server not responding",
		}
	}
	return health.CheckResult{
		Name:    "api-server",
		Status:  health.StatusHealthy,
		Message: "API server responding",
	}
}

func checkSSH(exec *ssh.Executor, host string) health.CheckResult {
	_, err := exec.RunOnHost(host, "echo ok")
	if err != nil {
		return health.CheckResult{
			Name:    fmt.Sprintf("ssh/%s", host),
			Status:  health.StatusUnhealthy,
			Message: fmt.Sprintf("SSH unreachable: %v", err),
		}
	}
	return health.CheckResult{
		Name:    fmt.Sprintf("ssh/%s", host),
		Status:  health.StatusHealthy,
		Message: "SSH reachable",
	}
}
