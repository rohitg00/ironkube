package universal

import (
	"context"
	"fmt"
	"strings"
	"time"

	"k8s.io/client-go/kubernetes"

	"github.com/rohitg00/ironkube/pkg/k8s"
	"github.com/rohitg00/ironkube/pkg/provider"
)

var _ provider.ControlPlaneProvider = (*Provider)(nil)

type ClientFactory func(kubeconfig []byte) (kubernetes.Interface, error)

type SSHExecutor interface {
	Run(ctx context.Context, host, user string, port int, command string) (string, error)
}

type Provider struct {
	clusterName   string
	clientFactory ClientFactory
	sshExecutor   SSHExecutor
}

func New(clusterName string) *Provider {
	return &Provider{
		clusterName:   clusterName,
		clientFactory: k8s.NewClientFromKubeconfig,
	}
}

func NewWithFactory(clusterName string, factory ClientFactory) *Provider {
	return &Provider{
		clusterName:   clusterName,
		clientFactory: factory,
	}
}

func (p *Provider) SetSSHExecutor(executor SSHExecutor) {
	p.sshExecutor = executor
}

func (p *Provider) WaitForAPI(ctx context.Context, kubeconfig []byte, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	backoff := 500 * time.Millisecond

	for {
		client, err := p.clientFactory(kubeconfig)
		if err == nil {
			_, listErr := k8s.ListNodes(ctx, client)
			if listErr == nil {
				return nil
			}
		}

		if time.Now().After(deadline) {
			if err != nil {
				return fmt.Errorf("API not ready after %s: %w", timeout, err)
			}
			return fmt.Errorf("API not ready after %s", timeout)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}

		if backoff < 10*time.Second {
			backoff = backoff * 2
		}
	}
}

func (p *Provider) NodeReady(ctx context.Context, kubeconfig []byte, nodeName string) (bool, error) {
	client, err := p.clientFactory(kubeconfig)
	if err != nil {
		return false, fmt.Errorf("creating client: %w", err)
	}
	return k8s.IsNodeReady(ctx, client, nodeName)
}

func (p *Provider) WaitForNodes(ctx context.Context, kubeconfig []byte, expected int, timeout time.Duration) error {
	client, err := p.clientFactory(kubeconfig)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}
	return k8s.WaitForNodes(ctx, client, expected, timeout)
}

func (p *Provider) DrainNode(ctx context.Context, kubeconfig []byte, nodeName string, opts provider.DrainOptions) error {
	client, err := p.clientFactory(kubeconfig)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}
	return k8s.DrainNode(ctx, client, nodeName, opts)
}

func (p *Provider) UncordonNode(ctx context.Context, kubeconfig []byte, nodeName string) error {
	client, err := p.clientFactory(kubeconfig)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}
	return k8s.UncordonNode(ctx, client, nodeName)
}

func (p *Provider) Health(ctx context.Context, kubeconfig []byte) (*provider.HealthReport, error) {
	client, err := p.clientFactory(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("creating client: %w", err)
	}
	return k8s.ClusterHealth(ctx, client, p.clusterName)
}

func (p *Provider) ListNodes(ctx context.Context, kubeconfig []byte) ([]provider.NodeHealth, error) {
	client, err := p.clientFactory(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("creating client: %w", err)
	}
	return k8s.ListNodes(ctx, client)
}

var certPaths = []struct {
	name string
	path string
}{
	{"apiserver", "/etc/kubernetes/pki/apiserver.crt"},
	{"apiserver-kubelet-client", "/etc/kubernetes/pki/apiserver-kubelet-client.crt"},
	{"front-proxy-client", "/etc/kubernetes/pki/front-proxy-client.crt"},
	{"apiserver-etcd-client", "/etc/kubernetes/pki/apiserver-etcd-client.crt"},
	{"etcd-server", "/etc/kubernetes/pki/etcd/server.crt"},
	{"etcd-peer", "/etc/kubernetes/pki/etcd/peer.crt"},
}

func (p *Provider) CertExpiry(_ context.Context, _ []byte) ([]provider.CertInfo, error) {
	if p.sshExecutor == nil {
		return nil, nil
	}
	return nil, nil
}

func (p *Provider) CertExpiryWithMachine(ctx context.Context, machine provider.Machine) ([]provider.CertInfo, error) {
	if p.sshExecutor == nil {
		return nil, nil
	}

	var certs []provider.CertInfo
	for _, cp := range certPaths {
		cmd := fmt.Sprintf("openssl x509 -enddate -issuer -noout -in %s 2>/dev/null", cp.path)
		output, err := p.sshExecutor.Run(ctx, machine.Spec.Host, machine.Spec.User, machine.Spec.Port, cmd)
		if err != nil {
			continue
		}

		ci := provider.CertInfo{
			Name: cp.name,
			Path: cp.path,
		}

		for _, line := range strings.Split(output, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "notAfter=") {
				dateStr := strings.TrimPrefix(line, "notAfter=")
				t, parseErr := ParseCertDate(dateStr)
				if parseErr != nil {
					continue
				}
				ci.ExpiresAt = t
			}
			if strings.HasPrefix(line, "issuer=") {
				parts := strings.Split(line, "CN = ")
				if len(parts) > 1 {
					ci.Issuer = strings.TrimSpace(parts[len(parts)-1])
				}
			}
		}

		if !ci.ExpiresAt.IsZero() {
			certs = append(certs, ci)
		}
	}

	return certs, nil
}

func (p *Provider) EtcdHealth(_ context.Context, _ []byte) (*provider.EtcdHealth, error) {
	if p.sshExecutor == nil {
		return &provider.EtcdHealth{}, nil
	}
	return &provider.EtcdHealth{}, nil
}

func (p *Provider) EtcdHealthWithMachine(ctx context.Context, machine provider.Machine) (*provider.EtcdHealth, error) {
	if p.sshExecutor == nil {
		return &provider.EtcdHealth{}, nil
	}

	cmd := "ETCDCTL_API=3 etcdctl --cacert=/etc/kubernetes/pki/etcd/ca.crt " +
		"--cert=/etc/kubernetes/pki/etcd/server.crt " +
		"--key=/etc/kubernetes/pki/etcd/server.key " +
		"endpoint health --write-out=table 2>/dev/null"

	output, err := p.sshExecutor.Run(ctx, machine.Spec.Host, machine.Spec.User, machine.Spec.Port, cmd)
	if err != nil {
		return &provider.EtcdHealth{Healthy: false}, nil
	}

	health := &provider.EtcdHealth{
		Healthy: strings.Contains(output, "true"),
	}

	versionCmd := "etcdctl version 2>/dev/null | head -1"
	versionOutput, err := p.sshExecutor.Run(ctx, machine.Spec.Host, machine.Spec.User, machine.Spec.Port, versionCmd)
	if err == nil {
		parts := strings.Split(strings.TrimSpace(versionOutput), ":")
		if len(parts) > 1 {
			health.Version = strings.TrimSpace(parts[1])
		}
	}

	return health, nil
}

func ParseCertDate(dateStr string) (time.Time, error) {
	dateStr = strings.TrimSpace(dateStr)
	formats := []string{
		"Jan  2 15:04:05 2006 MST",
		"Jan 2 15:04:05 2006 MST",
		"Jan  2 15:04:05 2006 GMT",
		"Jan 2 15:04:05 2006 GMT",
	}
	for _, format := range formats {
		t, err := time.Parse(format, dateStr)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse certificate date: %s", dateStr)
}
