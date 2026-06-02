package brotherql

import (
	"errors"
	"io"
	"net"
	"testing"
)

func TestBuildNetworkURI(t *testing.T) {
	t.Parallel()

	cases := []struct {
		host    string
		port    int
		want    string
		wantErr bool
	}{
		{"192.168.1.10", 9100, "tcp://192.168.1.10:9100", false},
		{"printer.local", 0, "tcp://printer.local:9100", false},
		{"", 9100, "", true},
	}
	for _, tc := range cases {
		got, err := BuildNetworkURI(tc.host, tc.port)
		if tc.wantErr {
			if err == nil {
				t.Errorf("BuildNetworkURI(%q, %d) expected error", tc.host, tc.port)
			}
			continue
		}
		if err != nil {
			t.Fatalf("BuildNetworkURI(%q, %d): %v", tc.host, tc.port, err)
		}
		if got != tc.want {
			t.Errorf("BuildNetworkURI(%q, %d) = %q, want %q", tc.host, tc.port, got, tc.want)
		}
	}
}

func TestParseNetworkURI(t *testing.T) {
	t.Parallel()

	cases := []struct {
		uri     string
		want    string
		wantErr bool
	}{
		{"tcp://192.168.1.10:9100", "192.168.1.10:9100", false},
		{"socket://192.168.1.10:9100", "192.168.1.10:9100", false},
		{"tcp://192.168.1.10", "192.168.1.10:9100", false},
		{"192.168.1.10", "192.168.1.10:9100", false},
		{"", "", true},
		{"ftp://192.168.1.10", "", true},
	}
	for _, tc := range cases {
		got, err := ParseNetworkURI(tc.uri)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParseNetworkURI(%q) expected error", tc.uri)
			}
			continue
		}
		if err != nil {
			t.Fatalf("ParseNetworkURI(%q): %v", tc.uri, err)
		}
		if got != tc.want {
			t.Errorf("ParseNetworkURI(%q) = %q, want %q", tc.uri, got, tc.want)
		}
	}
}

func TestNetworkBackend_ReadReturnsNotSupported(t *testing.T) {
	t.Parallel()

	b := &NetworkBackend{}
	_, err := b.Read(make([]byte, 32))
	if !errors.Is(err, ErrStatusNotSupported) {
		t.Errorf("Read err = %v, want ErrStatusNotSupported", err)
	}
}

func TestNetworkProvider_ConnectAndWrite(t *testing.T) {
	t.Parallel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = io.ReadAll(conn)
	}()

	provider := NewNetworkProvider([]NetworkPrinterConfig{
		{Name: "test", Host: "127.0.0.1", Port: ln.Addr().(*net.TCPAddr).Port, Model: "QL-800"},
	})
	printers, err := provider.FindPrinters()
	if err != nil {
		t.Fatalf("FindPrinters: %v", err)
	}
	if len(printers) != 1 {
		t.Fatalf("expected 1 printer, got %d", len(printers))
	}

	backend, err := provider.Connect(printers[0])
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer backend.Close()

	payload := []byte{0x1b, 0x40, 0x00, 0x00}
	if _, err := backend.Write(payload); err != nil {
		t.Fatalf("Write: %v", err)
	}
}

func TestValidateNetworkPrinters_DuplicateName(t *testing.T) {
	t.Parallel()

	err := ValidateNetworkPrinters([]NetworkPrinterConfig{
		{Name: "a", Host: "1.2.3.4", Model: "QL-800"},
		{Name: "a", Host: "1.2.3.5", Model: "QL-800"},
	})
	if err == nil {
		t.Fatal("expected duplicate name error")
	}
}
