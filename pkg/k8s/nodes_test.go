package k8s

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/rohitg00/ironkube/pkg/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestListNodes_MixedReadiness(t *testing.T) {
	client := fake.NewSimpleClientset(readyNode("cp1"), readyNode("cp2"), notReadyNode("worker1"))
	nodes, err := ListNodes(context.Background(), client)
	require.NoError(t, err)
	assert.Len(t, nodes, 3)

	readyCount := 0
	for _, n := range nodes {
		if n.Ready {
			readyCount++
		}
	}
	assert.Equal(t, 2, readyCount)
}

func TestListNodes_Empty(t *testing.T) {
	client := fake.NewSimpleClientset()
	nodes, err := ListNodes(context.Background(), client)
	require.NoError(t, err)
	assert.Empty(t, nodes)
}

func TestListNodes_RoleDetection(t *testing.T) {
	client := fake.NewSimpleClientset(readyNode("cp1"))
	nodes, err := ListNodes(context.Background(), client)
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, "control-plane", nodes[0].Roles)
}

func TestListNodes_WorkerRole(t *testing.T) {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "worker1",
			Labels: map[string]string{"node-role.kubernetes.io/worker": ""},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			},
			NodeInfo: corev1.NodeSystemInfo{KubeletVersion: "v1.34.3"},
		},
	}
	client := fake.NewSimpleClientset(node)
	nodes, err := ListNodes(context.Background(), client)
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, "worker", nodes[0].Roles)
}

func TestListNodes_VersionCaptured(t *testing.T) {
	client := fake.NewSimpleClientset(readyNode("cp1"))
	nodes, err := ListNodes(context.Background(), client)
	require.NoError(t, err)
	assert.Equal(t, "v1.34.3", nodes[0].Version)
}

func TestIsNodeReady_True(t *testing.T) {
	client := fake.NewSimpleClientset(readyNode("cp1"))
	ready, err := IsNodeReady(context.Background(), client, "cp1")
	require.NoError(t, err)
	assert.True(t, ready)
}

func TestIsNodeReady_False(t *testing.T) {
	client := fake.NewSimpleClientset(notReadyNode("worker1"))
	ready, err := IsNodeReady(context.Background(), client, "worker1")
	require.NoError(t, err)
	assert.False(t, ready)
}

func TestIsNodeReady_NotFound(t *testing.T) {
	client := fake.NewSimpleClientset()
	_, err := IsNodeReady(context.Background(), client, "nonexistent")
	assert.Error(t, err)
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

	err := DrainNode(context.Background(), client, "worker1", provider.DrainOptions{
		GracePeriod:      30,
		IgnoreDaemonSets: true,
		Force:            false,
	})
	require.NoError(t, err)

	updatedNode, err := client.CoreV1().Nodes().Get(context.Background(), "worker1", metav1.GetOptions{})
	require.NoError(t, err)
	assert.True(t, updatedNode.Spec.Unschedulable)

	pods, err := client.CoreV1().Pods("default").List(context.Background(), metav1.ListOptions{})
	require.NoError(t, err)
	assert.Empty(t, pods.Items)
}

func TestDrainNode_SkipsDaemonSetPods(t *testing.T) {
	node := readyNode("worker1")
	dsPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ds-pod",
			Namespace: "kube-system",
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "DaemonSet", Name: "kube-proxy"},
			},
		},
		Spec: corev1.PodSpec{NodeName: "worker1"},
	}
	client := fake.NewSimpleClientset(node, dsPod)

	err := DrainNode(context.Background(), client, "worker1", provider.DrainOptions{
		IgnoreDaemonSets: true,
	})
	require.NoError(t, err)

	pods, err := client.CoreV1().Pods("kube-system").List(context.Background(), metav1.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
}

func TestDrainNode_SkipsMirrorPods(t *testing.T) {
	node := readyNode("cp1")
	mirrorPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kube-apiserver-cp1",
			Namespace: "kube-system",
			Annotations: map[string]string{
				corev1.MirrorPodAnnotationKey: "some-hash",
			},
		},
		Spec: corev1.PodSpec{NodeName: "cp1"},
	}
	client := fake.NewSimpleClientset(node, mirrorPod)

	err := DrainNode(context.Background(), client, "cp1", provider.DrainOptions{})
	require.NoError(t, err)

	pods, err := client.CoreV1().Pods("kube-system").List(context.Background(), metav1.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
}

func TestDrainNode_NodeNotFound(t *testing.T) {
	client := fake.NewSimpleClientset()
	err := DrainNode(context.Background(), client, "nonexistent", provider.DrainOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "getting node")
}

func TestUncordonNode(t *testing.T) {
	node := readyNode("worker1")
	node.Spec.Unschedulable = true
	client := fake.NewSimpleClientset(node)

	err := UncordonNode(context.Background(), client, "worker1")
	require.NoError(t, err)

	updatedNode, err := client.CoreV1().Nodes().Get(context.Background(), "worker1", metav1.GetOptions{})
	require.NoError(t, err)
	assert.False(t, updatedNode.Spec.Unschedulable)
}

func TestUncordonNode_NotFound(t *testing.T) {
	client := fake.NewSimpleClientset()
	err := UncordonNode(context.Background(), client, "nonexistent")
	assert.Error(t, err)
}

func TestIsDaemonSetPod(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "DaemonSet", Name: "kube-proxy"},
			},
		},
	}
	assert.True(t, isDaemonSetPod(pod))

	regularPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "ReplicaSet", Name: "app-rs"},
			},
		},
	}
	assert.False(t, isDaemonSetPod(regularPod))
}

func TestIsMirrorPod(t *testing.T) {
	mirror := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				corev1.MirrorPodAnnotationKey: "hash-value",
			},
		},
	}
	assert.True(t, isMirrorPod(mirror))

	regular := &corev1.Pod{}
	assert.False(t, isMirrorPod(regular))
}

func TestWaitForNodes_AlreadyReady(t *testing.T) {
	client := fake.NewSimpleClientset(readyNode("cp1"), readyNode("worker1"))
	err := WaitForNodes(context.Background(), client, 2, 10*time.Second)
	assert.NoError(t, err)
}

func TestWaitForNodes_ContextCancelled(t *testing.T) {
	client := fake.NewSimpleClientset(readyNode("cp1"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := WaitForNodes(ctx, client, 5, 30*time.Second)
	assert.Error(t, err)
}
