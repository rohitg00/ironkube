package k8s

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/rohitg00/ironkube/pkg/provider"
)

func ListNodes(ctx context.Context, client kubernetes.Interface) ([]provider.NodeHealth, error) {
	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing nodes: %w", err)
	}

	var result []provider.NodeHealth
	for _, node := range nodes.Items {
		nh := provider.NodeHealth{
			Name:    node.Name,
			Version: node.Status.NodeInfo.KubeletVersion,
		}
		for _, cond := range node.Status.Conditions {
			if cond.Type == corev1.NodeReady {
				nh.Ready = cond.Status == corev1.ConditionTrue
			}
		}
		var roles []string
		for label := range node.Labels {
			if label == "node-role.kubernetes.io/control-plane" {
				roles = append(roles, "control-plane")
			} else if label == "node-role.kubernetes.io/worker" {
				roles = append(roles, "worker")
			}
		}
		if len(roles) > 0 {
			nh.Roles = roles[0]
		}
		result = append(result, nh)
	}
	return result, nil
}

func IsNodeReady(ctx context.Context, client kubernetes.Interface, nodeName string) (bool, error) {
	node, err := client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	for _, cond := range node.Status.Conditions {
		if cond.Type == corev1.NodeReady {
			return cond.Status == corev1.ConditionTrue, nil
		}
	}
	return false, nil
}

func WaitForNodes(ctx context.Context, client kubernetes.Interface, expected int, timeout time.Duration) error {
	deadline := time.After(timeout)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	nodes, err := ListNodes(ctx, client)
	if err == nil {
		ready := 0
		for _, n := range nodes {
			if n.Ready {
				ready++
			}
		}
		if ready >= expected {
			return nil
		}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return fmt.Errorf("timed out waiting for %d nodes", expected)
		case <-ticker.C:
			nodes, err := ListNodes(ctx, client)
			if err != nil {
				continue
			}
			ready := 0
			for _, n := range nodes {
				if n.Ready {
					ready++
				}
			}
			if ready >= expected {
				return nil
			}
		}
	}
}

func DrainNode(ctx context.Context, client kubernetes.Interface, nodeName string, opts provider.DrainOptions) error {
	node, err := client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("getting node: %w", err)
	}

	node.Spec.Unschedulable = true
	_, err = client.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("cordoning node: %w", err)
	}

	pods, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: "spec.nodeName=" + nodeName,
	})
	if err != nil {
		return fmt.Errorf("listing pods: %w", err)
	}

	for _, pod := range pods.Items {
		if isDaemonSetPod(&pod) && opts.IgnoreDaemonSets {
			continue
		}
		if isMirrorPod(&pod) {
			continue
		}
		gracePeriod := int64(opts.GracePeriod)
		err := client.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{
			GracePeriodSeconds: &gracePeriod,
		})
		if err != nil {
			if !opts.Force {
				return fmt.Errorf("evicting pod %s/%s: %w", pod.Namespace, pod.Name, err)
			}
		}
	}
	return nil
}

func UncordonNode(ctx context.Context, client kubernetes.Interface, nodeName string) error {
	node, err := client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("getting node: %w", err)
	}
	node.Spec.Unschedulable = false
	_, err = client.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("uncordoning node: %w", err)
	}
	return nil
}

func isDaemonSetPod(pod *corev1.Pod) bool {
	for _, ref := range pod.OwnerReferences {
		if ref.Kind == "DaemonSet" {
			return true
		}
	}
	return false
}

func isMirrorPod(pod *corev1.Pod) bool {
	_, exists := pod.Annotations[corev1.MirrorPodAnnotationKey]
	return exists
}
