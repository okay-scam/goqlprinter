package config_test

import (
	"testing"

	"goqlprinter/internal/config"
)

func TestLoadConfig_ReturnsNonNil(t *testing.T) {
	t.Parallel()

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil *Config")
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	t.Parallel()

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Server.Port <= 0 {
		t.Errorf("expected a positive default port, got %d", cfg.Server.Port)
	}
	if cfg.App.Backend == "" {
		t.Error("expected a non-empty default backend")
	}
}

func TestLoadConfig_EnvVarOverride(t *testing.T) {
	// Cannot be parallel: modifies environment variables.
	t.Setenv("LABELPRINTER_SERVER_PORT", "9999")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Server.Port != 9999 {
		t.Errorf("expected Port=9999 from env var, got %d", cfg.Server.Port)
	}
}

func TestLoadConfig_FontDirsDefault(t *testing.T) {
	t.Parallel()

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.App.FontDirs == nil {
		t.Fatal("expected non-nil FontDirs")
	}
	if len(cfg.App.FontDirs) == 0 {
		t.Error("expected at least one default font directory")
	}
}

func TestValidate_NetworkBackendRequiresPrinters(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		App: config.AppConfig{
			Backend: "network",
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for network backend without printers")
	}
}

func TestValidate_NetworkPrinterEntry(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		App: config.AppConfig{
			Printers: []config.ConfiguredPrinter{
				{Name: "p1", Host: "192.168.0.1", Model: "QL-800", Label: "62"},
			},
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	net := cfg.App.NetworkPrinters()
	if len(net) != 1 || net[0].Name != "p1" {
		t.Fatalf("NetworkPrinters() = %+v", net)
	}
}

func TestLoadConfig_IsIndependent(t *testing.T) {
	t.Parallel()

	cfg1, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error on first LoadConfig: %v", err)
	}

	cfg2, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error on second LoadConfig: %v", err)
	}

	// Mutate cfg1 and verify cfg2 is unaffected.
	original := cfg2.Server.Port
	cfg1.Server.Port = 0

	if cfg2.Server.Port != original {
		t.Errorf("modifying cfg1 affected cfg2: cfg2.Server.Port changed from %d to %d", original, cfg2.Server.Port)
	}
}
