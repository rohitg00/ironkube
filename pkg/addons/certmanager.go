package addons

const certManagerVersion = "1.17.2"

type CertManager struct{}

func (c *CertManager) Name() string { return "cert-manager" }

func (c *CertManager) InstallCmd() string {
	return "kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v" + certManagerVersion + "/cert-manager.yaml"
}

func (c *CertManager) WaitCmd() string {
	return "kubectl -n cert-manager wait --for=condition=ready pod -l app.kubernetes.io/instance=cert-manager --timeout=120s"
}
