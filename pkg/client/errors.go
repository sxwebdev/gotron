package client

import (
	"errors"
	"fmt"

	"github.com/sxwebdev/gotron/pkg/units"
	"github.com/sxwebdev/gotron/schema/pb/api"
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

// BroadcastError is returned by Client.BroadcastTransaction when the node
// rejects a transaction.
//
// The node routinely answers with Result=false and an empty Message while the
// only diagnostic is Code (DUP_TRANSACTION_ERROR, SIGERROR, BANDWITH_ERROR,
// TAPOS_ERROR, ...). Code is therefore always rendered, so the message never
// degrades to a bare "result error:". Callers should branch on Code via
// errors.As rather than matching on the message text.
type BroadcastError struct {
	// Code is the node's response code. Note that a rejection can carry
	// Return_SUCCESS (0) when the node reports failure through Result=false
	// alone, so a zero Code does not mean the broadcast succeeded.
	Code api.ReturnResponseCode
	// Message is the node's human-readable reason. Frequently empty.
	Message string
}

func (e *BroadcastError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("broadcast rejected: code=%s", e.Code)
	}
	return fmt.Sprintf("broadcast rejected: code=%s: %s", e.Code, e.Message)
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
	//
	// ErrInvalidAmount is units.ErrInvalidAmount rather than a second sentinel
	// with the same text: the amount constructors live in pkg/units, which
	// cannot import this package, and a caller must not have to know which
	// layer rejected the amount.
	ErrInvalidAmount           = units.ErrInvalidAmount
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
