package client

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
	"google.golang.org/protobuf/proto"
)

// A real /wallet/freezebalancev2 response captured from a mainnet node. The
// transaction is at the top level, not wrapped in a TransactionExtention.
const liveFreezeResponse = `{"raw_data":{"ref_block_bytes":"cc02","ref_block_hash":"c1171222b68f923e","expiration":1785238218000,"contract":[{"parameter":{"value":{"owner_address":"TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t","frozen_balance":1000000,"resource":"ENERGY"},"type_url":"type.googleapis.com/protocol.FreezeBalanceV2Contract"},"type":"FreezeBalanceV2Contract"}],"timestamp":1785238160928},"raw_data_hex":"0a02cc022208c1171222b68f923e4090caf5c3fa335a59083612550a34747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e467265657a6542616c616e63655632436f6e7472616374121d0a1541a614f803b6fd780986a42c78ec9c7f77e6ded13c10c0843d180170a08cf2c3fa33","txID":"233666eb2d145d3c577a308b362d3bc101cc8e0beb018b901ea819e099cb8cfc","visible":true}`

// newStubTransport serves body for every request and records the last request body.
func newStubTransport(t *testing.T, status int, body string) (*HTTPTransport, *map[string]any) {
	t.Helper()

	var lastReq map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		lastReq = nil
		_ = json.Unmarshal(raw, &lastReq)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	tr, err := NewHTTPTransport(NodeConfig{Protocol: ProtocolHTTP, Address: srv.URL})
	require.NoError(t, err)

	return tr, &lastReq
}

func TestHTTPFreezeBalanceV2ParsesTopLevelTransaction(t *testing.T) {
	tr, lastReq := newStubTransport(t, http.StatusOK, liveFreezeResponse)

	owner, err := tronutils.DecodeCheck(testAddr)
	require.NoError(t, err)

	tx, err := tr.FreezeBalanceV2(t.Context(), &core.FreezeBalanceV2Contract{
		OwnerAddress:  owner,
		FrozenBalance: 1_000_000,
		Resource:      core.ResourceCode_ENERGY,
	})
	require.NoError(t, err)

	// The transaction must actually be there - the bug this guards against is a
	// silently empty TransactionExtention with a nil error.
	require.NotNil(t, tx.GetTransaction().GetRawData())
	require.Len(t, tx.GetTransaction().GetRawData().GetContract(), 1)
	require.Equal(
		t,
		core.Transaction_Contract_FreezeBalanceV2Contract,
		tx.GetTransaction().GetRawData().GetContract()[0].GetType(),
	)

	// Re-serializing raw_data must reproduce txID, i.e. the transaction is
	// byte-identical to what the node built and is safe to sign.
	rawData, err := proto.Marshal(tx.GetTransaction().GetRawData())
	require.NoError(t, err)
	sum := sha256.Sum256(rawData)
	require.Equal(t, hex.EncodeToString(sum[:]), hex.EncodeToString(tx.GetTxid()))

	// Request shape.
	require.Equal(t, testAddr, (*lastReq)["owner_address"])
	require.Equal(t, "ENERGY", (*lastReq)["resource"])
	require.EqualValues(t, 1_000_000, (*lastReq)["frozen_balance"])
	require.Equal(t, true, (*lastReq)["visible"])
}

func TestHTTPTxRequestSurfacesNodeError(t *testing.T) {
	// Tron reports contract validation failures with HTTP 200 and an Error field.
	const body = `{"Error":"class org.tron.core.exception.ContractValidateException : frozenBalance must be positive"}`

	tr, _ := newStubTransport(t, http.StatusOK, body)

	_, err := tr.FreezeBalanceV2(t.Context(), &core.FreezeBalanceV2Contract{OwnerAddress: []byte{0x41}})
	require.Error(t, err)
	require.ErrorContains(t, err, "frozenBalance must be positive")

	// The generic "no raw_data_hex" fallback embeds the whole response body in its
	// message, so ErrorContains alone would pass even with the Error handling
	// removed. Pinning the sentinel is what actually distinguishes the two paths.
	require.NotErrorIs(t, err, ErrInvalidTransaction)
}

func TestHTTPTxRequestRejectsEmptyBody(t *testing.T) {
	tr, _ := newStubTransport(t, http.StatusOK, `{}`)

	_, err := tr.WithdrawExpireUnfreeze(t.Context(), &core.WithdrawExpireUnfreezeContract{OwnerAddress: []byte{0x41}})
	require.ErrorIs(t, err, ErrInvalidTransaction)
}

func TestHTTPTxRequestRejectsMalformedHex(t *testing.T) {
	tr, _ := newStubTransport(t, http.StatusOK, `{"raw_data_hex":"zzzz","txID":"aa"}`)

	_, err := tr.CancelAllUnfreezeV2(t.Context(), &core.CancelAllUnfreezeV2Contract{OwnerAddress: []byte{0x41}})
	require.Error(t, err)
	require.ErrorContains(t, err, "raw_data_hex")
}

