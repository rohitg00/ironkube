package config

import (
	"fmt"
	"os"

	"sigs.k8s.io/yaml"
)

func Load(path string) (*ClusterConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg ClusterConfig
	if err := yaml.UnmarshalStrict(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	ApplyDefaults(&cfg)

	return &cfg, nil
}
