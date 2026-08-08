package engine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"

	M "github.com/xjasonlyu/tun2socks/v2/metadata"
	upstreamproxy "github.com/xjasonlyu/tun2socks/v2/proxy"
)

type diagnosticProxy struct {
	next upstreamproxy.Proxy
	warn func(string)
}

func newDiagnosticProxy(next upstreamproxy.Proxy, warn func(string)) upstreamproxy.Proxy {
	return &diagnosticProxy{next: next, warn: warn}
}

func (proxy *diagnosticProxy) DialContext(ctx context.Context, metadata *M.Metadata) (net.Conn, error) {
	connection, err := proxy.next.DialContext(ctx, metadata)
	if err != nil {
		proxy.warn(fmt.Sprintf("TCP dial %s: %v", metadata.DestinationAddress(), err))
		return nil, err
	}
	return &diagnosticConn{
		Conn:        connection,
		destination: metadata.DestinationAddress(),
		warn:        proxy.warn,
	}, nil
}

func (proxy *diagnosticProxy) DialUDP(metadata *M.Metadata) (net.PacketConn, error) {
	connection, err := proxy.next.DialUDP(metadata)
	if err != nil {
		proxy.warn(fmt.Sprintf("UDP dial %s: %v", metadata.DestinationAddress(), err))
	}
	return connection, err
}

type diagnosticConn struct {
	net.Conn
	destination string
	warn        func(string)
}

func (connection *diagnosticConn) Read(buffer []byte) (int, error) {
	count, err := connection.Conn.Read(buffer)
	connection.report("read", err)
	return count, err
}

func (connection *diagnosticConn) Write(buffer []byte) (int, error) {
	count, err := connection.Conn.Write(buffer)
	connection.report("write", err)
	return count, err
}

func (connection *diagnosticConn) report(operation string, err error) {
	if err == nil || errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
		return
	}
	connection.warn(fmt.Sprintf("TCP %s %s: %v", operation, connection.destination, err))
}
