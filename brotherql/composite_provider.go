package brotherql

import "fmt"

// CompositeProvider merges a primary discovery backend with configured network printers.
type CompositeProvider struct {
	primary BackendProvider
	network *NetworkProvider
}

// NewCompositeProvider creates a provider that delegates by PrinterInfo.Backend.
// primary may be nil for network-only mode.
func NewCompositeProvider(primary BackendProvider, network *NetworkProvider) *CompositeProvider {
	return &CompositeProvider{primary: primary, network: network}
}

// FindPrinters returns printers from the primary provider plus configured network printers.
func (c *CompositeProvider) FindPrinters() ([]PrinterInfo, error) {
	var out []PrinterInfo
	if c.primary != nil {
		printers, err := c.primary.FindPrinters()
		if err != nil {
			return nil, err
		}
		out = append(out, printers...)
	}
	if c.network != nil && len(c.network.printers) > 0 {
		netPrinters, err := c.network.FindPrinters()
		if err != nil {
			return nil, err
		}
		out = append(out, netPrinters...)
	}
	return out, nil
}

// Connect routes to the network or primary provider based on printer.Backend.
func (c *CompositeProvider) Connect(printer PrinterInfo) (Backend, error) {
	switch printer.Backend {
	case BackendNetwork:
		if c.network == nil {
			return nil, fmt.Errorf("network backend not configured")
		}
		return c.network.Connect(printer)
	default:
		if c.primary == nil {
			return nil, fmt.Errorf("no primary backend configured for %s", printer.Backend)
		}
		return c.primary.Connect(printer)
	}
}

// SupportsStatus returns true if any sub-provider supports status queries.
func (c *CompositeProvider) SupportsStatus() bool {
	if c.primary != nil && c.primary.SupportsStatus() {
		return true
	}
	if c.network != nil && c.network.SupportsStatus() {
		return true
	}
	return false
}
