package client

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{"no nodes", Config{}, true},
		{"valid grpc", Config{Nodes: []NodeConfig{{Address: "grpc.example:50051"}}}, false},
		{"valid http", Config{Nodes: []NodeConfig{{Protocol: ProtocolHTTP, Address: "https://example"}}}, false},
		{"node missing address", Config{Nodes: []NodeConfig{{Address: ""}}}, true},
		{"invalid protocol", Config{Nodes: []NodeConfig{{Protocol: "ftp", Address: "x"}}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
				require.ErrorIs(t, err, ErrInvalidConfig)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestNodeConfigGetProtocol(t *testing.T) {
	require.Equal(t, ProtocolGRPC, NodeConfig{}.GetProtocol(), "empty protocol defaults to grpc")
	require.Equal(t, ProtocolHTTP, NodeConfig{Protocol: ProtocolHTTP}.GetProtocol())
	require.Equal(t, ProtocolGRPC, NodeConfig{Protocol: ProtocolGRPC}.GetProtocol())
}

func TestConfigValidateWrapsNodeError(t *testing.T) {
	err := Config{Nodes: []NodeConfig{{Address: "ok:1"}, {Address: ""}}}.Validate()
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidConfig)
	require.ErrorContains(t, err, "node 1")
}
