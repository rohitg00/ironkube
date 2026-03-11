package addons

type Monitoring struct{}

func (m *Monitoring) Name() string { return "monitoring" }

func (m *Monitoring) InstallCmd() string {
	return "kubectl apply -f https://raw.githubusercontent.com/prometheus-operator/kube-prometheus/main/manifests/setup/ && kubectl apply -f https://raw.githubusercontent.com/prometheus-operator/kube-prometheus/main/manifests/"
}

func (m *Monitoring) WaitCmd() string {
	return "kubectl -n monitoring wait --for=condition=ready pod -l app.kubernetes.io/name=prometheus --timeout=180s"
}
