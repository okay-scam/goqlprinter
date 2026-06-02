package config

import (
	"fmt"
	"strings"

	"goqlprinter/brotherql"
)

// Validate checks application configuration after load.
func (cfg *Config) Validate() error {
	if strings.EqualFold(cfg.App.Backend, "network") && len(cfg.App.Printers) == 0 {
		return fmt.Errorf("app.backend is network but app.printers is empty")
	}
	return brotherql.ValidateNetworkPrinters(cfg.App.NetworkPrinters())
}

// NetworkPrinters converts configured printers to brotherql network config.
func (app *AppConfig) NetworkPrinters() []brotherql.NetworkPrinterConfig {
	out := make([]brotherql.NetworkPrinterConfig, 0, len(app.Printers))
	for _, p := range app.Printers {
		out = append(out, brotherql.NetworkPrinterConfig{
			Name:  p.Name,
			Host:  p.Host,
			Port:  p.Port,
			Model: p.Model,
			Label: p.Label,
		})
	}
	return out
}
