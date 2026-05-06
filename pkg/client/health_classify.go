package client

import (
	"context"
	"errors"
	"io"
	"net"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// isNetworkError reports whether err looks like a transport/network failure
// rather than a server-side logical error. Only network errors count toward
// the per-node failure threshold and trigger tier-based fallback; logical
// errors (e.g. invalid arguments) are returned to the caller as-is and leave
// node health untouched.
//
// The classifier is conservative: when in doubt it returns false, so a healthy
// node is never evicted because of an ambiguous error. Callers can override
// the rules via HealthConfig.ClassifyErr.
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}

	// Unwrap TransportError so we look at the underlying cause.
	if te, ok := errors.AsType[*TransportError](err); ok {
		err = te.Err
	}

	// Context: a deadline that fired locally still indicates the network
	// did not respond in time, but explicit Cancel is not the node's fault.
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	if errors.Is(err, context.Canceled) {
		return false
	}

	// gRPC status codes — we have a wire-level reply with a structured code.
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.Unavailable,
			codes.DeadlineExceeded,
			codes.Aborted,
			codes.ResourceExhausted,
			codes.Internal,
			codes.Unknown:
			return true
		case codes.InvalidArgument,
			codes.NotFound,
			codes.AlreadyExists,
			codes.PermissionDenied,
			codes.Unauthenticated,
			codes.FailedPrecondition,
			codes.OutOfRange,
			codes.Unimplemented:
			return false
		}
	}

	// HTTP status: 5xx + 408 (request timeout) + 429 (rate limit) point at the
	// node having trouble; everything else (4xx) is a client-side logical issue.
	if he, ok := errors.AsType[*HTTPStatusError](err); ok {
		if he.Code >= 500 || he.Code == 408 || he.Code == 429 {
			return true
		}
		return false
	}

	// Pure net.Error timeouts (e.g. DNS, dial).
	if nerr, ok := errors.AsType[net.Error](err); ok && nerr.Timeout() {
		return true
	}

	// Connection-level breakages.
	if _, ok := errors.AsType[*net.OpError](err); ok {
		return true
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	if errors.Is(err, net.ErrClosed) {
		return true
	}

	return false
}
