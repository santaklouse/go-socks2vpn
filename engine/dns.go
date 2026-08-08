package engine

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/xjasonlyu/tun2socks/v2/core/adapter"
	"github.com/xjasonlyu/tun2socks/v2/dialer"
	M "github.com/xjasonlyu/tun2socks/v2/metadata"
	upstreamproxy "github.com/xjasonlyu/tun2socks/v2/proxy"
)

const (
	dnsPort           = 53
	dnsTypeAAAA       = 28
	dnsRequestTimeout = 10 * time.Second
	maxDNSMessageSize = 64 << 10
)

// DNSOverHTTPSConfig redirects DNS packets entering the TUN to an RFC 8484
// endpoint over the configured SOCKS proxy. Address must be a numeric
// host:port so resolving the resolver cannot recursively require DNS.
type DNSOverHTTPSConfig struct {
	URL         string
	Address     string
	DisableIPv6 bool
}

type dnsOverHTTPSHandler struct {
	next        adapter.TransportHandler
	proxy       upstreamproxy.Proxy
	url         string
	address     netip.Addr
	port        uint16
	disableIPv6 bool
	client      *http.Client
	transport   *http.Transport
	ctx         context.Context
	cancel      context.CancelFunc
}

type discardTransportHandler struct{}

func (discardTransportHandler) HandleTCP(conn adapter.TCPConn) { _ = conn.Close() }
func (discardTransportHandler) HandleUDP(conn adapter.UDPConn) { _ = conn.Close() }

