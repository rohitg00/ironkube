package k8s

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClusterHealth_AllHealthy(t *testing.T) {
	client := fake.NewSimpleClientset(
		readyNode("cp1"),
		readyNode("worker1"),
		readyNode("worker2"),
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "kube-apiserver", Namespace: "kube-system"},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning},
		},
	)
	report, err := ClusterHealth(context.Background(), client, "test-cluster")
	require.NoError(t, err)
	assert.Equal(t, "test-cluster", report.ClusterName)
	assert.Len(t, report.Nodes, 3)
	assert.True(t, report.AllHealthy())
	assert.Equal(t, 3, report.ReadyCount())
	assert.Equal(t, 3, report.TotalCount())
}

func TestClusterHealth_SomeUnhealthy(t *testing.T) {
	client := fake.NewSimpleClientset(
		readyNode("cp1"),
		notReadyNode("worker1"),
	)
	report, err := ClusterHealth(context.Background(), client, "prod")
	require.NoError(t, err)
	assert.False(t, report.AllHealthy())
	assert.Equal(t, 1, report.ReadyCount())
	assert.Equal(t, 2, report.TotalCount())
}

func TestClusterHealth_EmptyCluster(t *testing.T) {
	client := fake.NewSimpleClientset()
	report, err := ClusterHealth(context.Background(), client, "empty")
	require.NoError(t, err)
	assert.Equal(t, "empty", report.ClusterName)
	assert.Empty(t, report.Nodes)
	assert.True(t, report.AllHealthy())
	assert.Equal(t, 0, report.ReadyCount())
}

func TestClusterHealth_ClusterNamePreserved(t *testing.T) {
	client := fake.NewSimpleClientset(readyNode("cp1"))
	report, err := ClusterHealth(context.Background(), client, "my-cluster")
	require.NoError(t, err)
	assert.Equal(t, "my-cluster", report.ClusterName)
}

func TestClusterHealth_NodeDetailsPreserved(t *testing.T) {
	client := fake.NewSimpleClientset(readyNode("cp1"))
	report, err := ClusterHealth(context.Background(), client, "test")
	require.NoError(t, err)
	require.Len(t, report.Nodes, 1)
	assert.Equal(t, "cp1", report.Nodes[0].Name)
	assert.Equal(t, "v1.34.3", report.Nodes[0].Version)
	assert.Equal(t, "control-plane", report.Nodes[0].Roles)
	assert.True(t, report.Nodes[0].Ready)
}
