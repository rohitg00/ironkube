package certs

import (
	"context"
	"fmt"
	"strings"
	"time"

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

var knownCertPaths = []struct {
	name string
	path string
}{
	{"apiserver", "/etc/kubernetes/pki/apiserver.crt"},
	{"apiserver-kubelet-client", "/etc/kubernetes/pki/apiserver-kubelet-client.crt"},
	{"front-proxy-client", "/etc/kubernetes/pki/front-proxy-client.crt"},
	{"etcd-server", "/etc/kubernetes/pki/etcd/server.crt"},
}

func (m *Manager) CheckExpiry(ctx context.Context, machine provider.Machine) ([]provider.CertInfo, error) {
	var certs []provider.CertInfo

	for _, cert := range knownCertPaths {
		endDateCmd := fmt.Sprintf("openssl x509 -enddate -noout -in %s", cert.path)
		endDateOut, err := m.ssh.Run(ctx, machine.Spec.Host, machine.Spec.User, machine.Spec.Port, endDateCmd)
		if err != nil {
			continue
		}

		expiresAt, err := ParseCertDate(endDateOut)
		if err != nil {
			continue
		}

		issuerCmd := fmt.Sprintf("openssl x509 -issuer -noout -in %s", cert.path)
		issuerOut, err := m.ssh.Run(ctx, machine.Spec.Host, machine.Spec.User, machine.Spec.Port, issuerCmd)
		if err != nil {
			issuerOut = "unknown"
		}

		issuer := strings.TrimSpace(issuerOut)
		issuer = strings.TrimPrefix(issuer, "issuer=")
		issuer = strings.TrimPrefix(issuer, "issuer= ")

		certs = append(certs, provider.CertInfo{
			Name:      cert.name,
			Path:      cert.path,
			ExpiresAt: expiresAt,
			Issuer:    issuer,
		})
	}

	return certs, nil
}

func (m *Manager) ExpiringWithin(certs []provider.CertInfo, duration time.Duration) []provider.CertInfo {
	cutoff := time.Now().Add(duration)
	var expiring []provider.CertInfo
	for _, c := range certs {
		if c.ExpiresAt.Before(cutoff) {
			expiring = append(expiring, c)
		}
	}
	return expiring
}

func (m *Manager) Rotate(ctx context.Context, machine provider.Machine, distro string) error {
	var cmd string
	switch distro {
	case "kubeadm":
		cmd = "kubeadm certs renew all"
	case "k3s":
		cmd = "systemctl restart k3s"
	case "k0s":
		cmd = "k0s certs renew"
	default:
		return fmt.Errorf("unsupported distro for cert rotation: %s", distro)
	}

	_, err := m.ssh.Run(ctx, machine.Spec.Host, machine.Spec.User, machine.Spec.Port, cmd)
	return err
}

func ParseCertDate(output string) (time.Time, error) {
	output = strings.TrimSpace(output)
	prefix := "notAfter="
	idx := strings.Index(output, prefix)
	if idx == -1 {
		return time.Time{}, fmt.Errorf("could not find notAfter in output: %s", output)
	}

	dateStr := strings.TrimSpace(output[idx+len(prefix):])
	t, err := time.Parse("Jan  2 15:04:05 2006 GMT", dateStr)
	if err != nil {
		t, err = time.Parse("Jan 2 15:04:05 2006 GMT", dateStr)
		if err != nil {
			return time.Time{}, fmt.Errorf("parsing cert date %q: %w", dateStr, err)
		}
	}
	return t, nil
}