// CheckDNSOverHTTPS verifies that DNS wire messages can reach the configured
// RFC 8484 endpoint through the SOCKS proxy.
func CheckDNSOverHTTPS(ctx context.Context, proxyURL string, config DNSOverHTTPSConfig) error {
	proxy, err := parseProxy(proxyURL)
	if err != nil {
		return err
	}
	dialer.Reset()
	handler, err := newDNSOverHTTPSHandler(discardTransportHandler{}, proxy, config, nil)
	if err != nil {
		return err
	}
	defer handler.Close()

	// A query for example.com A. The public resolver response is used only to
	// validate the transport; no answer is consumed by the application.
	query := []byte{
		0x53, 0x32, 0x01, 0x00, 0x00, 0x01, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x07, 'e', 'x', 'a',
		'm', 'p', 'l', 'e', 0x03, 'c', 'o', 'm', 0x00,
		0x00, 0x01, 0x00, 0x01,
	}
	result := make(chan error, 1)
	go func() {
		_, err := handler.exchange(query)
		result <- err
	}()
	select {
	case err := <-result:
		if err != nil {
			return fmt.Errorf("DNS-over-HTTPS check failed: %w", err)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func newDNSOverHTTPSHandler(next adapter.TransportHandler, proxy upstreamproxy.Proxy, config DNSOverHTTPSConfig, tlsConfig *tls.Config) (*dnsOverHTTPSHandler, error) {
	if next == nil {
		return nil, errors.New("DNS-over-HTTPS transport handler is nil")
	}
	if proxy == nil {
		return nil, errors.New("DNS-over-HTTPS proxy is nil")
	}
	endpoint, err := url.Parse(config.URL)
	if err != nil || endpoint.Scheme != "https" || endpoint.Hostname() == "" || endpoint.Path == "" {
		return nil, fmt.Errorf("invalid DNS-over-HTTPS URL %q", config.URL)
	}
	host, portText, err := net.SplitHostPort(config.Address)
	if err != nil {
		return nil, fmt.Errorf("invalid DNS-over-HTTPS address %q: %w", config.Address, err)
	}
	address, err := netip.ParseAddr(host)
	if err != nil {
		return nil, fmt.Errorf("DNS-over-HTTPS address must use a numeric IP: %w", err)
	}
	port, err := strconv.ParseUint(portText, 10, 16)
	if err != nil || port == 0 {
		return nil, fmt.Errorf("invalid DNS-over-HTTPS port %q", portText)
	}

	if tlsConfig == nil {
		tlsConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	} else {
		tlsConfig = tlsConfig.Clone()
	}
	if tlsConfig.ServerName == "" {
		tlsConfig.ServerName = endpoint.Hostname()
	}
	ctx, cancel := context.WithCancel(context.Background())
	handler := &dnsOverHTTPSHandler{
		next:        next,
		proxy:       proxy,
		url:         endpoint.String(),
		address:     address.Unmap(),
		port:        uint16(port),
		disableIPv6: config.DisableIPv6,
		ctx:         ctx,
		cancel:      cancel,
	}
	handler.transport = &http.Transport{
		DialContext:         handler.dialContext,
		ForceAttemptHTTP2:   true,
		TLSClientConfig:     tlsConfig,
		DisableCompression:  true,
		MaxIdleConns:        4,
		MaxIdleConnsPerHost: 4,
		IdleConnTimeout:     30 * time.Second,
	}
	handler.client = &http.Client{
		Transport: handler.transport,
		Timeout:   dnsRequestTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return errors.New("DNS-over-HTTPS redirects are disabled")
		},
	}
	return handler, nil
}

func (h *dnsOverHTTPSHandler) dialContext(ctx context.Context, _, _ string) (net.Conn, error) {
	return h.proxy.DialContext(ctx, &M.Metadata{
		Network: M.TCP,
		DstIP:   h.address,
		DstPort: h.port,
	})
}

func (h *dnsOverHTTPSHandler) HandleTCP(conn adapter.TCPConn) {
	h.next.HandleTCP(conn)
}

func (h *dnsOverHTTPSHandler) HandleUDP(conn adapter.UDPConn) {
	if conn.ID().LocalPort != dnsPort {
		h.next.HandleUDP(conn)
		return
	}
	go h.serveDNS(conn)
}

func (h *dnsOverHTTPSHandler) serveDNS(conn adapter.UDPConn) {
	defer conn.Close()
	buffer := make([]byte, maxDNSMessageSize)
	for {
		n, _, err := conn.ReadFrom(buffer)
		if err != nil {
			return
		}
		query := append([]byte(nil), buffer[:n]...)
		response, err := h.exchange(query)
		if err != nil {
			response = dnsFailureResponse(query)
		}
		if len(response) == 0 {
			continue
		}
		if _, err := conn.WriteTo(response, nil); err != nil {
			return
		}
	}
}

func (h *dnsOverHTTPSHandler) exchange(query []byte) ([]byte, error) {
	if len(query) < 12 || len(query) > maxDNSMessageSize {
		return nil, errors.New("invalid DNS query length")
	}
	if h.disableIPv6 {
		questionTypes, questionEnd, err := parseDNSQuestions(query)
		if err != nil {
			return nil, err
		}
		for _, questionType := range questionTypes {
			if questionType == dnsTypeAAAA {
				return dnsNoDataResponse(query, questionEnd), nil
			}
		}
	}
	ctx, cancel := context.WithTimeout(h.ctx, dnsRequestTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, h.url, bytes.NewReader(query))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/dns-message")
	request.Header.Set("Content-Type", "application/dns-message")
	request.Header["User-Agent"] = nil
	response, err := h.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DNS-over-HTTPS returned %s", response.Status)
	}
	if contentType := response.Header.Get("Content-Type"); contentType != "" && !strings.HasPrefix(strings.ToLower(contentType), "application/dns-message") {
		return nil, fmt.Errorf("unexpected DNS-over-HTTPS content type %q", contentType)
	}
	payload, err := io.ReadAll(io.LimitReader(response.Body, maxDNSMessageSize+1))
	if err != nil {
		return nil, err
	}
	if len(payload) < 12 || len(payload) > maxDNSMessageSize {
		return nil, errors.New("invalid DNS-over-HTTPS response length")
	}
	if payload[0] != query[0] || payload[1] != query[1] || payload[2]&0x80 == 0 {
		return nil, errors.New("DNS-over-HTTPS response does not match the query")
	}
	return payload, nil
}

func parseDNSQuestions(message []byte) ([]uint16, int, error) {
	if len(message) < 12 {
		return nil, 0, errors.New("DNS message is shorter than its header")
	}
	questionCount := int(message[4])<<8 | int(message[5])
	if questionCount == 0 || questionCount > 64 {
		return nil, 0, errors.New("invalid DNS question count")
	}
	offset := 12
	types := make([]uint16, 0, questionCount)
	for range questionCount {
		for {
			if offset >= len(message) {
				return nil, 0, errors.New("truncated DNS question name")
			}
			length := int(message[offset])
			offset++
			switch {
			case length == 0:
				// End of the uncompressed name.
			case length&0xc0 == 0xc0:
				if offset >= len(message) {
					return nil, 0, errors.New("truncated DNS compression pointer")
				}
				offset++
			case length&0xc0 != 0 || length > 63 || offset+length > len(message):
				return nil, 0, errors.New("invalid DNS question name")
			default:
				offset += length
				continue
			}
			break
		}
		if offset+4 > len(message) {
			return nil, 0, errors.New("truncated DNS question")
		}
		types = append(types, uint16(message[offset])<<8|uint16(message[offset+1]))
		offset += 4
	}
	return types, offset, nil
}

func dnsNoDataResponse(query []byte, questionEnd int) []byte {
	response := append([]byte(nil), query[:questionEnd]...)
	response[2] |= 0x80
	response[3] &= 0xf0
	response[6], response[7] = 0, 0
	response[8], response[9] = 0, 0
	response[10], response[11] = 0, 0
	return response
}

func dnsFailureResponse(query []byte) []byte {
	if len(query) < 12 {
		return nil
	}
	response := append([]byte(nil), query...)
	response[2] |= 0x80
	response[3] = response[3]&0xf0 | 0x02
	response[6], response[7] = 0, 0
	response[8], response[9] = 0, 0
	return response
}

func (h *dnsOverHTTPSHandler) Close() {
	h.cancel()
	h.transport.CloseIdleConnections()
}
