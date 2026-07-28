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

// ContractValidateError is returned when a node refuses to *build* a
// transaction because the request itself is invalid - a negative amount, an
// account that does not exist, nothing to unfreeze, and so on.
//
// It is a distinct type because such a refusal says nothing about the node: it
// is the caller's request that is wrong, and retrying it against another node
// gives the same answer. Without a type the only way to tell it apart from a
// node problem was to match on the message text, and the two arrive by
// different routes - gRPC returns a transaction carrying an error Result, HTTP
// answers 200 with an "Error" field - so the text differed by transport too.
//
// Errors of this type never count toward a node's health: isNetworkError is
// conservative and answers false for anything it does not recognise as a
// transport failure. TestValidateErrorsDoNotMarkANodeUnhealthy pins that, so a
// future classifier rule cannot start evicting nodes over a bad request.
type ContractValidateError struct {
	// Code is the node's response code. It is zero when the node did not send
	// one, which is the normal case over HTTP - a zero Code therefore does not
	// mean success.
	Code api.ReturnResponseCode
	// Message is the node's reason, verbatim. java-tron usually prefixes it
	// with the exception class, e.g.
	// "class org.tron.core.exception.ContractValidateException : ...".
	Message string
}

// Unwrap keeps errors.Is(err, ErrInvalidTransaction) true. A refused request is
// a kind of invalid transaction, and callers matching that sentinel before this
// type existed should not have to change.
func (e *ContractValidateError) Unwrap() error { return ErrInvalidTransaction }

func (e *ContractValidateError) Error() string {
	switch {
	case e.Message != "" && e.Code != 0:
		return fmt.Sprintf("contract validate error: %s: %s", e.Code, e.Message)
	case e.Message != "":
		return "contract validate error: " + e.Message
	case e.Code != 0:
		return fmt.Sprintf("contract validate error: %s", e.Code)
	default:
		return "contract validate error"
	}
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

	// ErrContractCallFailed marks a constant contract call the node executed but
	// that did not complete - most often a revert.
	//
	// Such a call is not signalled by the result code: the node answers
	// result.result = true with code SUCCESS and reports the failure only in
	// result.message ("REVERT opcode executed"), leaving constant_result empty
	// and energy_used at whatever was burned before the revert. Treating that as
	// success is how an estimate for a transfer that cannot succeed comes back
	// an order of magnitude too cheap.
	ErrContractCallFailed = errors.New("contract call failed")

	// ErrNodeRefusedRequest marks a request an HTTP node would not process at
	// all - a malformed address, an unparseable number. The /wallet endpoints
	// report it as HTTP 200 with an "Error" field rather than a status code, so
	// without a sentinel it is only distinguishable from a genuine transport
	// failure by substring-matching a Java class name.
	//
	// It says the request was wrong, not that the node is: the health checker
	// leaves node health untouched, and retrying the same call elsewhere returns
	// the same refusal. The transaction-creating endpoints report the same class
	// of refusal as a ContractValidateError, which carries the code gRPC gives.
	ErrNodeRefusedRequest = errors.New("node refused the request")

	// ErrNoHealthyNodes is returned when no node in any tier is currently
	// marked healthy. The health-checker runs continuously and will return
	// nodes to the pool as soon as they recover; callers should retry with
	// backoff. Use errors.Is(err, client.ErrNoHealthyNodes) to detect it.
	ErrNoHealthyNodes = errors.New("no healthy nodes available in any tier")
)
