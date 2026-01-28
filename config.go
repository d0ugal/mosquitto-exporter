package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/d0ugal/promexporter/config"
)

// MosquittoExporterConfig extends the base configuration with Mosquitto-specific settings
type MosquittoExporterConfig struct {
	config.BaseConfig
	Mosquitto MosquittoConfig `yaml:"mosquitto"`
}

// MosquittoConfig holds Mosquitto broker connection settings
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
	cfg["Mosquitto Broker"] = c.Mosquitto.BrokerEndpoint
	cfg["MQTT Username"] = c.Mosquitto.Username
	cfg["MQTT Client ID"] = c.Mosquitto.ClientID
	cfg["TLS Enabled"] = c.Mosquitto.TLS.Enabled
	if c.Mosquitto.TLS.Enabled {
		cfg["TLS Certificate"] = c.Mosquitto.TLS.CertFile
		cfg["TLS Key File"] = c.Mosquitto.TLS.KeyFile
		cfg["TLS Skip Verify"] = c.Mosquitto.TLS.InsecureSkipVerify
	}
	return cfg
}

// LoadConfig loads configuration from YAML file or environment variables
func LoadConfig(configPath string, configFromEnv bool) (*MosquittoExporterConfig, error) {
	var cfg MosquittoExporterConfig

	if configFromEnv {
		// Load from environment variables
		if err := loadFromEnv(&cfg); err != nil {
			return nil, fmt.Errorf("failed to load config from environment: %w", err)
		}
	} else {
		// Load from YAML file
		baseConfig, err := config.Load(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load config from file: %w", err)
		}
		cfg.BaseConfig = *baseConfig

		// Apply Mosquitto-specific environment variable overrides
		if err := applyMosquittoEnvVars(&cfg); err != nil {
			return nil, fmt.Errorf("failed to apply environment overrides: %w", err)
		}
	}

	// Set defaults if not configured
	setDefaults(&cfg)

	return &cfg, nil
}

// loadFromEnv loads configuration entirely from environment variables
func loadFromEnv(cfg *MosquittoExporterConfig) error {
	// Create a base config with defaults
	cfg.BaseConfig = config.BaseConfig{}

	// Apply generic environment variables
	if err := config.ApplyGenericEnvVars(&cfg.BaseConfig); err != nil {
		return fmt.Errorf("failed to apply generic environment variables: %w", err)
	}

	// Apply Mosquitto-specific environment variables
	return applyMosquittoEnvVars(cfg)
}

// applyMosquittoEnvVars applies Mosquitto-specific environment variables
func applyMosquittoEnvVars(cfg *MosquittoExporterConfig) error {
	// Broker endpoint - support both new and legacy env var names
	if endpoint := getEnv("MOSQUITTO_BROKER_ENDPOINT", "BROKER_ENDPOINT"); endpoint != "" {
		cfg.Mosquitto.BrokerEndpoint = endpoint
	}

	// Username - support both new and legacy env var names
	if username := getEnv("MOSQUITTO_USERNAME", "MQTT_USER"); username != "" {
		cfg.Mosquitto.Username = username
	}

	// Password - support both new and legacy env var names
	if password := getEnv("MOSQUITTO_PASSWORD", "MQTT_PASS"); password != "" {
		cfg.Mosquitto.Password = config.NewSensitiveString(password)
	}

	// Client ID - support both new and legacy env var names
	if clientID := getEnv("MOSQUITTO_CLIENT_ID", "MQTT_CLIENT_ID"); clientID != "" {
		cfg.Mosquitto.ClientID = clientID
	}

	// TLS settings - support both new and legacy env var names
	if certFile := getEnv("MOSQUITTO_TLS_CERT_FILE", "MQTT_CERT"); certFile != "" {
		cfg.Mosquitto.TLS.CertFile = certFile
		cfg.Mosquitto.TLS.Enabled = true
	}

	if keyFile := getEnv("MOSQUITTO_TLS_KEY_FILE", "MQTT_KEY"); keyFile != "" {
		cfg.Mosquitto.TLS.KeyFile = keyFile
		cfg.Mosquitto.TLS.Enabled = true
	}

	if skipVerify := os.Getenv("MOSQUITTO_TLS_INSECURE_SKIP_VERIFY"); skipVerify != "" {
		if val, err := strconv.ParseBool(skipVerify); err == nil {
			cfg.Mosquitto.TLS.InsecureSkipVerify = val
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
	// Mosquitto defaults
	if cfg.Mosquitto.BrokerEndpoint == "" {
		cfg.Mosquitto.BrokerEndpoint = "tcp://127.0.0.1:1883"
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
