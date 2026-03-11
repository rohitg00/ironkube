package k8s

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/rohitg00/ironkube/pkg/provider"
)

func ClusterHealth(ctx context.Context, client kubernetes.Interface, clusterName string) (*provider.HealthReport, error) {
	report := &provider.HealthReport{ClusterName: clusterName}

	nodes, err := ListNodes(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("checking nodes: %w", err)
	}
	report.Nodes = nodes

	_, err = client.CoreV1().Pods("kube-system").List(ctx, metav1.ListOptions{})
	if err != nil {
		return report, nil
	}

	return report, nil
}
