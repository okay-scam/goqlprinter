package brotherql

import (
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultNetworkPort = 9100

// NetworkPrinterConfig describes a configured network printer entry.
type NetworkPrinterConfig struct {
	Name  string
	Host  string
	Port  int
	Model string
	Label string
}

// NetworkBackend sends raster data to a printer over TCP.
type NetworkBackend struct {
	conn net.Conn
}

// Write sends all data to the printer connection.
func (b *NetworkBackend) Write(data []byte) (int, error) {
	if b.conn == nil {
		return 0, fmt.Errorf("network connection not open")
	}
	total := 0
	for len(data) > 0 {
		n, err := b.conn.Write(data)
		total += n
		if err != nil {
			return total, err
		}
		data = data[n:]
	}
	return total, nil
}

// Read is not supported on raw TCP network backends.
func (b *NetworkBackend) Read(data []byte) (int, error) {
	return 0, ErrStatusNotSupported
}

// Close closes the TCP connection.
func (b *NetworkBackend) Close() error {
	if b.conn == nil {
		return nil
	}
	err := b.conn.Close()
	b.conn = nil
	return err
}

// NetworkProvider implements BackendProvider for config-driven TCP printers.
type NetworkProvider struct {
	printers []NetworkPrinterConfig
}

// NewNetworkProvider creates a provider from validated network printer config.
func NewNetworkProvider(printers []NetworkPrinterConfig) *NetworkProvider {
	return &NetworkProvider{printers: printers}
}

// FindPrinters returns all configured network printers.
func (p *NetworkProvider) FindPrinters() ([]PrinterInfo, error) {
	out := make([]PrinterInfo, 0, len(p.printers))
	for _, cfg := range p.printers {
		uri, err := BuildNetworkURI(cfg.Host, cfg.Port)
		if err != nil {
			return nil, err
		}
		name := cfg.Name
		if name == "" {
			name = cfg.Model
		}
		out = append(out, PrinterInfo{
			Name:         name,
			Model:        cfg.Model,
			URI:          uri,
			Backend:      BackendNetwork,
			DefaultLabel: cfg.Label,
		})
	}
	return out, nil
}

// Connect opens a TCP connection to the printer URI.
func (p *NetworkProvider) Connect(printer PrinterInfo) (Backend, error) {
	if printer.Backend != BackendNetwork {
		return nil, fmt.Errorf("printer is not a network printer")
	}
	addr, err := ParseNetworkURI(printer.URI)
	if err != nil {
		return nil, err
	}
	slog.Info("Network: connecting", "uri", printer.URI, "addr", addr)
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
	}
	return &NetworkBackend{conn: conn}, nil
}

// SupportsStatus returns false; raw TCP does not support ESC i S readback.
func (p *NetworkProvider) SupportsStatus() bool {
	return false
}

// BuildNetworkURI returns the canonical tcp:// URI for a host and port.
func BuildNetworkURI(host string, port int) (string, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return "", fmt.Errorf("network printer host is required")
	}
	if port <= 0 {
		port = defaultNetworkPort
	}
	return fmt.Sprintf("tcp://%s", net.JoinHostPort(host, strconv.Itoa(port))), nil
}

// ParseNetworkURI parses tcp:// or socket:// URIs into host:port for net.Dial.
func ParseNetworkURI(uri string) (string, error) {
	uri = strings.TrimSpace(uri)
	if uri == "" {
		return "", fmt.Errorf("empty printer URI")
	}
	if strings.HasPrefix(uri, "socket://") {
		uri = "tcp://" + strings.TrimPrefix(uri, "socket://")
	}
	if !strings.Contains(uri, "://") {
		// host or host:port without scheme
		if _, _, err := net.SplitHostPort(uri); err != nil {
			uri = net.JoinHostPort(uri, strconv.Itoa(defaultNetworkPort))
		}
		return uri, nil
	}
	u, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("invalid printer URI %q: %w", uri, err)
	}
	switch strings.ToLower(u.Scheme) {
	case "tcp", "socket":
	default:
		return "", fmt.Errorf("unsupported network URI scheme %q", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return "", fmt.Errorf("network printer URI missing host: %q", uri)
	}
	port := u.Port()
	if port == "" {
		port = strconv.Itoa(defaultNetworkPort)
	}
	return net.JoinHostPort(host, port), nil
}

// ValidateNetworkPrinterConfig checks a single configured printer entry.
func ValidateNetworkPrinterConfig(cfg NetworkPrinterConfig) error {
	if strings.TrimSpace(cfg.Name) == "" {
		return fmt.Errorf("network printer name is required")
	}
	if strings.TrimSpace(cfg.Host) == "" {
		return fmt.Errorf("network printer %q: host is required", cfg.Name)
	}
	if _, err := GetModel(cfg.Model); err != nil {
		return fmt.Errorf("network printer %q: %w", cfg.Name, err)
	}
	if cfg.Label != "" {
		if _, err := GetLabel(cfg.Label); err != nil {
			return fmt.Errorf("network printer %q: invalid label %q: %w", cfg.Name, cfg.Label, err)
		}
	}
	if cfg.Port < 0 {
		return fmt.Errorf("network printer %q: invalid port %d", cfg.Name, cfg.Port)
	}
	return nil
}

// ValidateNetworkPrinters validates a list and ensures unique names.
func ValidateNetworkPrinters(printers []NetworkPrinterConfig) error {
	seen := make(map[string]struct{}, len(printers))
	for _, cfg := range printers {
		if err := ValidateNetworkPrinterConfig(cfg); err != nil {
			return err
		}
		if _, dup := seen[cfg.Name]; dup {
			return fmt.Errorf("duplicate network printer name %q", cfg.Name)
		}
		seen[cfg.Name] = struct{}{}
	}
	return nil
}
