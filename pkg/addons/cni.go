package addons

const ciliumVersion = "1.17.4"
const calicoVersion = "3.30.1"

type Cilium struct{}

func (c *Cilium) Name() string { return "cilium" }

func (c *Cilium) InstallCmd() string {
	return "kubectl create -f https://raw.githubusercontent.com/cilium/cilium/v" + ciliumVersion + "/install/kubernetes/quick-install.yaml"
}

func (c *Cilium) WaitCmd() string {
	return "kubectl -n kube-system wait --for=condition=ready pod -l app.kubernetes.io/name=cilium-agent --timeout=120s"
}

type Calico struct{}

func (c *Calico) Name() string { return "calico" }

func (c *Calico) InstallCmd() string {
	return "kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/v" + calicoVersion + "/manifests/calico.yaml"
}

func (c *Calico) WaitCmd() string {
	return "kubectl -n kube-system wait --for=condition=ready pod -l k8s-app=calico-node --timeout=120s"
}
