package certs

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rohitg00/ironkube/pkg/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSSH struct {
	responses map[string]string
	errors    map[string]error
	calls     []string
}

func newMockSSH() *mockSSH {
	return &mockSSH{
		responses: make(map[string]string),
		errors:    make(map[string]error),
	}
}

func (m *mockSSH) Run(_ context.Context, host, user string, port int, command string) (string, error) {
	m.calls = append(m.calls, command)
	if err, ok := m.errors[command]; ok {
		return "", err
	}
	if resp, ok := m.responses[command]; ok {
		return resp, nil
	}
	return "", fmt.Errorf("unexpected command: %s", command)
}

func testMachine() provider.Machine {
	return provider.Machine{
		ID: "test-1",
		Spec: provider.MachineSpec{
			Role: "control-plane",
			Host: "10.0.0.1",
			User: "root",
			Port: 22,
		},
	}
}

func TestCheckExpiryParsesValidOutput(t *testing.T) {
	ssh := newMockSSH()
	ssh.responses["openssl x509 -enddate -noout -in /etc/kubernetes/pki/apiserver.crt"] = "notAfter=Dec 31 23:59:59 2026 GMT"
	ssh.responses["openssl x509 -issuer -noout -in /etc/kubernetes/pki/apiserver.crt"] = "issuer=CN = kubernetes"
	ssh.errors["openssl x509 -enddate -noout -in /etc/kubernetes/pki/apiserver-kubelet-client.crt"] = fmt.Errorf("file not found")
	ssh.errors["openssl x509 -enddate -noout -in /etc/kubernetes/pki/front-proxy-client.crt"] = fmt.Errorf("file not found")
	ssh.errors["openssl x509 -enddate -noout -in /etc/kubernetes/pki/etcd/server.crt"] = fmt.Errorf("file not found")

	mgr := New(ssh)
	certs, err := mgr.CheckExpiry(context.Background(), testMachine())
	require.NoError(t, err)
	require.Len(t, certs, 1)

	assert.Equal(t, "apiserver", certs[0].Name)
	assert.Equal(t, "/etc/kubernetes/pki/apiserver.crt", certs[0].Path)
	assert.Equal(t, 2026, certs[0].ExpiresAt.Year())
	assert.Equal(t, "CN = kubernetes", certs[0].Issuer)
}

func TestCheckExpiryHandlesMissingCert(t *testing.T) {
	ssh := newMockSSH()
	for _, cert := range knownCertPaths {
		ssh.errors[fmt.Sprintf("openssl x509 -enddate -noout -in %s", cert.path)] = fmt.Errorf("file not found")
	}

	mgr := New(ssh)
	certs, err := mgr.CheckExpiry(context.Background(), testMachine())
	require.NoError(t, err)
	assert.Empty(t, certs)
}

func TestCheckExpiryParsesMultipleCerts(t *testing.T) {
	ssh := newMockSSH()
	for _, cert := range knownCertPaths {
		ssh.responses[fmt.Sprintf("openssl x509 -enddate -noout -in %s", cert.path)] = "notAfter=Jun 15 12:00:00 2027 GMT"
		ssh.responses[fmt.Sprintf("openssl x509 -issuer -noout -in %s", cert.path)] = "issuer=CN = kubernetes-ca"
	}

	mgr := New(ssh)
	certs, err := mgr.CheckExpiry(context.Background(), testMachine())
	require.NoError(t, err)
	assert.Len(t, certs, 4)

	for _, c := range certs {
		assert.Equal(t, 2027, c.ExpiresAt.Year())
		assert.Equal(t, "CN = kubernetes-ca", c.Issuer)
	}
}

func TestCheckExpiryIssuerErrorUsesEmpty(t *testing.T) {
	ssh := newMockSSH()
	ssh.responses["openssl x509 -enddate -noout -in /etc/kubernetes/pki/apiserver.crt"] = "notAfter=Dec 31 23:59:59 2026 GMT"
	ssh.errors["openssl x509 -issuer -noout -in /etc/kubernetes/pki/apiserver.crt"] = fmt.Errorf("command failed")
	ssh.errors["openssl x509 -enddate -noout -in /etc/kubernetes/pki/apiserver-kubelet-client.crt"] = fmt.Errorf("not found")
	ssh.errors["openssl x509 -enddate -noout -in /etc/kubernetes/pki/front-proxy-client.crt"] = fmt.Errorf("not found")
	ssh.errors["openssl x509 -enddate -noout -in /etc/kubernetes/pki/etcd/server.crt"] = fmt.Errorf("not found")

	mgr := New(ssh)
	certs, err := mgr.CheckExpiry(context.Background(), testMachine())
	require.NoError(t, err)
	require.Len(t, certs, 1)
	assert.Equal(t, "unknown", certs[0].Issuer)
}

