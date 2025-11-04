package client

import (
	"context"
	"testing"
	"time"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "empty address",
			config: &Config{
				GRPCAddress: "",
			},
			wantErr: true,
		},
		{
			name: "valid config",
			config: &Config{
				GRPCAddress: "grpc.trongrid.io:50051",
			},
			wantErr: false,
		},
		{
			name: "valid config with TLS",
			config: &Config{
				GRPCAddress: "grpc.trongrid.io:50051",
				UseTLS:      true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "invalid config",
			config: &Config{
				GRPCAddress: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := New(context.Background(), tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
			if client != nil {
				client.Close()
			}
		})
	}
}

func TestClientDefaultTimeout(t *testing.T) {
	cfg := &Config{
		GRPCAddress: "localhost:50051",
		Timeout:     0,
	}

	// This will fail to connect but we're testing timeout default
	client, _ := New(context.Background(), cfg)
	if client != nil {
		defer client.Close()
		if client.config.Timeout != 30*time.Second {
			t.Errorf("Expected default timeout 30s, got %v", client.config.Timeout)
		}
	}
}
