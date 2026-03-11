package security

import (
	"fmt"
	"strings"
)

type AuditFinding struct {
	Severity    string
	Category    string
	Description string
	Remediation string
}

type AuditResult struct {
	Profile  string
	Findings []AuditFinding
	Passed   int
	Failed   int
	Score    float64
}

type Auditor struct{}

func NewAuditor() *Auditor {
	return &Auditor{}
}

func (a *Auditor) Audit(profileName string, currentFlags map[string][]string) (*AuditResult, error) {
	profile, err := GetProfile(profileName)
	if err != nil {
		return nil, err
	}

	result := &AuditResult{
		Profile: profileName,
	}

	a.checkFlags(result, "api-server", profile.APIServerFlags(), currentFlags["api-server"])
	a.checkFlags(result, "kubelet", profile.KubeletFlags(), currentFlags["kubelet"])
	a.checkFlags(result, "etcd", profile.EtcdFlags(), currentFlags["etcd"])

	total := result.Passed + result.Failed
	if total > 0 {
		result.Score = float64(result.Passed) / float64(total) * 100
	} else {
		result.Score = 100
	}

	return result, nil
}

func (a *Auditor) checkFlags(result *AuditResult, category string, required []string, current []string) {
	currentSet := make(map[string]bool, len(current))
	for _, f := range current {
		currentSet[f] = true
	}

	for _, flag := range required {
		if currentSet[flag] {
			result.Passed++
		} else {
			result.Failed++
			flagName := extractFlagName(flag)
			result.Findings = append(result.Findings, AuditFinding{
				Severity:    "critical",
				Category:    category,
				Description: fmt.Sprintf("Missing required flag: %s", flag),
				Remediation: fmt.Sprintf("Add %s to %s configuration", flagName, category),
			})
		}
	}
}

func extractFlagName(flag string) string {
	parts := strings.SplitN(flag, "=", 2)
	return parts[0]
}
