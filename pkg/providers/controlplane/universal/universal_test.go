package universal

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/rohitg00/ironkube/pkg/provider"
)

func readyNode(name string) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{"node-role.kubernetes.io/control-plane": ""},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			},
			NodeInfo: corev1.NodeSystemInfo{KubeletVersion: "v1.34.3"},
		},
	}
}

func notReadyNode(name string) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionFalse},
			},
		},
	}
}

func newFakeProvider(client *fake.Clientset) *Provider {
	return &Provider{
		clusterName: "test-cluster",
		clientFactory: func(_ []byte) (kubernetes.Interface, error) {
			return client, nil
		},
	}
}

func errorFactory(msg string) ClientFactory {
	return func(_ []byte) (kubernetes.Interface, error) {
		return nil, fmt.Errorf("%s", msg)
	}
}

type mockSSHExecutor struct {
	outputs map[string]string
	errors  map[string]error
}

func newMockSSH() *mockSSHExecutor {
	return &mockSSHExecutor{
		outputs: make(map[string]string),
		errors:  make(map[string]error),
	}
}

func (m *mockSSHExecutor) Run(_ context.Context, _, _ string, _ int, command string) (string, error) {
	if err, ok := m.errors[command]; ok {
		return "", err
	}
	if output, ok := m.outputs[command]; ok {
		return output, nil
	}
	return "", fmt.Errorf("command not mocked: %s", command)
}

var _ provider.ControlPlaneProvider = (*Provider)(nil)

func TestWaitForAPI_Success(t *testing.T) {
	client := fake.NewSimpleClientset(readyNode("cp1"))
	p := newFakeProvider(client)

	err := p.WaitForAPI(context.Background(), []byte("kc"), 5*time.Second)
	assert.NoError(t, err)
}

func TestWaitForAPI_InvalidKubeconfig(t *testing.T) {
	p := &Provider{
		clusterName:   "test",
		clientFactory: errorFactory("invalid kubeconfig"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := p.WaitForAPI(ctx, []byte("bad"), 100*time.Millisecond)
	assert.Error(t, err)
}

func TestWaitForAPI_ContextCancelled(t *testing.T) {
	p := &Provider{
		clusterName:   "test",
		clientFactory: errorFactory("not ready"),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := p.WaitForAPI(ctx, []byte("kc"), 30*time.Second)
	assert.Error(t, err)
}

func TestNodeReady_True(t *testing.T) {
	client := fake.NewSimpleClientset(readyNode("cp1"))
	p := newFakeProvider(client)

	ready, err := p.NodeReady(context.Background(), []byte("kc"), "cp1")
	require.NoError(t, err)
	assert.True(t, ready)
}

func TestNodeReady_False(t *testing.T) {
	client := fake.NewSimpleClientset(notReadyNode("worker1"))
	p := newFakeProvider(client)

	ready, err := p.NodeReady(context.Background(), []byte("kc"), "worker1")
	require.NoError(t, err)
	assert.False(t, ready)
}

func TestNodeReady_NotFound(t *testing.T) {
	client := fake.NewSimpleClientset()
	p := newFakeProvider(client)

	_, err := p.NodeReady(context.Background(), []byte("kc"), "nonexistent")
	assert.Error(t, err)
}

func TestNodeReady_ClientError(t *testing.T) {
	p := &Provider{
		clusterName:   "test",
		clientFactory: errorFactory("connection refused"),
	}

	_, err := p.NodeReady(context.Background(), []byte("kc"), "cp1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "creating client")
}

func TestWaitForNodes_AlreadyReady(t *testing.T) {
	client := fake.NewSimpleClientset(readyNode("cp1"), readyNode("worker1"))
	p := newFakeProvider(client)

	err := p.WaitForNodes(context.Background(), []byte("kc"), 2, 5*time.Second)
	assert.NoError(t, err)
}

func TestWaitForNodes_ClientError(t *testing.T) {
	p := &Provider{
		clusterName:   "test",
		clientFactory: errorFactory("broken"),
	}

	err := p.WaitForNodes(context.Background(), []byte("kc"), 1, 5*time.Second)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "creating client")
}

func TestDrainNode_CordonsAndDeletesPods(t *testing.T) {
	node := readyNode("worker1")
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app-pod",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "ReplicaSet", Name: "app-rs"},
			},
		},
		Spec: corev1.PodSpec{NodeName: "worker1"},
	}
	client := fake.NewSimpleClientset(node, pod)
	p := newFakeProvider(client)

	err := p.DrainNode(context.Background(), []byte("kc"), "worker1", provider.DrainOptions{
		GracePeriod:      30,
		IgnoreDaemonSets: true,
	})
	require.NoError(t, err)

	updatedNode, err := client.CoreV1().Nodes().Get(context.Background(), "worker1", metav1.GetOptions{})
	require.NoError(t, err)
	assert.True(t, updatedNode.Spec.Unschedulable)
}

