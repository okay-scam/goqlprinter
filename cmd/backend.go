package cmd

import (
	"log/slog"

	"goqlprinter/brotherql"
	icfg "goqlprinter/internal/config"
)

// InitBackendProvider selects and initializes the appropriate backend provider.
func InitBackendProvider(cfg *icfg.Config) brotherql.BackendProvider {
	network := brotherql.NewNetworkProvider(cfg.App.NetworkPrinters())

	var primary brotherql.BackendProvider
	switch cfg.App.Backend {
	case "network":
		slog.Info("Using network backend (TCP)")
		return wrapComposite(nil, network, cfg)
	case "usb":
		slog.Info("Using USB backend (gousb/libusb)")
		primary = initUSBProvider()
	case "native":
		slog.Info("Using native OS backend")
		primary = brotherql.NewNativeProvider()
	case "auto":
		primary = autoDetectProvider()
	default:
		slog.Warn("Unknown backend, falling back to auto mode", "backend", cfg.App.Backend)
		primary = autoDetectProvider()
	}
	return wrapComposite(primary, network, cfg)
}

func wrapComposite(primary brotherql.BackendProvider, network *brotherql.NetworkProvider, cfg *icfg.Config) brotherql.BackendProvider {
	if len(cfg.App.Printers) == 0 {
		if primary != nil {
			return primary
		}
		return network
	}
	slog.Info("Merging configured network printers", "count", len(cfg.App.Printers))
	return brotherql.NewCompositeProvider(primary, network)
}

func autoDetectProvider() brotherql.BackendProvider {
	slog.Info("Auto mode: trying USB backend first")
	usbProvider := initUSBProvider()
	if usbProvider != nil {
		printers, err := usbProvider.FindPrinters()
		if err == nil && len(printers) > 0 {
			slog.Info("USB backend found printers", "count", len(printers))
			return usbProvider
		}
		slog.Info("USB backend found no printers, trying native backend")
	}

	nativeProvider := brotherql.NewNativeProvider()
	printers, err := nativeProvider.FindPrinters()
	if err == nil && len(printers) > 0 {
		slog.Info("Native backend found printers", "count", len(printers))
		return nativeProvider
	}

	slog.Info("No printers found with any backend")
	return nativeProvider
}
