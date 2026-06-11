package config_test

import (
	"strings"
	"testing"

	"github.com/PedroEvaldt/recall/internal/config"
)

func TestLoadFrom(t *testing.T) {
	tests := []struct {
		name          string
		env           map[string]string
		wantCfg       *config.Config
		wantErrSubstr string
	}{
		{
			name: "All correct",
			env: map[string]string{
				"DB_URL":       "testurl",
				"AUTH_TOKEN":   "testtoken",
				"SERVER_HOST":  "testhost",
				"SERVER_PORT":  "8000",
				"STORAGE_PATH": "./teststorage",
				"LOG_LEVEL":    "info",
				"LOG_FORMAT":   "text",
			},
			wantCfg: &config.Config{
				DBURL:       "testurl",
				AuthToken:   "testtoken",
				Host:        "testhost",
				Port:        "8000",
				StoragePath: "./teststorage",
				LogLevel:    "info",
				LogFormat:   "text",
			},
		},
		{
			name: "No AUTH_TOKEN",
			env: map[string]string{
				"DB_URL":       "testurl",
				"SERVER_HOST":  "testhost",
				"SERVER_PORT":  "8000",
				"STORAGE_PATH": "./teststorage",
				"LOG_LEVEL":    "info",
				"LOG_FORMAT":   "text",
			},
			wantErrSubstr: "AUTH_TOKEN is required",
		},
		{
			name: "No DB_URL",
			env: map[string]string{
				"AUTH_TOKEN":   "testtoken",
				"SERVER_HOST":  "testhost",
				"SERVER_PORT":  "8000",
				"STORAGE_PATH": "./teststorage",
				"LOG_LEVEL":    "info",
				"LOG_FORMAT":   "text",
			},
			wantErrSubstr: "DB_URL is required",
		},
		{
			name: "No SERVER_HOST",
			env: map[string]string{
				"DB_URL":       "testurl",
				"AUTH_TOKEN":   "testtoken",
				"SERVER_PORT":  "8000",
				"STORAGE_PATH": "./teststorage",
				"LOG_LEVEL":    "info",
				"LOG_FORMAT":   "text",
			},
			wantCfg: &config.Config{
				DBURL:       "testurl",
				AuthToken:   "testtoken",
				Host:        "localhost",
				Port:        "8000",
				StoragePath: "./teststorage",
				LogLevel:    "info",
				LogFormat:   "text",
			},
		},
		{
			name: "No SERVER_PORT",
			env: map[string]string{
				"DB_URL":       "testurl",
				"AUTH_TOKEN":   "testtoken",
				"SERVER_HOST":  "testhost",
				"STORAGE_PATH": "./teststorage",
				"LOG_LEVEL":    "info",
				"LOG_FORMAT":   "text",
			},
			wantCfg: &config.Config{
				DBURL:       "testurl",
				AuthToken:   "testtoken",
				Host:        "testhost",
				Port:        "8080",
				StoragePath: "./teststorage",
				LogLevel:    "info",
				LogFormat:   "text",
			},
		},
		{
			name: "No STORAGE_PATH",
			env: map[string]string{
				"DB_URL":      "testurl",
				"AUTH_TOKEN":  "testtoken",
				"SERVER_HOST": "testhost",
				"SERVER_PORT": "8000",
				"LOG_LEVEL":   "info",
				"LOG_FORMAT":  "text",
			},
			wantCfg: &config.Config{
				DBURL:       "testurl",
				AuthToken:   "testtoken",
				Host:        "testhost",
				Port:        "8000",
				StoragePath: "./storage",
				LogLevel:    "info",
				LogFormat:   "text",
			},
		},
		{
			name: "No LOG_LEVEL",
			env: map[string]string{
				"DB_URL":       "testurl",
				"AUTH_TOKEN":   "testtoken",
				"SERVER_PORT":  "8000",
				"STORAGE_PATH": "./teststorage",
				"LOG_FORMAT":   "text",
			},
			wantCfg: &config.Config{
				DBURL:       "testurl",
				AuthToken:   "testtoken",
				Host:        "localhost",
				Port:        "8000",
				StoragePath: "./teststorage",
				LogLevel:    "info",
				LogFormat:   "text",
			},
		},
		{
			name: "No LOG_FORMAT",
			env: map[string]string{
				"DB_URL":       "testurl",
				"AUTH_TOKEN":   "testtoken",
				"SERVER_PORT":  "8000",
				"STORAGE_PATH": "./teststorage",
				"LOG_LEVEL":    "info",
			},
			wantCfg: &config.Config{
				DBURL:       "testurl",
				AuthToken:   "testtoken",
				Host:        "localhost",
				Port:        "8000",
				StoragePath: "./teststorage",
				LogLevel:    "info",
				LogFormat:   "text",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(t.TempDir())

			getenv := func(k string) string { return tt.env[k] }
			cfg, err := config.LoadFrom(getenv)
			if tt.wantErrSubstr != "" {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Errorf("err = %v; want substring %q", err, tt.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadFrom: %v", err)
			}
			if *cfg != *tt.wantCfg {
				t.Errorf("cfg = %+v, want %+v", *cfg, *tt.wantCfg)
			}
		})
	}
}
