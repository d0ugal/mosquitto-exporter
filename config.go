package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/d0ugal/promexporter/config"
	"gopkg.in/yaml.v3"
)

// MosquittoExporterConfig extends the base configuration with Mosquito-specific settings
type MosquittoExporterConfig struct {
	config.BaseConfig

	Mosquito MosquittoConfig `yaml:"mosquito"`
}

// MosquittoConfig holds Mosquito broker connection settings
type MosquittoConfig struct {
	BrokerEndpoint string                 `yaml:"broker_endpoint"`
	Username       string                 `yaml:"username"`
	Password       config.SensitiveString `yaml:"password"`
	ClientID       string                 `yaml:"client_id"`
	TLS            TLSConfig              `yaml:"tls"`
}

// TLSConfig holds TLS/SSL settings
type TLSConfig struct {
	Enabled            bool   `yaml:"enabled"`
	CertFile           string `yaml:"cert_file"`
	KeyFile            string `yaml:"key_file"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify"`
}

// GetDisplayConfig returns the configuration for display in the web UI
func (c *MosquittoExporterConfig) GetDisplayConfig() map[string]interface{} {
	cfg := c.BaseConfig.GetDisplayConfig()
	cfg["Mosquito Broker"] = c.Mosquito.BrokerEndpoint
	cfg["MQTT Username"] = c.Mosquito.Username
	cfg["MQTT Client ID"] = c.Mosquito.ClientID

	cfg["TLS Enabled"] = c.Mosquito.TLS.Enabled
	if c.Mosquito.TLS.Enabled {
		cfg["TLS Certificate"] = c.Mosquito.TLS.CertFile
		cfg["TLS Key File"] = c.Mosquito.TLS.KeyFile
		cfg["TLS Skip Verify"] = c.Mosquito.TLS.InsecureSkipVerify
	}

	return cfg
}

// LoadConfig loads configuration from an optional YAML file, then overlays environment variables.
func LoadConfig(configPath string) (*MosquittoExporterConfig, error) {
	var cfg MosquittoExporterConfig

	// Try to load from YAML (optional — silently skip if file not found)
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err == nil {
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
			}
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
		}
	}

	// Always apply environment variable overrides
	if err := config.ApplyGenericEnvVars(&cfg.BaseConfig); err != nil {
		return nil, fmt.Errorf("failed to apply generic environment variables: %w", err)
	}

	if err := applyMosquittoEnvVars(&cfg); err != nil {
		return nil, fmt.Errorf("failed to apply environment overrides: %w", err)
	}

	setDefaults(&cfg)

	return &cfg, nil
}

// applyMosquittoEnvVars applies Mosquito-specific environment variables
func applyMosquittoEnvVars(cfg *MosquittoExporterConfig) error {
	// Broker endpoint - support both new and legacy env var names
	if endpoint := getEnv("MOSQUITO_BROKER_ENDPOINT", "BROKER_ENDPOINT"); endpoint != "" {
		cfg.Mosquito.BrokerEndpoint = endpoint
	}

	// Username - support both new and legacy env var names
	if username := getEnv("MOSQUITO_USERNAME", "MQTT_USER"); username != "" {
		cfg.Mosquito.Username = username
	}

	// Password - support both new and legacy env var names
	if password := getEnv("MOSQUITO_PASSWORD", "MQTT_PASS"); password != "" {
		cfg.Mosquito.Password = config.NewSensitiveString(password)
	}

	// Client ID - support both new and legacy env var names
	if clientID := getEnv("MOSQUITO_CLIENT_ID", "MQTT_CLIENT_ID"); clientID != "" {
		cfg.Mosquito.ClientID = clientID
	}

	// Explicit enable TLS - support both new and legacy env var names
	if tlsEnabled := getEnv("MOSQUITO_TLS_ENABLED", "MQTT_TLS_ENABLED"); tlsEnabled != "" {
		if val, err := strconv.ParseBool(tlsEnabled); err == nil {
			cfg.Mosquito.TLS.Enabled = val
		}
	}

	// TLS settings - support both new and legacy env var names
	if certFile := getEnv("MOSQUITO_TLS_CERT_FILE", "MQTT_CERT"); certFile != "" {
		cfg.Mosquito.TLS.CertFile = certFile
		cfg.Mosquito.TLS.Enabled = true
	}

	if keyFile := getEnv("MOSQUITO_TLS_KEY_FILE", "MQTT_KEY"); keyFile != "" {
		cfg.Mosquito.TLS.KeyFile = keyFile
		cfg.Mosquito.TLS.Enabled = true
	}

	if skipVerify := os.Getenv("MOSQUITO_TLS_INSECURE_SKIP_VERIFY"); skipVerify != "" {
		if val, err := strconv.ParseBool(skipVerify); err == nil {
			cfg.Mosquito.TLS.InsecureSkipVerify = val
			cfg.Mosquito.TLS.Enabled = true
		}
	}

	// Server bind address - legacy BIND_ADDRESS support
	if bindAddress := os.Getenv("BIND_ADDRESS"); bindAddress != "" {
		// Parse bind address (format: host:port)
		host, port := parseBindAddress(bindAddress)
		if host != "" {
			cfg.Server.Host = host
		}

		if port != 0 {
			cfg.Server.Port = port
		}
	}

	return nil
}

// setDefaults sets default values for unconfigured options
func setDefaults(cfg *MosquittoExporterConfig) {
	// Mosquito defaults
	if cfg.Mosquito.BrokerEndpoint == "" {
		cfg.Mosquito.BrokerEndpoint = "tcp://127.0.0.1:1883"
	}

	// Server defaults (maintain backward compatibility with port 9234)
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 9234
	}

	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}

	// Enable web UI and health endpoint by default
	if cfg.Server.EnableWebUI == nil {
		enabled := true
		cfg.Server.EnableWebUI = &enabled
	}

	if cfg.Server.EnableHealth == nil {
		enabled := true
		cfg.Server.EnableHealth = &enabled
	}

	// Logging defaults
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}

	if cfg.Logging.Format == "" {
		cfg.Logging.Format = "json"
	}
}

// getEnv gets environment variable with fallback to legacy name
func getEnv(newName, legacyName string) string {
	if val := os.Getenv(newName); val != "" {
		return val
	}

	return os.Getenv(legacyName)
}

// parseBindAddress parses bind address in format "host:port"
func parseBindAddress(bindAddress string) (string, int) {
	// Simple parsing - find last colon
	for i := len(bindAddress) - 1; i >= 0; i-- {
		if bindAddress[i] == ':' {
			host := bindAddress[:i]

			portStr := bindAddress[i+1:]
			if port, err := strconv.Atoi(portStr); err == nil {
				return host, port
			}

			break
		}
	}

	return "", 0
}
