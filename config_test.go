package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	return path
}

func clearPasswordEnv(t *testing.T) {
	t.Helper()
	t.Setenv("MOSQUITTO_PASSWORD", "")
	t.Setenv("MQTT_PASS", "")
}

func TestLoadConfigYAMLPassword(t *testing.T) {
	clearPasswordEnv(t)

	path := writeConfig(t, "mosquitto:\n  username: prometheus\n  password: \"plaintext\"\n")

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if got := cfg.Mosquitto.Password.Value(); got != "plaintext" {
		t.Errorf("password = %q, want %q", got, "plaintext")
	}
}

func TestLoadConfigYAMLEmptyPassword(t *testing.T) {
	clearPasswordEnv(t)

	path := writeConfig(t, "mosquitto:\n  password: \"\"\n")

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if got := cfg.Mosquitto.Password.Value(); got != "" {
		t.Errorf("password = %q, want empty", got)
	}
}

func TestLoadConfigEnvOverridesYAMLPassword(t *testing.T) {
	clearPasswordEnv(t)
	t.Setenv("MOSQUITTO_PASSWORD", "fromenv")

	path := writeConfig(t, "mosquitto:\n  password: \"fromyaml\"\n")

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if got := cfg.Mosquitto.Password.Value(); got != "fromenv" {
		t.Errorf("password = %q, want %q", got, "fromenv")
	}
}
