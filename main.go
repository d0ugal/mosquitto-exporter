package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/d0ugal/promexporter/app"
	"github.com/d0ugal/promexporter/logging"
)

const (
	appName = "Mosquitto Exporter"
)

func main() {
	// Parse command-line flags
	var (
		showVersion   bool
		configPath    string
		configFromEnv bool
		showConfig    bool
	)

	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.BoolVar(&showVersion, "v", false, "Show version information (shorthand)")
	flag.StringVar(&configPath, "config", "config.yaml", "Path to configuration file")
	flag.BoolVar(&configFromEnv, "config-from-env", false, "Load configuration from environment variables only")
	flag.BoolVar(&showConfig, "show-config", false, "Show loaded configuration and exit")
	flag.Parse()

	// Show version if requested
	if showVersion {
		fmt.Printf("%s %s\n", appName, versionString())
		os.Exit(0)
	}

	// Check for CONFIG_FROM_ENV environment variable
	if os.Getenv("CONFIG_FROM_ENV") == "true" {
		configFromEnv = true
	}

	// Load configuration
	cfg, err := LoadConfig(configPath, configFromEnv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Show configuration if requested
	if showConfig {
		displayConfig := cfg.GetDisplayConfig()
		fmt.Printf("Configuration:\n")
		for key, value := range displayConfig {
			fmt.Printf("  %s: %v\n", key, value)
		}
		os.Exit(0)
	}

	// Configure logging
	logging.Configure(&logging.Config{
		Level:  cfg.BaseConfig.Logging.Level,
		Format: cfg.BaseConfig.Logging.Format,
	})

	slog.Info("Starting Mosquitto Exporter",
		"version", versionString(),
		"broker", cfg.Mosquitto.BrokerEndpoint,
		"bind_address", fmt.Sprintf("%s:%d", cfg.BaseConfig.Server.Host, cfg.BaseConfig.Server.Port),
		"config_from_env", configFromEnv,
	)

	// Initialize metrics registry
	metricsRegistry := NewMosquittoMetrics()

	// Build application
	application := app.New(appName).
		WithConfig(&cfg.BaseConfig).
		WithMetrics(metricsRegistry.GetRegistry()).
		WithVersionInfo(versionString(), "unknown", "unknown")

	// Create collector with reference to app for potential tracing
	collector := NewMosquittoCollector(cfg, metricsRegistry, application)
	application.WithCollector(collector)

	// Build and run the application
	if err := application.Build().Run(); err != nil {
		slog.Error("Application failed", "error", err)
		os.Exit(1)
	}
}