func TestExpiringWithinFiltersSomeExpiring(t *testing.T) {
	mgr := New(nil)
	now := time.Now()
	certs := []provider.CertInfo{
		{Name: "soon", ExpiresAt: now.Add(24 * time.Hour)},
		{Name: "later", ExpiresAt: now.Add(365 * 24 * time.Hour)},
		{Name: "very-soon", ExpiresAt: now.Add(1 * time.Hour)},
	}

	expiring := mgr.ExpiringWithin(certs, 30*24*time.Hour)
	assert.Len(t, expiring, 2)
	names := []string{expiring[0].Name, expiring[1].Name}
	assert.Contains(t, names, "soon")
	assert.Contains(t, names, "very-soon")
}

func TestExpiringWithinEmptyList(t *testing.T) {
	mgr := New(nil)
	expiring := mgr.ExpiringWithin(nil, 30*24*time.Hour)
	assert.Empty(t, expiring)
}

func TestExpiringWithinAllExpiring(t *testing.T) {
	mgr := New(nil)
	now := time.Now()
	certs := []provider.CertInfo{
		{Name: "a", ExpiresAt: now.Add(1 * time.Hour)},
		{Name: "b", ExpiresAt: now.Add(2 * time.Hour)},
		{Name: "c", ExpiresAt: now.Add(3 * time.Hour)},
	}

	expiring := mgr.ExpiringWithin(certs, 24*time.Hour)
	assert.Len(t, expiring, 3)
}

func TestExpiringWithinNoneExpiring(t *testing.T) {
	mgr := New(nil)
	now := time.Now()
	certs := []provider.CertInfo{
		{Name: "a", ExpiresAt: now.Add(365 * 24 * time.Hour)},
		{Name: "b", ExpiresAt: now.Add(730 * 24 * time.Hour)},
	}

	expiring := mgr.ExpiringWithin(certs, 30*24*time.Hour)
	assert.Empty(t, expiring)
}

func TestExpiringWithinAlreadyExpired(t *testing.T) {
	mgr := New(nil)
	now := time.Now()
	certs := []provider.CertInfo{
		{Name: "expired", ExpiresAt: now.Add(-24 * time.Hour)},
		{Name: "valid", ExpiresAt: now.Add(365 * 24 * time.Hour)},
	}

	expiring := mgr.ExpiringWithin(certs, 30*24*time.Hour)
	assert.Len(t, expiring, 1)
	assert.Equal(t, "expired", expiring[0].Name)
}

func TestRotateKubeadm(t *testing.T) {
	ssh := newMockSSH()
	ssh.responses["kubeadm certs renew all"] = "done"
	mgr := New(ssh)

	err := mgr.Rotate(context.Background(), testMachine(), "kubeadm")
	require.NoError(t, err)
	assert.Contains(t, ssh.calls, "kubeadm certs renew all")
}

func TestRotateK3s(t *testing.T) {
	ssh := newMockSSH()
	ssh.responses["systemctl restart k3s"] = ""
	mgr := New(ssh)

	err := mgr.Rotate(context.Background(), testMachine(), "k3s")
	require.NoError(t, err)
	assert.Contains(t, ssh.calls, "systemctl restart k3s")
}

func TestRotateK0s(t *testing.T) {
	ssh := newMockSSH()
	ssh.responses["k0s certs renew"] = "done"
	mgr := New(ssh)

	err := mgr.Rotate(context.Background(), testMachine(), "k0s")
	require.NoError(t, err)
	assert.Contains(t, ssh.calls, "k0s certs renew")
}

func TestRotateUnknownDistro(t *testing.T) {
	ssh := newMockSSH()
	mgr := New(ssh)

	err := mgr.Rotate(context.Background(), testMachine(), "unknown-distro")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported distro")
}

func TestRotateSSHError(t *testing.T) {
	ssh := newMockSSH()
	ssh.errors["kubeadm certs renew all"] = fmt.Errorf("connection refused")
	mgr := New(ssh)

	err := mgr.Rotate(context.Background(), testMachine(), "kubeadm")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")
}

func TestParseCertDateValid(t *testing.T) {
	result, err := ParseCertDate("notAfter=Dec 31 23:59:59 2026 GMT")
	require.NoError(t, err)
	assert.Equal(t, 2026, result.Year())
	assert.Equal(t, time.December, result.Month())
	assert.Equal(t, 31, result.Day())
}

func TestParseCertDateSingleDigitDay(t *testing.T) {
	result, err := ParseCertDate("notAfter=Jan 5 10:30:00 2027 GMT")
	require.NoError(t, err)
	assert.Equal(t, 2027, result.Year())
	assert.Equal(t, time.January, result.Month())
	assert.Equal(t, 5, result.Day())
}

func TestParseCertDateInvalid(t *testing.T) {
	_, err := ParseCertDate("garbage data")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not find notAfter")
}

func TestParseCertDateInvalidDate(t *testing.T) {
	_, err := ParseCertDate("notAfter=not-a-date")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing cert date")
}
