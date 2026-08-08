package engine

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/xjasonlyu/tun2socks/v2/core/adapter"
	M "github.com/xjasonlyu/tun2socks/v2/metadata"
)

type nopTransportHandler struct{}

func (nopTransportHandler) HandleTCP(adapter.TCPConn) {}
func (nopTransportHandler) HandleUDP(adapter.UDPConn) {}

type directTestProxy struct {
	address string
	mu      sync.Mutex
	last    *M.Metadata
}

func (p *directTestProxy) DialContext(ctx context.Context, metadata *M.Metadata) (net.Conn, error) {
	p.mu.Lock()
	copy := *metadata
	p.last = &copy
	p.mu.Unlock()
	return (&net.Dialer{}).DialContext(ctx, "tcp", p.address)
}

func (*directTestProxy) DialUDP(*M.Metadata) (net.PacketConn, error) {
	return nil, errors.ErrUnsupported
}

func TestDNSOverHTTPSExchangeUsesProxy(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Errorf("method = %s", request.Method)
		}
		if got := request.Header.Get("Content-Type"); got != "application/dns-message" {
			t.Errorf("content type = %q", got)
		}
		query, err := io.ReadAll(request.Body)
		if err != nil {
			t.Errorf("read query: %v", err)
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		response := append([]byte(nil), query...)
		response[2] |= 0x80
		writer.Header().Set("Content-Type", "application/dns-message")
		_, _ = writer.Write(response)
	}))
	defer server.Close()

	proxy := &directTestProxy{address: server.Listener.Addr().String()}
	tlsConfig := server.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
	handler, err := newDNSOverHTTPSHandler(
		nopTransportHandler{},
		proxy,
		DNSOverHTTPSConfig{URL: server.URL + "/dns-query", Address: "1.1.1.1:443"},
		tlsConfig,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer handler.Close()

	query := []byte{0x12, 0x34, 0x01, 0x00, 0, 0, 0, 0, 0, 0, 0, 0}
	response, err := handler.exchange(query)
	if err != nil {
		t.Fatal(err)
	}
	if response[0] != 0x12 || response[1] != 0x34 || response[2]&0x80 == 0 {
		t.Fatalf("unexpected DNS response: %x", response)
	}
	proxy.mu.Lock()
	metadata := proxy.last
	proxy.mu.Unlock()
	if metadata == nil || metadata.DestinationAddress() != "1.1.1.1:443" || metadata.Network != M.TCP {
		t.Fatalf("proxy metadata = %#v", metadata)
	}
}

func TestDNSFailureResponse(t *testing.T) {
	query := []byte{0xab, 0xcd, 0x01, 0x00, 0, 1, 0, 2, 0, 3, 0, 1}
	response := dnsFailureResponse(query)
	if response[0] != 0xab || response[1] != 0xcd || response[2]&0x80 == 0 || response[3]&0x0f != 2 {
		t.Fatalf("invalid SERVFAIL response: %x", response)
	}
	if response[6] != 0 || response[7] != 0 || response[8] != 0 || response[9] != 0 {
		t.Fatalf("answer counts were not cleared: %x", response[6:10])
	}
}

func TestDNSOverHTTPSConfigRequiresNumericAddress(t *testing.T) {
	proxy := &directTestProxy{}
	_, err := newDNSOverHTTPSHandler(
		nopTransportHandler{},
		proxy,
		DNSOverHTTPSConfig{URL: "https://dns.example/dns-query", Address: "dns.example:443"},
		&tls.Config{},
	)
	if err == nil {
		t.Fatal("hostname bootstrap address unexpectedly accepted")
	}
}

func TestDNSOverHTTPSDisablesIPv6WithoutCallingProxy(t *testing.T) {
	proxy := &directTestProxy{address: "127.0.0.1:1"}
	handler, err := newDNSOverHTTPSHandler(
		nopTransportHandler{},
		proxy,
		DNSOverHTTPSConfig{
			URL:         "https://dns.example/dns-query",
			Address:     "1.1.1.1:443",
			DisableIPv6: true,
		},
		&tls.Config{},
	)
	if err != nil {
		t.Fatal(err)
	}
	defer handler.Close()

	query := []byte{
		0x12, 0x34, 0x01, 0x00, 0x00, 0x01, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01, 0x06, 'g', 'o', 'o',
		'g', 'l', 'e', 0x03, 'c', 'o', 'm', 0x00, 0x00,
		0x1c, 0x00, 0x01,
		// An EDNS OPT record that must not remain after ARCOUNT is cleared.
		0x00, 0x00, 0x29, 0x10, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00,
	}
	response, err := handler.exchange(query)
	if err != nil {
		t.Fatal(err)
	}
	if len(response) != 28 {
		t.Fatalf("response length = %d, want 28", len(response))
	}
	if response[0] != 0x12 || response[1] != 0x34 || response[2]&0x80 == 0 {
		t.Fatalf("invalid DNS response header: %x", response[:12])
	}
	if response[6] != 0 || response[7] != 0 || response[10] != 0 || response[11] != 0 {
		t.Fatalf("DNS NODATA counts were not cleared: %x", response[4:12])
	}
	proxy.mu.Lock()
	called := proxy.last != nil
	proxy.mu.Unlock()
	if called {
		t.Fatal("AAAA query unexpectedly reached the proxy")
	}
}
