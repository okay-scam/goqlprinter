package brotherql

import (
	"io"
	"net"
	"testing"
)

type stubBackend struct{}

func (s *stubBackend) Write(p []byte) (int, error) { return len(p), nil }
func (s *stubBackend) Read(p []byte) (int, error)  { return 0, io.EOF }
func (s *stubBackend) Close() error                { return nil }

type stubPrimary struct {
	printers []PrinterInfo
}

func (s *stubPrimary) FindPrinters() ([]PrinterInfo, error) {
	return s.printers, nil
}

func (s *stubPrimary) Connect(printer PrinterInfo) (Backend, error) {
	return &stubBackend{}, nil
}

func (s *stubPrimary) SupportsStatus() bool {
	return true
}

func TestCompositeProvider_FindPrintersMerge(t *testing.T) {
	t.Parallel()

	primary := &stubPrimary{
		printers: []PrinterInfo{
			{Name: "USB", Model: "QL-700", URI: "usb:0x04f9:0x2048", Backend: BackendUSB},
		},
	}
	network := NewNetworkProvider([]NetworkPrinterConfig{
		{Name: "Net", Host: "10.0.0.1", Model: "QL-800"},
	})
	composite := NewCompositeProvider(primary, network)

	found, err := composite.FindPrinters()
	if err != nil {
		t.Fatalf("FindPrinters: %v", err)
	}
	if len(found) != 2 {
		t.Fatalf("expected 2 printers, got %d", len(found))
	}
	if found[0].Backend != BackendUSB {
		t.Errorf("first backend = %q, want usb", found[0].Backend)
	}
	if found[1].Backend != BackendNetwork {
		t.Errorf("second backend = %q, want network", found[1].Backend)
	}
}

func TestCompositeProvider_ConnectRoutesNetwork(t *testing.T) {
	t.Parallel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer ln.Close()
	go func() {
		c, _ := ln.Accept()
		if c != nil {
			c.Close()
		}
	}()

	network := NewNetworkProvider([]NetworkPrinterConfig{
		{Name: "Net", Host: "127.0.0.1", Port: ln.Addr().(*net.TCPAddr).Port, Model: "QL-800"},
	})
	composite := NewCompositeProvider(&stubPrimary{}, network)

	printers, _ := network.FindPrinters()
	backend, err := composite.Connect(printers[0])
	if err != nil {
		t.Fatalf("Connect network: %v", err)
	}
	backend.Close()
}
