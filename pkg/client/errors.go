package client

import (
	"errors"
	"fmt"
	"strings"

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

// Is matches a refusal to the verdicts this package names
// (ErrDelegateStakeShort, ErrDelegateBelowMinimum, ErrAccountExists,
// ErrCreateAccountFeeShort, ErrContractNotExist, ErrEstimateEnergyUnsupported),
// so a caller can branch on
// errors.Is instead of on java-tron's wording, which differs between releases
// and is prefixed differently by gRPC and HTTP. errors.As and the
// ErrInvalidTransaction match through Unwrap are unaffected.
func (e *ContractValidateError) Is(target error) bool {
	return isVerdict(e.Message, target)
}

// isVerdict reports whether msg, a node's refusal, is the verdict target names.
func isVerdict(msg string, target error) bool {
	// A switch, not a map: errors.Is hands Is any target, and hashing one of
	// an uncomparable type would panic where == just answers false.
	switch target {
	case ErrDelegateStakeShort:
		return refusalIsOneOf(msg, delegateStakeShortVerdicts)
	case ErrDelegateBelowMinimum:
		return refusalIsOneOf(msg, delegateBelowMinimumVerdicts)
	case ErrAccountExists:
		return refusalIsOneOf(msg, accountExistsVerdicts)
	case ErrCreateAccountFeeShort:
		return refusalIsOneOf(msg, createAccountFeeShortVerdicts)
	case ErrContractNotExist:
		return refusalIsOneOf(msg, contractNotExistVerdicts)
	case ErrEstimateEnergyUnsupported:
		return refusalIsOneOf(msg, estimateEnergyUnsupportedVerdicts)
	default:
		return false
	}
}

// The actuators' own texts (java-tron DelegateResourceActuator.validate). Each
// verdict is matched whole, so no other refusal about delegateBalance reads as
// one of them.
var (
	// GreatVoyage-v4.7.3 and later, then v4.7.0 to v4.7.2.
	delegateStakeShortVerdicts = []string{
		"delegateBalance must be less than or equal to available FreezeEnergyV2 balance",
		"delegateBalance must be less than or equal to available FreezeBandwidthV2 balance",
		"delegateBalance must be less than available FreezeEnergyV2 balance",
		"delegateBalance must be less than available FreezeBandwidthV2 balance",
	}
	// Both releases refuse under 1 TRX; v4.7.0 to v4.7.2 worded it as "more
	// than".
	delegateBelowMinimumVerdicts = []string{
		delegateBelowMinimumVerdict,
		"delegateBalance must be more than 1TRX",
	}
)

const delegateBelowMinimumVerdict = "delegateBalance must be greater than or equal to 1 TRX"

// The texts of the other verdicts, as GreatVoyage-v4.8.2.3 answers them over
// both transports (tests/local_build_refusal_test.go asks a node for each).
var (
	// CreateAccountActuator.validate.
	accountExistsVerdicts         = []string{"Account has existed"}
	createAccountFeeShortVerdicts = []string{"Validate CreateAccountActuator error, insufficient fee."}
	// Wallet.triggerConstantContract and Wallet.estimateEnergy for a
	// constant call or an estimate, Wallet.triggerContract for a call to be
	// built, and VMActuator for a call that reaches the VM without one.
	contractNotExistVerdicts = []string{
		"Smart contract is not exist.",
		"No contract or not a valid smart contract",
		"No contract or not a smart contract",
	}
	// Wallet.estimateEnergy on a node without vm.estimateEnergy, as the
	// public nodes answer it.
	estimateEnergyUnsupportedVerdicts = []string{"this node does not support estimate energy"}
)

// refusalIsOneOf reports whether msg is one of the verdicts, bare or after the
// prefix the node puts in front of the actuator's message: "Contract validate
// error : " over gRPC, the exception class and " : " over HTTP. The verdict
// has to end the message and start at a word, so a longer sentence that only
// contains one is not it.
func refusalIsOneOf(msg string, verdicts []string) bool {
	msg = strings.TrimSpace(msg)
	for _, v := range verdicts {
		if rest, ok := strings.CutSuffix(msg, v); ok && (rest == "" || strings.HasSuffix(rest, " ")) {
			return true
		}
	}
	return false
}

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

// ContractCallError is returned when a contract call did not run to
// completion, as a constant call (TriggerConstantContract) or an energy
// estimate (EstimateEnergy). Code says how: SUCCESS from a constant call whose
// VM failure (a revert) is in Message only, CONTRACT_EXE_ERROR from an
// estimate whose call failed in the VM, and CONTRACT_VALIDATE_ERROR from a
// constant call the node would not run at all (no contract at the address).
// An estimate the node would not run is a ContractValidateError instead.
//
// It matches ErrContractCallFailed, as every failed constant call did before
// the type existed, and ErrContractNotExist on that verdict.
type ContractCallError struct {
	// Code is the node's response code; SUCCESS (0) for a reverted constant
	// call, which the node reports through the message alone.
	Code api.ReturnResponseCode
	// Message is the node's reason, verbatim (gRPC puts "Contract validate
	// error : " in front of a refusal, HTTP does not).
	Message string
}

func (e *ContractCallError) Error() string {
	if e.Message != "" {
		return ErrContractCallFailed.Error() + ": " + e.Message
	}
	return ErrContractCallFailed.Error() + ": " + e.Code.String()
}

// Unwrap keeps errors.Is(err, ErrContractCallFailed) true.
func (e *ContractCallError) Unwrap() error { return ErrContractCallFailed }

// Is matches ErrContractNotExist: the address holds no contract.
func (e *ContractCallError) Is(target error) bool {
	return target == ErrContractNotExist && isVerdict(e.Message, target)
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

	// ErrContractCallFailed marks a contract call that did not complete - most
	// often a revert - as a constant call (any *ContractCallError from
	// TriggerConstantContract, a call to an address without a contract
	// included) or an estimate (a *ContractCallError from EstimateEnergy; an
	// estimate the node refuses to run is a *ContractValidateError).
	//
	// A reverted constant call is not signalled by the result code: the node answers
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

	// ErrDelegateStakeShort matches a DelegateResource the node refused because
	// the owner's stake of the resource, less what its own usage holds, is
	// below the amount. It is the chain's state now, not a malformed request:
	// the same delegation can build once the owner stakes more or its usage
	// recovers. Match it with errors.Is on the error DelegateResource returns.
	ErrDelegateStakeShort = errors.New("delegate balance exceeds the owner's available stake")

	// ErrDelegateBelowMinimum matches a DelegateResource for less than
	// MinDelegateBalance. No node builds one, so DelegateResource refuses it
	// before calling the node, with the node's own verdict; a refusal from the
	// node matches as well.
	ErrDelegateBelowMinimum = errors.New("delegate balance below the minimum delegation")

	// ErrAccountExists matches a refusal to create an account that is already
	// on chain (a ContractValidateError from CreateAccount).
	ErrAccountExists = errors.New("account already exists")

	// ErrCreateAccountFeeShort matches a CreateAccount the node refused
	// because the owner's balance does not hold the fee the system contract
	// burns for a new account (getCreateNewAccountFeeInSystemContract),
	// whatever bandwidth it has staked. It is the chain's state now: the same
	// request builds once the owner holds the fee.
	ErrCreateAccountFeeShort = errors.New("owner balance below the account creation fee")

	// ErrContractNotExist matches a call to an address that holds no
	// contract: an account, or an address the network has never seen. A
	// ContractValidateError from TriggerContract or EstimateEnergy matches it,
	// and so does a ContractCallError from TriggerConstantContract. Every
	// node and every retry answers the same until a contract is deployed
	// there.
	ErrContractNotExist = errors.New("contract does not exist")

	// ErrEstimateEnergyUnsupported matches the ContractValidateError of an
	// EstimateEnergy on a node that does not serve the estimate API (the
	// public nodes do not): another node may, and TriggerConstantContract
	// prices a call on any.
	ErrEstimateEnergyUnsupported = errors.New("node does not support estimate energy")

	// ErrNoHealthyNodes is returned when no node in any tier is currently
	// marked healthy. The health-checker runs continuously and will return
	// nodes to the pool as soon as they recover; callers should retry with
	// backoff. Use errors.Is(err, client.ErrNoHealthyNodes) to detect it.
	ErrNoHealthyNodes = errors.New("no healthy nodes available in any tier")
)
