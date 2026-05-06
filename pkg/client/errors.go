package client

import (
	"errors"
	"fmt"
)

// TransportError represents an error from a transport-level RPC call.
// Use errors.AsType[*TransportError] (Go 1.26+) to extract it and access
// Host, Protocol, Method fields.
type TransportError struct {
	// Host is the address of the server (e.g. "grpc.trongrid.io:50051" or "https://api.trongrid.io")
	Host string
	// Protocol is the transport protocol ("grpc" or "http")
	Protocol string
	// Method is the RPC method or HTTP endpoint (e.g. "/protocol.Wallet/GetAccount" or "/wallet/getaccount")
	Method string
	// Err is the original error
	Err error
}

func (e *TransportError) Error() string {
	return fmt.Sprintf("%s %s (%s): %s", e.Protocol, e.Method, e.Host, e.Err)
}

func (e *TransportError) Unwrap() error {
	return e.Err
}

// HTTPStatusError is returned by HTTPTransport when the remote responds with
// a non-2xx status. It is wrapped in a TransportError before reaching the
// caller, so use errors.AsType[*HTTPStatusError](err) to inspect the
// status code.
//
// The health-checker's default classifier treats 5xx, 408 and 429 as
// network-level failures (count toward the unhealthy threshold) and other
// 4xx codes as logical errors (do not affect node health).
type HTTPStatusError struct {
	// Code is the HTTP status code (e.g. 503).
	Code int
	// Body is the raw response body — kept for diagnostics.
	Body string
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("http status %d: %s", e.Code, e.Body)
}

var (
	// Common errors
	ErrInvalidConfig = errors.New("invalid client configuration")
	ErrNotConnected  = errors.New("client not connected")
	ErrInvalidParams = errors.New("invalid parameters")
	ErrNilResponse   = errors.New("nil response from server")

	// Address errors
	ErrInvalidAddress      = errors.New("invalid address")
	ErrEmptyAddress        = errors.New("address is empty")
	ErrAccountNotActivated = errors.New("account is not activated")

	// Transaction errors
	ErrInvalidAmount           = errors.New("invalid amount")
	ErrInvalidTransaction      = errors.New("invalid transaction")
	ErrInvalidPrivateKey       = errors.New("invalid private key")
	ErrTransactionNotFound     = errors.New("transaction not found")
	ErrTransactionInfoNotFound = errors.New("transaction info not found")

	// Resources errors
	ErrInvalidResourceType = errors.New("invalid resource type")

	// ErrNoHealthyNodes is returned when no node in any tier is currently
	// marked healthy. The health-checker runs continuously and will return
	// nodes to the pool as soon as they recover; callers should retry with
	// backoff. Use errors.Is(err, client.ErrNoHealthyNodes) to detect it.
	ErrNoHealthyNodes = errors.New("no healthy nodes available in any tier")
)
