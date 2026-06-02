//go:build !usb

package services

// ConnectToPrinter handles printer connection using the configured BackendProvider.
func ConnectToPrinter(svc *PrinterService, printerIdentifier, modelOverride string, handler PrinterHandler) error {
	printerLock.Lock()
	defer printerLock.Unlock()
	return connectViaProvider(svc, printerIdentifier, modelOverride, handler)
}
