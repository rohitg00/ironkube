package kubeconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/yaml"
)

type Config struct {
	APIVersion     string    `json:"apiVersion"`
	Kind           string    `json:"kind"`
	Clusters       []Cluster `json:"clusters"`
	Contexts       []Context `json:"contexts"`
	Users          []User    `json:"users"`
	CurrentContext string    `json:"current-context"`
}

type Cluster struct {
	Name    string      `json:"name"`
	Cluster ClusterData `json:"cluster"`
}

type ClusterData struct {
	Server                   string `json:"server"`
	CertificateAuthorityData string `json:"certificate-authority-data,omitempty"`
}

type Context struct {
	Name    string      `json:"name"`
	Context ContextData `json:"context"`
}

type ContextData struct {
	Cluster string `json:"cluster"`
	User    string `json:"user"`
}

type User struct {
	Name string   `json:"name"`
	User UserData `json:"user"`
}

type UserData struct {
	ClientCertificateData string `json:"client-certificate-data,omitempty"`
	ClientKeyData         string `json:"client-key-data,omitempty"`
	Token                 string `json:"token,omitempty"`
}

func RewriteServer(data []byte, host string, clusterName string) ([]byte, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig: %w", err)
	}

	for i := range cfg.Clusters {
		server := cfg.Clusters[i].Cluster.Server
		server = strings.Replace(server, "127.0.0.1", host, 1)
		server = strings.Replace(server, "localhost", host, 1)
		cfg.Clusters[i].Cluster.Server = server
		cfg.Clusters[i].Name = clusterName
	}

	for i := range cfg.Contexts {
		cfg.Contexts[i].Name = clusterName
		cfg.Contexts[i].Context.Cluster = clusterName
		cfg.Contexts[i].Context.User = clusterName + "-admin"
	}

	for i := range cfg.Users {
		cfg.Users[i].Name = clusterName + "-admin"
	}

	cfg.CurrentContext = clusterName

	out, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal kubeconfig: %w", err)
	}
	return out, nil
}

func SaveToFile(data []byte, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write kubeconfig: %w", err)
	}
	return nil
}

func Merge(existingPath string, newData []byte) error {
	var newCfg Config
	if err := yaml.Unmarshal(newData, &newCfg); err != nil {
		return fmt.Errorf("failed to parse new kubeconfig: %w", err)
	}

	existingData, err := os.ReadFile(existingPath)
	if err != nil {
		if os.IsNotExist(err) {
			return SaveToFile(newData, existingPath)
		}
		return fmt.Errorf("failed to read existing kubeconfig: %w", err)
	}

	var existingCfg Config
	if err := yaml.Unmarshal(existingData, &existingCfg); err != nil {
		return fmt.Errorf("failed to parse existing kubeconfig: %w", err)
	}

	for _, c := range newCfg.Clusters {
		replaced := false
		for i, existing := range existingCfg.Clusters {
			if existing.Name == c.Name {
				existingCfg.Clusters[i] = c
				replaced = true
				break
			}
		}
		if !replaced {
			existingCfg.Clusters = append(existingCfg.Clusters, c)
		}
	}

	for _, c := range newCfg.Contexts {
		replaced := false
		for i, existing := range existingCfg.Contexts {
			if existing.Name == c.Name {
				existingCfg.Contexts[i] = c
				replaced = true
				break
			}
		}
		if !replaced {
			existingCfg.Contexts = append(existingCfg.Contexts, c)
		}
	}

	for _, u := range newCfg.Users {
		replaced := false
		for i, existing := range existingCfg.Users {
			if existing.Name == u.Name {
				existingCfg.Users[i] = u
				replaced = true
				break
			}
		}
		if !replaced {
			existingCfg.Users = append(existingCfg.Users, u)
		}
	}

	existingCfg.CurrentContext = newCfg.CurrentContext

	out, err := yaml.Marshal(existingCfg)
	if err != nil {
		return fmt.Errorf("failed to marshal merged kubeconfig: %w", err)
	}

	return SaveToFile(out, existingPath)
}