func TestDrainNode_ClientError(t *testing.T) {
	p := &Provider{
		clusterName:   "test",
		clientFactory: errorFactory("no client"),
	}

	err := p.DrainNode(context.Background(), []byte("kc"), "worker1", provider.DrainOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "creating client")
}

func TestUncordonNode_Success(t *testing.T) {
	node := readyNode("worker1")
	node.Spec.Unschedulable = true
	client := fake.NewSimpleClientset(node)
	p := newFakeProvider(client)

	err := p.UncordonNode(context.Background(), []byte("kc"), "worker1")
	require.NoError(t, err)

	updatedNode, err := client.CoreV1().Nodes().Get(context.Background(), "worker1", metav1.GetOptions{})
	require.NoError(t, err)
	assert.False(t, updatedNode.Spec.Unschedulable)
}

func TestUncordonNode_ClientError(t *testing.T) {
	p := &Provider{
		clusterName:   "test",
		clientFactory: errorFactory("down"),
	}

	err := p.UncordonNode(context.Background(), []byte("kc"), "worker1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "creating client")
}

func TestHealth_ReturnsReport(t *testing.T) {
	client := fake.NewSimpleClientset(
		readyNode("cp1"),
		readyNode("worker1"),
		notReadyNode("worker2"),
	)
	p := newFakeProvider(client)

	report, err := p.Health(context.Background(), []byte("kc"))
	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Equal(t, "test-cluster", report.ClusterName)
	assert.Len(t, report.Nodes, 3)
	assert.Equal(t, 2, report.ReadyCount())
	assert.False(t, report.AllHealthy())
}

func TestHealth_ClientError(t *testing.T) {
	p := &Provider{
		clusterName:   "test",
		clientFactory: errorFactory("timeout"),
	}

	_, err := p.Health(context.Background(), []byte("kc"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "creating client")
}

func TestListNodes_ReturnsList(t *testing.T) {
	client := fake.NewSimpleClientset(readyNode("cp1"), readyNode("worker1"))
	p := newFakeProvider(client)

	nodes, err := p.ListNodes(context.Background(), []byte("kc"))
	require.NoError(t, err)
	assert.Len(t, nodes, 2)
}

func TestListNodes_ClientError(t *testing.T) {
	p := &Provider{
		clusterName:   "test",
		clientFactory: errorFactory("gone"),
	}

	_, err := p.ListNodes(context.Background(), []byte("kc"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "creating client")
}

func TestCertExpiry_NoSSHExecutor(t *testing.T) {
	client := fake.NewSimpleClientset()
	p := newFakeProvider(client)

	certs, err := p.CertExpiry(context.Background(), []byte("kc"))
	assert.NoError(t, err)
	assert.Nil(t, certs)
}

func TestEtcdHealth_NoSSHExecutor(t *testing.T) {
	client := fake.NewSimpleClientset()
	p := newFakeProvider(client)

	health, err := p.EtcdHealth(context.Background(), []byte("kc"))
	assert.NoError(t, err)
	assert.NotNil(t, health)
}

func TestCertExpiryWithMachine_NoSSHExecutor(t *testing.T) {
	client := fake.NewSimpleClientset()
	p := newFakeProvider(client)

	machine := provider.Machine{
		Spec: provider.MachineSpec{Host: "10.0.0.1", User: "root", Port: 22},
	}
	certs, err := p.CertExpiryWithMachine(context.Background(), machine)
	assert.NoError(t, err)
	assert.Nil(t, certs)
}

func TestCertExpiryWithMachine_ParsesCerts(t *testing.T) {
	client := fake.NewSimpleClientset()
	p := newFakeProvider(client)

	ssh := newMockSSH()
	for _, cp := range certPaths {
		cmd := fmt.Sprintf("openssl x509 -enddate -issuer -noout -in %s 2>/dev/null", cp.path)
		ssh.outputs[cmd] = "notAfter=Mar 11 12:00:00 2027 GMT\nissuer=O = kubernetes, CN = kubernetes-ca"
	}
	p.SetSSHExecutor(ssh)

	machine := provider.Machine{
		Spec: provider.MachineSpec{Host: "10.0.0.1", User: "root", Port: 22},
	}
	certs, err := p.CertExpiryWithMachine(context.Background(), machine)
	require.NoError(t, err)
	assert.Len(t, certs, len(certPaths))
	assert.Equal(t, "apiserver", certs[0].Name)
	assert.Equal(t, "kubernetes-ca", certs[0].Issuer)
	assert.Equal(t, 2027, certs[0].ExpiresAt.Year())
}

func TestCertExpiryWithMachine_SSHFails(t *testing.T) {
	client := fake.NewSimpleClientset()
	p := newFakeProvider(client)

	ssh := newMockSSH()
	for _, cp := range certPaths {
		cmd := fmt.Sprintf("openssl x509 -enddate -issuer -noout -in %s 2>/dev/null", cp.path)
		ssh.errors[cmd] = fmt.Errorf("connection refused")
	}
	p.SetSSHExecutor(ssh)

	machine := provider.Machine{
		Spec: provider.MachineSpec{Host: "10.0.0.1", User: "root", Port: 22},
	}
	certs, err := p.CertExpiryWithMachine(context.Background(), machine)
	assert.NoError(t, err)
	assert.Empty(t, certs)
}

func TestEtcdHealthWithMachine_Healthy(t *testing.T) {
	client := fake.NewSimpleClientset()
	p := newFakeProvider(client)

	ssh := newMockSSH()
	healthCmd := "ETCDCTL_API=3 etcdctl --cacert=/etc/kubernetes/pki/etcd/ca.crt " +
		"--cert=/etc/kubernetes/pki/etcd/server.crt " +
		"--key=/etc/kubernetes/pki/etcd/server.key " +
		"endpoint health --write-out=table 2>/dev/null"
	ssh.outputs[healthCmd] = "https://127.0.0.1:2379 is healthy: true"

	versionCmd := "etcdctl version 2>/dev/null | head -1"
	ssh.outputs[versionCmd] = "etcdctl version: 3.5.12"
	p.SetSSHExecutor(ssh)

	machine := provider.Machine{
		Spec: provider.MachineSpec{Host: "10.0.0.1", User: "root", Port: 22},
	}
	health, err := p.EtcdHealthWithMachine(context.Background(), machine)
	require.NoError(t, err)
	assert.True(t, health.Healthy)
	assert.Equal(t, "3.5.12", health.Version)
}

func TestEtcdHealthWithMachine_Unhealthy(t *testing.T) {
	client := fake.NewSimpleClientset()
	p := newFakeProvider(client)

	ssh := newMockSSH()
	healthCmd := "ETCDCTL_API=3 etcdctl --cacert=/etc/kubernetes/pki/etcd/ca.crt " +
		"--cert=/etc/kubernetes/pki/etcd/server.crt " +
		"--key=/etc/kubernetes/pki/etcd/server.key " +
		"endpoint health --write-out=table 2>/dev/null"
	ssh.errors[healthCmd] = fmt.Errorf("connection refused")
	p.SetSSHExecutor(ssh)

	machine := provider.Machine{
		Spec: provider.MachineSpec{Host: "10.0.0.1", User: "root", Port: 22},
	}
	health, err := p.EtcdHealthWithMachine(context.Background(), machine)
	require.NoError(t, err)
	assert.False(t, health.Healthy)
}

func TestEtcdHealthWithMachine_NoSSH(t *testing.T) {
	client := fake.NewSimpleClientset()
	p := newFakeProvider(client)

	machine := provider.Machine{
		Spec: provider.MachineSpec{Host: "10.0.0.1", User: "root", Port: 22},
	}
	health, err := p.EtcdHealthWithMachine(context.Background(), machine)
	assert.NoError(t, err)
	assert.NotNil(t, health)
}

func TestParseCertDate_ValidFormat(t *testing.T) {
	date, err := ParseCertDate("Mar 11 12:00:00 2027 GMT")
	require.NoError(t, err)
	assert.Equal(t, 2027, date.Year())
	assert.Equal(t, time.March, date.Month())
	assert.Equal(t, 11, date.Day())
}

func TestParseCertDate_ValidFormatWithExtraSpace(t *testing.T) {
	date, err := ParseCertDate("Jan  2 15:04:05 2027 GMT")
	require.NoError(t, err)
	assert.Equal(t, 2027, date.Year())
	assert.Equal(t, time.January, date.Month())
}

func TestParseCertDate_InvalidFormat(t *testing.T) {
	_, err := ParseCertDate("2027-03-11")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to parse")
}

func TestParseCertDate_EmptyString(t *testing.T) {
	_, err := ParseCertDate("")
	assert.Error(t, err)
}

func TestNew_DefaultFactory(t *testing.T) {
	p := New("my-cluster")
	assert.NotNil(t, p)
	assert.Equal(t, "my-cluster", p.clusterName)
	assert.NotNil(t, p.clientFactory)
	assert.Nil(t, p.sshExecutor)
}

func TestNewWithFactory(t *testing.T) {
	called := false
	factory := func(_ []byte) (kubernetes.Interface, error) {
		called = true
		return fake.NewSimpleClientset(readyNode("cp1")), nil
	}
	p := NewWithFactory("custom", factory)
	assert.Equal(t, "custom", p.clusterName)

	_, err := p.ListNodes(context.Background(), []byte("kc"))
	assert.NoError(t, err)
	assert.True(t, called)
}
