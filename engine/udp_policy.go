package engine

import (
	"sync"

	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
	"gvisor.dev/gvisor/pkg/waiter"

	"github.com/xjasonlyu/tun2socks/v2/core/adapter"
)

// delayedLinkEndpoint keeps the TUN reader stopped while the UDP handler is
// replaced. SetTransportProtocolHandler is only supported during stack
// initialization, so no packet may enter the stack before Activate.
type delayedLinkEndpoint struct {
	stack.LinkEndpoint
	mu         sync.Mutex
	dispatcher stack.NetworkDispatcher
	activated  bool
}

func newDelayedLinkEndpoint(endpoint stack.LinkEndpoint) *delayedLinkEndpoint {
	return &delayedLinkEndpoint{LinkEndpoint: endpoint}
}

func (endpoint *delayedLinkEndpoint) Attach(dispatcher stack.NetworkDispatcher) {
	endpoint.mu.Lock()
	if !endpoint.activated {
		endpoint.dispatcher = dispatcher
		endpoint.mu.Unlock()
		return
	}
	endpoint.mu.Unlock()
	endpoint.LinkEndpoint.Attach(dispatcher)
}

func (endpoint *delayedLinkEndpoint) IsAttached() bool {
	endpoint.mu.Lock()
	defer endpoint.mu.Unlock()
	if !endpoint.activated {
		return endpoint.dispatcher != nil
	}
	return endpoint.LinkEndpoint.IsAttached()
}

func (endpoint *delayedLinkEndpoint) Activate() {
	endpoint.mu.Lock()
	if endpoint.activated {
		endpoint.mu.Unlock()
		return
	}
	endpoint.activated = true
	dispatcher := endpoint.dispatcher
	endpoint.mu.Unlock()
	endpoint.LinkEndpoint.Attach(dispatcher)
}

// installUDPRejectPolicy replaces the default UDP forwarder with an equivalent
// one that leaves selected destination ports unhandled. gVisor then returns an
// ICMP port-unreachable response, allowing protocols such as browser QUIC to
// fail fast and fall back to TCP instead of timing out inside a broken SOCKS5
// UDP relay.
func installUDPRejectPolicy(networkStack *stack.Stack, handler adapter.TransportHandler, ports []uint16) {
	rejected := make(map[uint16]struct{}, len(ports))
	for _, port := range ports {
		if port != 0 {
			rejected[port] = struct{}{}
		}
	}
	if len(rejected) == 0 {
		return
	}

	forwarder := udp.NewForwarder(networkStack, func(request *udp.ForwarderRequest) bool {
		id := request.ID()
		if rejectUDPPort(id.LocalPort, rejected) {
			return false
		}

		var queue waiter.Queue
		endpoint, err := request.CreateEndpoint(&queue)
		if err != nil {
			return false
		}
		handler.HandleUDP(&policyUDPConn{
			UDPConn: gonet.NewUDPConn(&queue, endpoint),
			id:      id,
		})
		return true
	})
	networkStack.SetTransportProtocolHandler(udp.ProtocolNumber, forwarder.HandlePacket)
}

func rejectUDPPort(port uint16, rejected map[uint16]struct{}) bool {
	_, found := rejected[port]
	return found
}

type policyUDPConn struct {
	*gonet.UDPConn
	id stack.TransportEndpointID
}

func (connection *policyUDPConn) ID() stack.TransportEndpointID {
	return connection.id
}
