package app

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	cfg.SetDefaults()
	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func validateConfig(cfg *Config) error {
	if cfg.Policy.Default != "allow" && cfg.Policy.Default != "ask" && cfg.Policy.Default != "deny" {
		return fmt.Errorf("policy.default must be 'allow', 'ask', or 'deny', got %q", cfg.Policy.Default)
	}
	return nil
}