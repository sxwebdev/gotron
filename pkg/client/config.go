package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"google.golang.org/grpc"
)

// Logger is the minimal interface HealthAwareTransport uses to emit
// informational events (node state transitions, tier shifts, recovered probe
// panics). HealthConfig.Logger defaults to a no-op when left nil; supply your
// own implementation to bridge to slog, log, zap, or any other backend.
type Logger interface {
	Infof(format string, args ...any)
}

// noopLogger discards all messages. Used as the default when
// HealthConfig.Logger is nil.
type noopLogger struct{}

func (noopLogger) Infof(string, ...any) {}

// Protocol represents the transport protocol type
type Protocol string

const (
	// ProtocolGRPC uses gRPC transport (default)
	ProtocolGRPC Protocol = "grpc"
	// ProtocolHTTP uses HTTP REST API transport
	ProtocolHTTP Protocol = "http"
)

// NodeConfig represents configuration for a single node
type NodeConfig struct {
	// Protocol specifies the transport protocol (grpc or http)
	// Default: grpc
	Protocol Protocol

	// Address is the node address
	// For gRPC: "grpc.trongrid.io:50051"
	// For HTTP: "https://api.trongrid.io"
	Address string

	// UseTLS enables TLS for gRPC connections
	UseTLS bool

	// DialOptions are additional gRPC dial options (gRPC only)
	DialOptions []grpc.DialOption

	// HTTPClient allows providing a custom HTTP client (HTTP only)
	HTTPClient *http.Client

	// Headers are custom headers/metadata for requests (HTTP headers and gRPC metadata)
	Headers map[string]string

	// Tier is the priority of this node. 0 = primary, 1 = fallback, 2+ = next.
	// Lower-numbered tiers are preferred: requests are routed to the
	// minimum-numbered tier that has at least one healthy node. A higher tier is
	// only used when every node of every lower tier is unhealthy. Default 0
	// (single primary group, fully backwards compatible).
	Tier int
}

// Config holds the configuration for the Tron client
type Config struct {
	// Nodes is a list of node configurations
	// Round-robin load balancing is used across all nodes
	Nodes []NodeConfig

	// Network specifies the Tron network type (informational only)
	Network Network

	// Blockchain identifies this blockchain for metrics labels.
	// Default: "tron"
	Blockchain string

	// Metrics is an optional metrics collector.
	// If nil, no metrics are collected.
	// Use NewMetrics() for built-in Prometheus metrics,
	// or provide a custom MetricsCollector implementation.
	Metrics MetricsCollector

	// Health configures the per-node health-checker and tier-based fallback.
	// Zero value means "use sane defaults": health-checker is enabled,
	// FailureThreshold/SuccessThreshold = 2, HealthyInterval = 30s,
	// UnhealthyInterval = 5s, InactiveTierInterval = 5m, ProbeTimeout = 5s,
	// Probe = GetNowBlock, ClassifyErr = isNetworkError.
	// To completely disable the health-checker (and use the legacy
	// RoundRobinTransport behaviour 1:1), set Health.Disabled = true.
	Health HealthConfig
}

// HealthConfig configures the health-checker and tier-based fallback behaviour.
// Zero value is valid — defaults are filled in by NewHealthAwareTransport.
type HealthConfig struct {
	// Disabled, if true, switches the client back to the legacy RoundRobinTransport
	// (no background probes, no per-node health state, NodeConfig.Tier ignored).
	Disabled bool

	// FailureThreshold is the number of consecutive network-level failures
	// (probe or live request) required to mark a node unhealthy. Default 2.
	FailureThreshold int

	// SuccessThreshold is the number of consecutive successes (probe or live
	// request) required to bring an unhealthy node back to the pool. Default 2.
	SuccessThreshold int

	// HealthyInterval is the probe interval for healthy nodes whose tier is the
	// active tier (i.e. currently serving traffic). Default 30s.
	HealthyInterval time.Duration

	// UnhealthyInterval is the probe interval for any unhealthy node, regardless
	// of tier. Tighter so recovery is detected quickly. Default 5s.
	UnhealthyInterval time.Duration

	// InactiveTierInterval is the probe interval for healthy nodes that belong
	// to an inactive (lower-priority) tier — i.e. fallbacks not currently
	// serving traffic. Looser to save rate-limit/billing on fallback nodes.
	// Default 5m.
	InactiveTierInterval time.Duration

	// ProbeTimeout bounds a single health-probe invocation. Default 5s.
	ProbeTimeout time.Duration

	// Probe is the function used as a health-check. By default, GetNowBlock(ctx).
	// Custom probes must do a read-only call and return an error when the node
	// is misbehaving.
	Probe func(ctx context.Context, t Transport) error

	// ClassifyErr decides whether a given error counts as a network-level
	// failure. By default, isNetworkError covers gRPC status codes,
	// context.DeadlineExceeded, net.Error, io.EOF, net.ErrClosed and the
	// HTTPStatusError 5xx/408/429 family.
	ClassifyErr func(err error) bool

	// Logger receives informational events on node state transitions and tier
	// shifts via Infof. nil means a no-op logger is used (silent).
	Logger Logger
}

// Validate validates the client configuration
func (c Config) Validate() error {
	if len(c.Nodes) == 0 {
		return fmt.Errorf("%w: at least one node is required", ErrInvalidConfig)
	}

	for i, node := range c.Nodes {
		if err := node.Validate(); err != nil {
			return fmt.Errorf("%w: node %d: %v", ErrInvalidConfig, i, err)
		}
	}

	return nil
}

// Validate validates the node configuration
func (n NodeConfig) Validate() error {
	if n.Address == "" {
		return fmt.Errorf("address is required")
	}

	switch n.GetProtocol() {
	case ProtocolGRPC, ProtocolHTTP:
		// valid protocols
	default:
		return fmt.Errorf("invalid protocol %s", n.Protocol)
	}

	return nil
}

// GetProtocol returns the node protocol, defaulting to gRPC
func (n NodeConfig) GetProtocol() Protocol {
	if n.Protocol == "" {
		return ProtocolGRPC
	}
	return n.Protocol
}
