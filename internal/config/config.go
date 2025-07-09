package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	LoadBalancer LoadBalancerConfig `yaml:"loadbalancer"`
	Backends     []BackendConfig    `yaml:"backends"`
	HealthCheck  HealthCheckConfig  `yaml:"healthcheck"`
}

// LoadBalancerConfig contains load balancer specific settings
type LoadBalancerConfig struct {
	ListenAddress string `yaml:"listen_address"`
	Algorithm     string `yaml:"algorithm"`
}

// BackendConfig represents a backend server configuration
type BackendConfig struct {
	Address string `yaml:"address"`
	Port    int    `yaml:"port"`
}

// HealthCheckConfig contains health check settings
type HealthCheckConfig struct {
	Interval time.Duration `yaml:"interval"`
	Timeout  time.Duration `yaml:"timeout"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// GetDefaultConfig returns a default configuration
func GetDefaultConfig() *Config {
	return &Config{
		LoadBalancer: LoadBalancerConfig{
			ListenAddress: ":8080",
			Algorithm:     "round_robin",
		},
		Backends: []BackendConfig{
			{Address: "localhost", Port: 8081},
			{Address: "localhost", Port: 8082},
		},
		HealthCheck: HealthCheckConfig{
			Interval: 30 * time.Second,
			Timeout:  5 * time.Second,
		},
	}
}
