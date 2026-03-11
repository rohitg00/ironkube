package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAudit_MinimalProfile_AllPass(t *testing.T) {
	auditor := NewAuditor()
	result, err := auditor.Audit("minimal", map[string][]string{})
	require.NoError(t, err)
	assert.Equal(t, "minimal", result.Profile)
	assert.Equal(t, 0, result.Failed)
	assert.Empty(t, result.Findings)
	assert.Equal(t, float64(100), result.Score)
}

func TestAudit_CIS_AllFlagsPresent(t *testing.T) {
	auditor := NewAuditor()
	flags := map[string][]string{
		"api-server": {
			"--anonymous-auth=false",
			"--audit-log-path=/var/log/kubernetes/audit.log",
			"--audit-log-maxage=30",
			"--audit-log-maxbackup=10",
			"--audit-log-maxsize=100",
			"--authorization-mode=Node,RBAC",
			"--profiling=false",
			"--service-account-lookup=true",
		},
		"kubelet": {
			"--protect-kernel-defaults=true",
			"--read-only-port=0",
			"--streaming-connection-idle-timeout=5m",
			"--make-iptables-util-chains=true",
		},
		"etcd": {
			"--client-cert-auth=true",
			"--peer-client-cert-auth=true",
			"--auto-tls=false",
		},
	}
	result, err := auditor.Audit("cis", flags)
	require.NoError(t, err)
	assert.Equal(t, float64(100), result.Score)
	assert.Equal(t, 15, result.Passed)
	assert.Equal(t, 0, result.Failed)
	assert.Empty(t, result.Findings)
}

func TestAudit_CIS_MissingFlags(t *testing.T) {
	auditor := NewAuditor()
	flags := map[string][]string{
		"api-server": {
			"--anonymous-auth=false",
		},
		"kubelet": {},
		"etcd":    {},
	}
	result, err := auditor.Audit("cis", flags)
	require.NoError(t, err)
	assert.Greater(t, result.Failed, 0)
	assert.NotEmpty(t, result.Findings)
	assert.Less(t, result.Score, float64(100))
}

func TestAudit_CIS_MissingSingleFlag(t *testing.T) {
	auditor := NewAuditor()
	flags := map[string][]string{
		"api-server": {
			"--anonymous-auth=false",
			"--audit-log-path=/var/log/kubernetes/audit.log",
			"--audit-log-maxage=30",
			"--audit-log-maxbackup=10",
			"--audit-log-maxsize=100",
			"--authorization-mode=Node,RBAC",
			"--profiling=false",
		},
		"kubelet": {
			"--protect-kernel-defaults=true",
			"--read-only-port=0",
			"--streaming-connection-idle-timeout=5m",
			"--make-iptables-util-chains=true",
		},
		"etcd": {
			"--client-cert-auth=true",
			"--peer-client-cert-auth=true",
			"--auto-tls=false",
		},
	}
	result, err := auditor.Audit("cis", flags)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Failed)
	assert.Equal(t, 14, result.Passed)
	require.Len(t, result.Findings, 1)
	assert.Contains(t, result.Findings[0].Description, "--service-account-lookup=true")
	assert.Equal(t, "api-server", result.Findings[0].Category)
	assert.Equal(t, "critical", result.Findings[0].Severity)
}

func TestAudit_Hardened_PartialFlags(t *testing.T) {
	auditor := NewAuditor()
	flags := map[string][]string{
		"api-server": {
			"--anonymous-auth=false",
			"--audit-log-path=/var/log/kubernetes/audit.log",
		},
		"kubelet": {
			"--protect-kernel-defaults=true",
		},
		"etcd": {
			"--client-cert-auth=true",
		},
	}
	result, err := auditor.Audit("hardened", flags)
	require.NoError(t, err)
	assert.Greater(t, result.Failed, 0)
	assert.Greater(t, result.Passed, 0)
	assert.Greater(t, result.Score, float64(0))
	assert.Less(t, result.Score, float64(100))
}

func TestAudit_UnknownProfile(t *testing.T) {
	auditor := NewAuditor()
	_, err := auditor.Audit("nonexistent", map[string][]string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown profile")
}

func TestAudit_ScoreCalculation(t *testing.T) {
	auditor := NewAuditor()
	flags := map[string][]string{
		"api-server": {
			"--anonymous-auth=false",
			"--audit-log-path=/var/log/kubernetes/audit.log",
			"--audit-log-maxage=30",
			"--audit-log-maxbackup=10",
			"--audit-log-maxsize=100",
		},
		"kubelet": {},
		"etcd":    {},
	}
	result, err := auditor.Audit("cis", flags)
	require.NoError(t, err)
	assert.Equal(t, 5, result.Passed)
	assert.Equal(t, 10, result.Failed)
	expectedScore := float64(5) / float64(15) * 100
	assert.InDelta(t, expectedScore, result.Score, 0.01)
}

func TestAudit_EmptyCurrentFlags_AllFail(t *testing.T) {
	auditor := NewAuditor()
	result, err := auditor.Audit("cis", map[string][]string{})
	require.NoError(t, err)
	assert.Equal(t, 0, result.Passed)
	assert.Equal(t, 15, result.Failed)
	assert.Equal(t, float64(0), result.Score)
	assert.Len(t, result.Findings, 15)
}

func TestAudit_FindingsHaveCorrectSeverity(t *testing.T) {
	auditor := NewAuditor()
	result, err := auditor.Audit("cis", map[string][]string{})
	require.NoError(t, err)
	for _, f := range result.Findings {
		assert.Equal(t, "critical", f.Severity)
		assert.NotEmpty(t, f.Category)
		assert.NotEmpty(t, f.Description)
		assert.NotEmpty(t, f.Remediation)
	}
}

func TestAudit_FindingsHaveCorrectCategories(t *testing.T) {
	auditor := NewAuditor()
	result, err := auditor.Audit("cis", map[string][]string{})
	require.NoError(t, err)

	categories := make(map[string]int)
	for _, f := range result.Findings {
		categories[f.Category]++
	}
	assert.Equal(t, 8, categories["api-server"])
	assert.Equal(t, 4, categories["kubelet"])
	assert.Equal(t, 3, categories["etcd"])
}

func TestNewAuditor(t *testing.T) {
	auditor := NewAuditor()
	assert.NotNil(t, auditor)
}
