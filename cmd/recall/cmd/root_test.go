package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestConfigPrecedence_FlagOverridesEnv(t *testing.T) {
	resetState(t)
	t.Chdir(t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("RECALL_SERVER", "http://from-env")
	if err := rootCmd.PersistentFlags().Set("server", "http://from-flag"); err != nil {
		t.Fatalf("could not set server flag: %v", err)
	}
	initConfig()
	got := viper.GetString("server")
	if got != "http://from-flag" {
		t.Errorf("expected %q; got %q", "http://from-flag", got)
	}
}

func TestConfigPrecedence_EnvOverridesConfig(t *testing.T) {
	resetState(t)
	t.Chdir(t.TempDir())
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeConfig(t, home, "server: http://from-config")
	t.Setenv("RECALL_SERVER", "http://from-env")
	initConfig()
	got := viper.GetString("server")
	if got != "http://from-env" {
		t.Errorf("expected %q; got %q", "http://from-env", got)
	}
}

func TestConfigPrecedence_ConfigOnly(t *testing.T) {
	resetState(t)
	t.Chdir(t.TempDir())
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("RECALL_SERVER", "")
	writeConfig(t, home, "server: http://from-config")
	initConfig()
	got := viper.GetString("server")
	if got != "http://from-config" {
		t.Errorf("expected %q; got %q", "http://from-config", got)
	}
}

// writeConfig creates ~/.config/recall/config.yaml inside the given home with
// the provided YAML body, failing the test on any filesystem error.
func writeConfig(t *testing.T, home, body string) {
	t.Helper()
	configDir := filepath.Join(home, ".config", "recall")
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		t.Fatalf("could not create config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(body), 0o600); err != nil {
		t.Fatalf("could not write config file: %v", err)
	}
}
