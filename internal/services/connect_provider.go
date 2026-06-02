package services

import (
	"fmt"
	"log/slog"
	"strings"

	"goqlprinter/brotherql"
)

func isNetworkURI(uid string) bool {
	return strings.HasPrefix(uid, "tcp://") || strings.HasPrefix(uid, "socket://")
}

func backendTypeForUID(uid string) brotherql.BackendType {
	if isNetworkURI(uid) {
		return brotherql.BackendNetwork
	}
	if strings.HasPrefix(uid, "usb:") {
		return brotherql.BackendUSB
	}
	return brotherql.BackendNative
}

// connectViaProvider resolves the printer and connects through BackendProvider.Connect.
func connectViaProvider(svc *PrinterService, printerIdentifier, modelOverride string, handler PrinterHandler) error {
	if svc == nil || svc.provider == nil {
		return fmt.Errorf("no backend provider configured")
	}

	resolvedPrinter, modelToUse, err := connectCommon(svc, printerIdentifier, modelOverride)
	if err != nil {
		return err
	}

	printerInfo := brotherql.PrinterInfo{
		Name:         resolvedPrinter.Name,
		Model:        resolvedPrinter.Model,
		URI:          resolvedPrinter.UID,
		Backend:      backendTypeForUID(resolvedPrinter.UID),
		DefaultLabel: resolvedPrinter.DefaultLabel,
	}

	if printerInfo.Name == "" {
		printerInfo.Name = printerInfo.Model
	}

	// USB-formatted URIs on native builds: match discovered printer entry.
	if strings.HasPrefix(resolvedPrinter.UID, "usb:") && printerInfo.Backend != brotherql.BackendNetwork {
		printers, err := svc.provider.FindPrinters()
		if err != nil {
			return fmt.Errorf("failed to find printers: %w", err)
		}
		for _, p := range printers {
			if p.URI == resolvedPrinter.UID || p.Model == resolvedPrinter.Model {
				printerInfo = p
				break
			}
		}
	}

	backend, err := svc.provider.Connect(printerInfo)
	if err != nil {
		return fmt.Errorf("failed to connect to printer: %w", err)
	}
	defer func() {
		if cerr := backend.Close(); cerr != nil {
			slog.Warn("failed to close backend", "error", cerr)
		}
	}()

	return handler(backend, modelToUse)
}