func TestHTTPGetAccountMapsStakeBalances(t *testing.T) {
	// Shape captured from a live node: "type" is omitted for BANDWIDTH and
	// "amount" is omitted when zero.
	const body = `{"address":"TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t","balance":42,
		"frozenV2":[{"amount":13000000000},{"type":"ENERGY","amount":75550000000},{"type":"TRON_POWER"}],
		"unfrozenV2":[{"type":"ENERGY","unfreeze_amount":5000000,"unfreeze_expire_time":1785236015936}]}`

	tr, _ := newStubTransport(t, http.StatusOK, body)

	owner, err := tronutils.DecodeCheck(testAddr)
	require.NoError(t, err)

	acc, err := tr.GetAccount(t.Context(), &core.Account{Address: owner})
	require.NoError(t, err)

	require.Len(t, acc.GetFrozenV2(), 3)
	require.Equal(t, core.ResourceCode_BANDWIDTH, acc.GetFrozenV2()[0].GetType())
	require.EqualValues(t, 13_000_000_000, acc.GetFrozenV2()[0].GetAmount())
	require.Equal(t, core.ResourceCode_ENERGY, acc.GetFrozenV2()[1].GetType())
	require.EqualValues(t, 75_550_000_000, acc.GetFrozenV2()[1].GetAmount())
	require.Equal(t, core.ResourceCode_TRON_POWER, acc.GetFrozenV2()[2].GetType())

	require.Len(t, acc.GetUnfrozenV2(), 1)
	require.Equal(t, core.ResourceCode_ENERGY, acc.GetUnfrozenV2()[0].GetType())
	require.EqualValues(t, 5_000_000, acc.GetUnfrozenV2()[0].GetUnfreezeAmount())
	require.EqualValues(t, 1785236015936, acc.GetUnfrozenV2()[0].GetUnfreezeExpireTime())
}

func TestHTTPGetRewardInfoReadsRewardField(t *testing.T) {
	// The node answers with "reward", not NumberMessage's "num". Parsing it as a
	// NumberMessage compiles, never errors, and always yields 0.
	tr, _ := newStubTransport(t, http.StatusOK, `{"reward":123456}`)

	owner, err := tronutils.DecodeCheck(testAddr)
	require.NoError(t, err)

	res, err := tr.GetRewardInfo(t.Context(), owner)
	require.NoError(t, err)
	require.EqualValues(t, 123456, res.GetNum())
}

func TestHTTPGetBrokerageInfoReadsBrokerageField(t *testing.T) {
	tr, _ := newStubTransport(t, http.StatusOK, `{"brokerage":20}`)

	owner, err := tronutils.DecodeCheck(testAddr)
	require.NoError(t, err)

	res, err := tr.GetBrokerageInfo(t.Context(), owner)
	require.NoError(t, err)
	require.EqualValues(t, 20, res.GetNum())
}

func TestHTTPListWitnessesDecodesHexAddresses(t *testing.T) {
	// /wallet/listwitnesses ignores "visible" and always returns hex addresses.
	const body = `{"witnesses":[
		{"address":"41d376d829440505ea13c9d1c455317d51b62e4ab6","voteCount":1345312795,"url":"http://blockchain.org","totalProduced":10,"totalMissed":1,"latestBlockNum":7,"latestSlotNum":3,"isJobs":true}
	]}`

	tr, _ := newStubTransport(t, http.StatusOK, body)

	res, err := tr.ListWitnesses(t.Context())
	require.NoError(t, err)
	require.Len(t, res.GetWitnesses(), 1)

	w := res.GetWitnesses()[0]
	require.Len(t, w.GetAddress(), 21)
	require.EqualValues(t, 0x41, w.GetAddress()[0])
	require.Equal(t, "http://blockchain.org", w.GetUrl())
	require.EqualValues(t, 1345312795, w.GetVoteCount())
	require.True(t, w.GetIsJobs())
}

func TestHTTPGetCanWithdrawUnfreezeAmountSendsTimestamp(t *testing.T) {
	tr, lastReq := newStubTransport(t, http.StatusOK, `{"amount":777}`)

	owner, err := tronutils.DecodeCheck(testAddr)
	require.NoError(t, err)

	res, err := tr.GetCanWithdrawUnfreezeAmount(t.Context(), &api.CanWithdrawUnfreezeAmountRequestMessage{OwnerAddress: owner, Timestamp: 1785236015936})
	require.NoError(t, err)
	require.EqualValues(t, 777, res.GetAmount())
	require.EqualValues(t, 1785236015936, (*lastReq)["timestamp"])
}

