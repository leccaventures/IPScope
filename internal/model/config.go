package model

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

const (
	DefaultHost   = "127.0.0.1"
	DefaultPort   = 9100
	DefaultPrefix = "ipscope"
)

type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Metrics MetricsConfig `yaml:"metrics"`
	Nodes   []NodeConfig  `yaml:"nodes"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type MetricsConfig struct {
	Prefix string `yaml:"prefix"`
}

type NodeConfig struct {
	Name     string `yaml:"name"`
	Endpoint string `yaml:"endpoint"`
}

func (c *Config) ApplyDefaults() {
	if strings.TrimSpace(c.Server.Host) == "" {
		c.Server.Host = DefaultHost
	}

	if c.Server.Port == 0 {
		c.Server.Port = DefaultPort
	}

	if strings.TrimSpace(c.Metrics.Prefix) == "" {
		c.Metrics.Prefix = DefaultPrefix
	}
}

func (c Config) Validate() error {
	host := strings.TrimSpace(c.Server.Host)
	if host == "" {
		return fmt.Errorf("server.host is required")
	}

	if net.ParseIP(host) == nil {
		return fmt.Errorf("server.host must be a valid IP address")
	}

	if host != "127.0.0.1" && host != "0.0.0.0" {
		return fmt.Errorf("server.host must be either 127.0.0.1 or 0.0.0.0")
	}

	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535")
	}

	if len(c.Nodes) == 0 {
		return fmt.Errorf("at least one node must be configured")
	}

	for i, node := range c.Nodes {
		if strings.TrimSpace(node.Name) == "" {
			return fmt.Errorf("nodes[%s].name is required", strconv.Itoa(i))
		}

		if net.ParseIP(strings.TrimSpace(node.Endpoint)) == nil {
			return fmt.Errorf("nodes[%s].endpoint must be a valid IP address", strconv.Itoa(i))
		}
	}

	return nil
}
