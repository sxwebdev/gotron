package client

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

// newRoutedTransport serves a different body per endpoint, which the delegation
// walk needs: it reads the index and then one record per receiver it names.
//
// A request to an endpoint with no body registered fails the test rather than
// answering "{}", so a lookup that goes somewhere unexpected is not mistaken
// for one that found nothing.
func newRoutedTransport(t *testing.T, routes map[string]string) *HTTPTransport {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := routes[r.URL.Path]
		if !ok {
			t.Errorf("unexpected request to %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	tr, err := NewHTTPTransport(NodeConfig{Protocol: ProtocolHTTP, Address: srv.URL})
	require.NoError(t, err)

	return tr
}

// Bodies a live node returns, verbatim, for a request it will not process. They
// arrive with HTTP 200 and no other marker, so a decoder that does not look for
// the field builds the zero message and reports success.
const (
	refusalInvalidHex = `{"Error":"class org.tron.core.services.http.JsonFormat$ParseException : 1:10: INVALID hex String"}`
	refusalBadInteger = `{"Error":"class org.tron.core.services.http.JsonFormat$ParseException : ` +
		`1:8: Couldn't parse integer: For input string: \"\"abc\"\""}`
)

// Every decoding path has to look for the refusal, not just the protojson one:
// an empty block with a nil error reads as "that block is empty", and an empty
// contract as "there is no contract at that address" - both of which a caller
// acts on, while the request that produced them was never run.
func TestHTTPDecodingPathsSurfaceNodeRefusal(t *testing.T) {
	address, err := tronutils.DecodeCheck(testAddr)
	require.NoError(t, err)

	cases := []struct {
		name     string
		endpoint string
		body     string
		call     func(*HTTPTransport) error
	}{
		{
			// doBlockRequest
			name:     "block by num",
			endpoint: "/wallet/getblockbynum",
			body:     refusalBadInteger,
			call: func(tr *HTTPTransport) error {
				_, err := tr.GetBlockByNum(t.Context(), 1)

				return err
			},
		},
		{
			// doBlockListRequest
			name:     "block list",
			endpoint: "/wallet/getblockbylimitnext",
			body:     refusalBadInteger,
			call: func(tr *HTTPTransport) error {
				_, err := tr.GetBlockByLimitNext(t.Context(), 1, 2)

				return err
			},
		},
		{
			// doRequestTransformed
			name:     "contract",
			endpoint: "/wallet/getcontract",
			body:     refusalInvalidHex,
			call: func(tr *HTTPTransport) error {
				_, err := tr.GetContract(t.Context(), address)

				return err
			},
		},
		{
			// doRequest, which still backs a dozen endpoints - the asset reads
			// this case used to name have since moved to fetchJSON, which the
			// "reward" case below already covers.
			name:     "transaction by id",
			endpoint: "/wallet/gettransactionbyid",
			body:     refusalInvalidHex,
			call: func(tr *HTTPTransport) error {
				_, err := tr.GetTransactionById(t.Context(), []byte{0x01})

				return err
			},
		},
		{
			// fetchJSON
			name:     "reward",
			endpoint: "/wallet/getReward",
			body:     refusalInvalidHex,
			call: func(tr *HTTPTransport) error {
				_, err := tr.GetRewardInfo(t.Context(), address)

				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tr := newRoutedTransport(t, map[string]string{tc.endpoint: tc.body})

			err := tc.call(tr)
			// The sentinel is what a caller matches on; the node's own sentence
			// is kept alongside it because it names the offending field.
			require.ErrorIs(t, err, ErrNodeRefusedRequest)
			require.ErrorContains(t, err, "JsonFormat$ParseException")
		})
	}
}

// A refusal is about this request, so it must not be reported as a transaction
// that does not exist - the caller would go on believing the hash they hold is
// simply not on chain yet.
func TestHTTPRefusalIsNotAMissingTransaction(t *testing.T) {
	tr := newRoutedTransport(t, map[string]string{"/wallet/gettransactionbyid": refusalInvalidHex})

	_, err := tr.GetTransactionById(t.Context(), []byte{0x01})
	require.ErrorIs(t, err, ErrNodeRefusedRequest)
	require.NotErrorIs(t, err, ErrTransactionNotFound)
}

// encoding/json matches field names case-insensitively, so a body carrying a
// key of its own spelled "error" - a gateway diagnostic alongside a valid
// payload - must not be mistaken for the node's "Error".
func TestHTTPRefusalCheckIsCaseSensitive(t *testing.T) {
	tr := newRoutedTransport(t, map[string]string{
		"/wallet/getReward": `{"error":"upstream note","reward":42}`,
	})

	address, err := tronutils.DecodeCheck(testAddr)
	require.NoError(t, err)

	reward, err := tr.GetRewardInfo(t.Context(), address)
	require.NoError(t, err)
	require.Equal(t, int64(42), reward.GetNum())
}

// A refusal is about the request, not the node. Counting it as a network
// failure would evict a healthy node over a caller's own malformed argument,
// and enough of them would empty the pool.
func TestHTTPRefusalIsNotANetworkError(t *testing.T) {
	tr := newRoutedTransport(t, map[string]string{"/wallet/getblockbynum": refusalBadInteger})

	_, err := tr.GetBlockByNum(t.Context(), 1)
	require.Error(t, err)
	require.False(t, isNetworkError(err))
}

// A body that is not a JSON object carries no such field, and probing it must
// not turn a valid answer into an error: this endpoint answers with an array.
func TestHTTPRefusalCheckIgnoresNonObjectBodies(t *testing.T) {
	tr := newRoutedTransport(t, map[string]string{
		"/wallet/gettransactioninfobyblocknum": `[{"id":"0a0b","blockNumber":42}]`,
	})

	infos, err := tr.GetTransactionInfoByBlockNum(t.Context(), 42)
	require.NoError(t, err)
	require.Len(t, infos.GetTransactionInfo(), 1)
	require.Equal(t, int64(42), infos.GetTransactionInfo()[0].GetBlockNumber())
}

// The transaction endpoints report the same refusal, and there it has to keep
// its type so a caller matches it the way they match the gRPC equivalent.
func TestHTTPTransactionRefusalStaysTyped(t *testing.T) {
	tr := newRoutedTransport(t, map[string]string{
		"/wallet/createtransaction": `{"Error":"class org.tron.core.exception.ContractValidateException : ` +
			`Validate TransferContract error, balance is not sufficient."}`,
	})

	owner, err := tronutils.DecodeCheck(testAddr)
	require.NoError(t, err)
	to, err := tronutils.DecodeCheck(testAddr2)
	require.NoError(t, err)

	_, err = tr.CreateTransaction(t.Context(), &core.TransferContract{
		OwnerAddress: owner,
		ToAddress:    to,
		Amount:       1,
	})

	var refusal *ContractValidateError
	require.ErrorAs(t, err, &refusal)
	require.Contains(t, refusal.Message, "balance is not sufficient")
}