// The node reports broadcast rejections with a plain-text "message", but
// api.Return.Message is a protobuf bytes field that protojson decodes as base64.
// Unmarshaling straight into api.Return fails, so the caller used to get an opaque
// protojson error instead of a BroadcastError it could branch on.
func TestHTTPBroadcastTransactionSurfacesCodeAndMessage(t *testing.T) {
	// Verbatim response from tron-rpc.publicnode.com for a badly signed transaction.
	const body = `{"code":"SIGERROR","message":"Validate signature error: java.lang.IllegalArgumentException: Invalid point compression","txid":"dcf138efcf48986b970cee315ca769ab48485f413aa6cb8893ebdc14ea85c207"}`

	tr, _ := newStubTransport(t, http.StatusOK, body)

	ret, err := tr.BroadcastTransaction(t.Context(), &core.Transaction{})
	require.NoError(t, err, "the transport must decode the rejection, not fail on it")
	require.False(t, ret.GetResult())
	require.Equal(t, api.Return_SIGERROR, ret.GetCode())
	require.Contains(t, string(ret.GetMessage()), "Invalid point compression")

	// End to end: the caller must be able to branch on the code.
	c := &Client{transport: tr}
	_, err = c.BroadcastTransaction(t.Context(), &core.Transaction{})

	var broadcastErr *BroadcastError
	require.ErrorAs(t, err, &broadcastErr)
	require.Equal(t, api.Return_SIGERROR, broadcastErr.Code)
	require.Contains(t, broadcastErr.Message, "Invalid point compression")
}

func TestHTTPBroadcastTransactionSuccess(t *testing.T) {
	tr, _ := newStubTransport(t, http.StatusOK,
		`{"result":true,"txid":"dcf138efcf48986b970cee315ca769ab48485f413aa6cb8893ebdc14ea85c207"}`)

	ret, err := tr.BroadcastTransaction(t.Context(), &core.Transaction{})
	require.NoError(t, err)
	require.True(t, ret.GetResult())
	require.Equal(t, api.Return_SUCCESS, ret.GetCode())
}

// A malformed request answers with a top-level "Error" and no code at all.
func TestHTTPBroadcastTransactionTopLevelError(t *testing.T) {
	tr, _ := newStubTransport(t, http.StatusOK,
		`{"Error":"class java.lang.NullPointerException : null"}`)

	_, err := tr.BroadcastTransaction(t.Context(), &core.Transaction{})
	require.ErrorContains(t, err, "NullPointerException")
}

// The doTxRequest retrofit onto the seven pre-existing tx-creating methods had no
// test: reverting any of them to the old doRequest call left the suite green while
// the method silently returned an empty TransactionExtention.
func TestHTTPTxCreatingMethodsParseTopLevelTransaction(t *testing.T) {
	owner, err := tronutils.DecodeCheck(testAddr)
	require.NoError(t, err)
	other, err := tronutils.DecodeCheck(testAddr2)
	require.NoError(t, err)

	calls := map[string]func(*HTTPTransport) (*api.TransactionExtention, error){
		"CreateTransaction": func(tr *HTTPTransport) (*api.TransactionExtention, error) {
			return tr.CreateTransaction(t.Context(), &core.TransferContract{OwnerAddress: owner, ToAddress: other, Amount: 1})
		},
		"CreateAccount": func(tr *HTTPTransport) (*api.TransactionExtention, error) {
			return tr.CreateAccount(t.Context(), &core.AccountCreateContract{OwnerAddress: owner, AccountAddress: other})
		},
		"DeployContract": func(tr *HTTPTransport) (*api.TransactionExtention, error) {
			return tr.DeployContract(t.Context(), &core.CreateSmartContract{
				OwnerAddress: owner,
				NewContract:  &core.SmartContract{Name: "x", Bytecode: []byte{0x60}},
			})
		},
		"UpdateSetting": func(tr *HTTPTransport) (*api.TransactionExtention, error) {
			return tr.UpdateSetting(t.Context(), &core.UpdateSettingContract{OwnerAddress: owner, ContractAddress: other})
		},
		"UpdateEnergyLimit": func(tr *HTTPTransport) (*api.TransactionExtention, error) {
			return tr.UpdateEnergyLimit(t.Context(), &core.UpdateEnergyLimitContract{OwnerAddress: owner, ContractAddress: other})
		},
		"DelegateResource": func(tr *HTTPTransport) (*api.TransactionExtention, error) {
			return tr.DelegateResource(t.Context(), &core.DelegateResourceContract{OwnerAddress: owner, ReceiverAddress: other, Balance: 1})
		},
		"UnDelegateResource": func(tr *HTTPTransport) (*api.TransactionExtention, error) {
			return tr.UnDelegateResource(t.Context(), &core.UnDelegateResourceContract{OwnerAddress: owner, ReceiverAddress: other, Balance: 1})
		},
	}

	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			tr, _ := newStubTransport(t, http.StatusOK, liveFreezeResponse)

			tx, err := call(tr)
			require.NoError(t, err)

			// The old doRequest path produced an empty message with a nil error.
			require.NotNil(t, tx.GetTransaction().GetRawData(), "transaction must be decoded")
			require.NotEmpty(t, tx.GetTransaction().GetRawData().GetContract())
			require.NotEmpty(t, tx.GetTxid())

			rawData, err := proto.Marshal(tx.GetTransaction().GetRawData())
			require.NoError(t, err)
			sum := sha256.Sum256(rawData)
			require.Equal(t, hex.EncodeToString(sum[:]), hex.EncodeToString(tx.GetTxid()))
		})
	}
}
