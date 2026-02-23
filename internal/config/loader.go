package config

import (
	"fmt"
	"os"

	"ipscope/internal/model"

	"gopkg.in/yaml.v3"
)

func Load(path string) (model.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return model.Config{}, fmt.Errorf("read config file: %w", err)
	}

	var cfg model.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return model.Config{}, fmt.Errorf("parse yaml config: %w", err)
	}

	cfg.ApplyDefaults()
	if err := cfg.Validate(); err != nil {
		return model.Config{}, fmt.Errorf("validate config: %w", err)
	}

	return cfg, nil
}
