package services

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"goqlprinter/brotherql"
)

// FoundPrinter holds information about a discovered printer.
type FoundPrinter struct {
	Name         string `json:"name"`
	Model        string `json:"model"`
	UID          string `json:"id"`
	DefaultLabel string `json:"default_label_size,omitempty"`
}

// PrinterService manages printer discovery and connection, with all state encapsulated.
type PrinterService struct {
	mu             sync.Mutex
	provider       brotherql.BackendProvider
	defaultPrinter *FoundPrinter
}

// NewPrinterService creates a new PrinterService with the given BackendProvider.
func NewPrinterService(provider brotherql.BackendProvider) *PrinterService {
	return &PrinterService{provider: provider}
}

// InitializeDefaultPrinter finds and sets the default printer for the session.
func (s *PrinterService) InitializeDefaultPrinter(configuredName string) {
	printers, err := s.FindPrinters()
	if err != nil {
		slog.Error("Error finding printers during initialization", "error", err)
		return
	}
	if len(printers) == 0 {
		slog.Warn("No Brother QL printers found")
		return
	}

	if configuredName != "" {
		if p, ok := matchConfiguredPrinter(printers, configuredName); ok {
			slog.Info("Default printer set from config", "name", p.Name, "model", p.Model, "uid", p.UID)
			s.mu.Lock()
			s.defaultPrinter = &p
			s.mu.Unlock()
			return
		}
		slog.Warn("Configured default printer not found, falling back to first available", "configured", configuredName)
	}

	p := printers[0]
	slog.Info("Default printer set to first available", "name", p.Name, "model", p.Model, "uid", p.UID)
	s.mu.Lock()
	s.defaultPrinter = &p
	s.mu.Unlock()
}

// FindPrinters scans for connected Brother printers.
func (s *PrinterService) FindPrinters() ([]FoundPrinter, error) {
	slog.Debug("Discovering printers using provider", "type", fmt.Sprintf("%T", s.provider))

	printerInfos, err := s.provider.FindPrinters()
	if err != nil {
		slog.Error("Provider failed to find printers", "error", err)
		return nil, fmt.Errorf("failed to discover printers: %w", err)
	}

	slog.Info("Found printers via provider", "count", len(printerInfos))

	foundPrinters := make([]FoundPrinter, 0, len(printerInfos))
	for _, info := range printerInfos {
		name := info.Name
		if name == "" {
			name = info.Model
		}
		foundPrinter := FoundPrinter{
			Name:         name,
			Model:        info.Model,
			UID:          info.URI,
			DefaultLabel: info.DefaultLabel,
		}
		foundPrinters = append(foundPrinters, foundPrinter)
		slog.Debug("Found printer", "name", name, "model", info.Model, "uri", info.URI, "backend", info.Backend)
	}

	return foundPrinters, nil
}

// ResolvePrinter finds a printer by identifier (name, id, or model). Empty → default.
func (s *PrinterService) ResolvePrinter(identifier string) (FoundPrinter, error) {
	if identifier == "" {
		s.mu.Lock()
		def := s.defaultPrinter
		s.mu.Unlock()
		if def != nil {
			return *def, nil
		}
		return FoundPrinter{}, errors.New("no printer specified and no default printer is configured or connected")
	}

	if isPrinterURI(identifier) {
		printers, _ := s.FindPrinters()
		for _, p := range printers {
			if p.UID == identifier {
				return p, nil
			}
		}
		return FoundPrinter{UID: identifier, Model: "Unknown", Name: "Unknown"}, nil
	}

	printers, err := s.FindPrinters()
	if err != nil {
		return FoundPrinter{}, fmt.Errorf("could not list printers to resolve name '%s': %w", identifier, err)
	}

	if p, ok := matchConfiguredPrinter(printers, identifier); ok {
		return p, nil
	}

	return FoundPrinter{}, fmt.Errorf("printer '%s' not found", identifier)
}

// RefreshDefaultPrinter updates the default printer if the same logical printer
// reappeared at a different address (common on Linux where /dev/usb/lp* changes).
func (s *PrinterService) RefreshDefaultPrinter(currentPrinters []FoundPrinter) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.defaultPrinter == nil {
		if len(currentPrinters) > 0 {
			p := currentPrinters[0]
			s.defaultPrinter = &p
			slog.Info("Default printer set to first available", "name", p.Name, "model", p.Model, "uid", p.UID)
		}
		return
	}

	key := defaultPrinterKey(*s.defaultPrinter)
	for _, p := range currentPrinters {
		if defaultPrinterKey(p) == key || p.Name == s.defaultPrinter.Name {
			if p.UID != s.defaultPrinter.UID {
				slog.Info("Default printer URI updated", "name", p.Name, "old", s.defaultPrinter.UID, "new", p.UID)
			}
			s.defaultPrinter = &p
			return
		}
	}

	slog.Warn("Default printer no longer available", "name", s.defaultPrinter.Name, "model", s.defaultPrinter.Model)
	s.defaultPrinter = nil
}

// GetDefaultPrinter returns the current default printer (thread-safe).
func (s *PrinterService) GetDefaultPrinter() *FoundPrinter {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.defaultPrinter
}

func isPrinterURI(identifier string) bool {
	return strings.HasPrefix(identifier, "usb:") ||
		strings.HasPrefix(identifier, "/dev/") ||
		strings.HasPrefix(identifier, "tcp://") ||
		strings.HasPrefix(identifier, "socket://")
}

func matchConfiguredPrinter(printers []FoundPrinter, identifier string) (FoundPrinter, bool) {
	for _, p := range printers {
		if p.Name == identifier || p.UID == identifier || p.Model == identifier {
			return p, true
		}
	}
	return FoundPrinter{}, false
}

func defaultPrinterKey(p FoundPrinter) string {
	if p.UID != "" {
		return p.UID
	}
	return p.Name
}
