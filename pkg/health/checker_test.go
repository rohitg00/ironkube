package health

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckResult_Fields(t *testing.T) {
	r := CheckResult{
		Name:    "api-server",
		Status:  StatusHealthy,
		Message: "API server is responding",
	}
	assert.Equal(t, "api-server", r.Name)
	assert.Equal(t, StatusHealthy, r.Status)
	assert.Equal(t, "API server is responding", r.Message)
}

func TestStatus_Constants(t *testing.T) {
	assert.Equal(t, Status("healthy"), StatusHealthy)
	assert.Equal(t, Status("unhealthy"), StatusUnhealthy)
	assert.Equal(t, Status("unknown"), StatusUnknown)
}

func TestClusterHealth_AllHealthy(t *testing.T) {
	ch := &ClusterHealth{
		ClusterName: "test-cluster",
		Results: []CheckResult{
			{Name: "api-server", Status: StatusHealthy, Message: "ok"},
			{Name: "etcd", Status: StatusHealthy, Message: "ok"},
			{Name: "scheduler", Status: StatusHealthy, Message: "ok"},
		},
	}
	assert.True(t, ch.IsHealthy())
}

func TestClusterHealth_OneUnhealthy(t *testing.T) {
	ch := &ClusterHealth{
		ClusterName: "test-cluster",
		Results: []CheckResult{
			{Name: "api-server", Status: StatusHealthy, Message: "ok"},
			{Name: "etcd", Status: StatusUnhealthy, Message: "connection refused"},
			{Name: "scheduler", Status: StatusHealthy, Message: "ok"},
		},
	}
	assert.False(t, ch.IsHealthy())
}

func TestClusterHealth_Summary(t *testing.T) {
	ch := &ClusterHealth{
		ClusterName: "test-cluster",
		Results: []CheckResult{
			{Name: "api-server", Status: StatusHealthy, Message: "ok"},
			{Name: "etcd", Status: StatusHealthy, Message: "ok"},
			{Name: "scheduler", Status: StatusUnhealthy, Message: "not running"},
		},
	}
	s := ch.Summary()
	assert.Equal(t, 2, s.Healthy)
	assert.Equal(t, 1, s.Unhealthy)
	assert.Equal(t, 0, s.Unknown)
	assert.Equal(t, 3, s.Total)
}

func TestClusterHealth_EmptyResults(t *testing.T) {
	ch := &ClusterHealth{
		ClusterName: "empty",
		Results:     []CheckResult{},
	}
	assert.True(t, ch.IsHealthy())
	s := ch.Summary()
	assert.Equal(t, 0, s.Total)
}

func TestClusterHealth_SummaryWithUnknown(t *testing.T) {
	ch := &ClusterHealth{
		ClusterName: "mixed",
		Results: []CheckResult{
			{Name: "api-server", Status: StatusHealthy, Message: "ok"},
			{Name: "etcd", Status: StatusUnknown, Message: "timeout"},
			{Name: "scheduler", Status: StatusUnhealthy, Message: "crashed"},
		},
	}
	assert.False(t, ch.IsHealthy())
	s := ch.Summary()
	assert.Equal(t, 1, s.Healthy)
	assert.Equal(t, 1, s.Unhealthy)
	assert.Equal(t, 1, s.Unknown)
	assert.Equal(t, 3, s.Total)
}
