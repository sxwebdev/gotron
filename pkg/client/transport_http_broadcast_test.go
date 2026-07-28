package client

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// signedTransfer builds a signed TransferContract transaction shaped like one a
// node would hand back, so the encoding assertions below run against a body with
// every kind of field that used to be mangled: raw bytes, addresses inside a
// contract Any, and a signature.
func signedTransfer(t *testing.T) *core.Transaction {
	t.Helper()

	owner, err := tronutils.DecodeCheck(testAddr)
	require.NoError(t, err)
	to, err := tronutils.DecodeCheck(testAddr2)
	require.NoError(t, err)

	param, err := anypb.New(&core.TransferContract{
		OwnerAddress: owner,
		ToAddress:    to,
		Amount:       100_000,
	})
	require.NoError(t, err)

	return &core.Transaction{
		RawData: &core.TransactionRaw{
			RefBlockBytes: []byte{0xb5, 0xd3},
			RefBlockHash:  []byte{0x8e, 0x47, 0x82, 0x27, 0x23, 0x00, 0xfc, 0xb5},
			Expiration:    1785249621000,
			Timestamp:     1785249562174,
			Contract: []*core.Transaction_Contract{{
				Type:      core.Transaction_Contract_TransferContract,
				Parameter: param,
			}},
		},
		Signature: [][]byte{{0x32, 0xf8, 0xf4, 0xc3, 0x56, 0x1a, 0xcb, 0x23}},
	}
}

// A signed transaction has to reach the node as the exact bytes that were
// signed. protojson renders bytes as base64 and a contract Any as "@type", a
// dialect no Tron node reads: it answered
// "class java.lang.NullPointerException : null" and every HTTP broadcast failed
// while the same transaction went through over gRPC.
//
// Probed against Nile, /wallet/broadcasttransaction rebuilds the transaction
// from the JSON "raw_data" object and ignores "raw_data_hex" and "txID"
// completely - sending those two alone still returns the NullPointerException.
// /wallet/broadcasthex takes the marshalled protobuf instead, which is why the
// request body must be exactly proto.Marshal(tx).
func TestHTTPBroadcastSendsProtobufHex(t *testing.T) {
	tr, lastReq := newStubTransport(t, http.StatusOK,
		`{"result":true,"code":"SUCCESS","txid":"cc2204ed2a452dd65dcddc2fb13f4ef977b808a41e00d6db314bbcdc4328d900"}`)

	tx := signedTransfer(t)

	res, err := tr.BroadcastTransaction(t.Context(), tx)
	require.NoError(t, err)
	require.True(t, res.GetResult())
	require.Equal(t, api.Return_SUCCESS, res.GetCode())

	req := *lastReq
	require.NotNil(t, req)

	sent, ok := req["transaction"].(string)
	require.True(t, ok, "the node reads the transaction from the %q field, got body %v", "transaction", req)

	// Byte-for-byte identity with the marshalled transaction. Anything less -
	// a re-encoded field, a swapped pair, a dropped contract - changes the hash
	// the signature covers.
	wantBytes, err := proto.Marshal(tx)
	require.NoError(t, err)
	require.Equal(t, hex.EncodeToString(wantBytes), sent)

	// The bytes must round-trip back into the same transaction, which is what
	// the node does before checking the signature.
	decoded, err := hex.DecodeString(sent)
	require.NoError(t, err)
	got := &core.Transaction{}
	require.NoError(t, proto.Unmarshal(decoded, got))
	require.True(t, proto.Equal(tx, got), "the transaction must survive the round-trip unchanged")

	// The node derives the txID as sha256 of the marshalled raw_data and checks
	// the signature against it. That derivation must land on the same hash the
	// caller signed, or the node reports SIGERROR for a correctly signed
	// transaction.
	wantRaw, err := proto.Marshal(tx.GetRawData())
	require.NoError(t, err)
	gotRaw, err := proto.Marshal(got.GetRawData())
	require.NoError(t, err)
	require.Equal(t, sha256.Sum256(wantRaw), sha256.Sum256(gotRaw))

	// The contract Any has to arrive intact; dropping it is what produced
	// "Contract validate error : No contract!" on a body the node did parse.
	require.Len(t, got.GetRawData().GetContract(), 1)
	require.Equal(t, tx.GetSignature(), got.GetSignature())
}

// The body must contain no base64 and no protojson Any marker - those are the
// fingerprints of the encoding the node rejects.
func TestHTTPBroadcastBodyCarriesNoProtojsonArtifacts(t *testing.T) {
	tr, lastReq := newStubTransport(t, http.StatusOK, `{"result":true,"code":"SUCCESS"}`)

	tx := signedTransfer(t)
	_, err := tr.BroadcastTransaction(t.Context(), tx)
	require.NoError(t, err)

	req := *lastReq
	require.Len(t, req, 1, "broadcasthex takes only the transaction field, got %v", req)

	sent, ok := req["transaction"].(string)
	require.True(t, ok)

	// Hex, and only hex.
	_, err = hex.DecodeString(sent)
	require.NoError(t, err, "the transaction field must be hex")
	require.NotContains(t, sent, "=", "base64 padding means protojson encoding leaked back in")
	require.NotContains(t, sent, "/")
	require.NotContains(t, sent, "+")

	// The old body nested the transaction under raw_data/signature and marked
	// the contract with @type. None of that may be present.
	require.NotContains(t, req, "raw_data")
	require.NotContains(t, req, "signature")
	require.NotContains(t, req, "visible")

	txBytes, err := proto.Marshal(tx)
	require.NoError(t, err)
	require.Equal(t, len(txBytes)*2, len(sent), "hex is exactly two characters per byte")

	// The contract's type URL must survive as protobuf bytes inside the payload:
	// it is what tells the node which contract class to build, and losing it was
	// the difference between a parsed body and a NullPointerException.
	require.Contains(t, string(txBytes), "type.googleapis.com/protocol.TransferContract")
	require.NotContains(t, strings.ToLower(sent), "type.googleapis",
		"the type URL must be hex-encoded, not embedded as JSON text")
}

// A node that cannot parse the body answers HTTP 200 with a top-level "Error",
// which must surface as an error rather than an empty api.Return.
func TestHTTPBroadcastSurfacesNodeError(t *testing.T) {
	tr, _ := newStubTransport(t, http.StatusOK,
		`{"Error":"class java.lang.NullPointerException : null"}`)

	_, err := tr.BroadcastTransaction(t.Context(), signedTransfer(t))
	require.Error(t, err)
	require.ErrorContains(t, err, "NullPointerException")
}

// A rejection must keep reaching Client.BroadcastTransaction as a typed
// BroadcastError carrying the node's response code.
func TestHTTPBroadcastRejectionKeepsCode(t *testing.T) {
	tr, _ := newStubTransport(t, http.StatusOK,
		`{"result":false,"code":"SIGERROR","message":"Validate signature error: ... is signed by T... but it is not contained of permission.","txid":"844c"}`)

	res, err := tr.BroadcastTransaction(t.Context(), signedTransfer(t))
	require.NoError(t, err, "a rejection is a valid response at the transport layer")
	require.False(t, res.GetResult())
	require.Equal(t, api.Return_SIGERROR, res.GetCode())
	require.Contains(t, string(res.GetMessage()), "Validate signature error")
}
