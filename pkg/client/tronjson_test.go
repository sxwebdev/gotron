package client

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// Verbatim answers of a java-tron 4.8.2 private network, requested without
// "visible": every bytes field is hex.
const (
	liveReceiptWithLog = `{"id": "0ca17a0da65cb2df52be4e5edd8654358f7ad2b6f083180e719784ff8ff1c8ae","fee": 2965000,` +
		`"blockNumber": 156,"blockTimeStamp": 1790865477000,` +
		`"contractResult": ["0000000000000000000000000000000000000000000000000000000000000000"],` +
		`"contract_address": "41d53adb8bb7b45de60d0dc3d03ba0c187093b919d",` +
		`"receipt": {"energy_fee": 2965000,"energy_usage_total": 29650,"net_usage": 345,"result": "SUCCESS"},` +
		`"log": [{"address": "d53adb8bb7b45de60d0dc3d03ba0c187093b919d","topics": ` +
		`["ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",` +
		`"0000000000000000000000007e5f4552091a69125d5dfcb7b8c2659029395bdf",` +
		`"000000000000000000000000dcf2957055e8fcde477a077ecfeddf5ab2357547"],` +
		`"data": "00000000000000000000000000000000000000000000000000000000000f4240"}]}`

	liveReceiptWithInternalTx = `{"id": "7c44d197b2dcdc7452663f1d7fab7636e06aeb29393f7cd32074a39dde9d4139","fee": 676100,` +
		`"blockNumber": 158,"blockTimeStamp": 1790865489000,"contractResult": [""],` +
		`"contract_address": "41a4de5410785ffffc7c210f126f7534d868c2ecc9",` +
		`"receipt": {"energy_fee": 676100,"energy_usage_total": 6761,"net_usage": 279,"result": "SUCCESS"},` +
		`"internal_transactions": [{"hash": "1227752b70cbd45384e9276bddf4ed3e165c83f938a4282d734bf0458f8aee8d",` +
		`"caller_address": "41a4de5410785ffffc7c210f126f7534d868c2ecc9",` +
		`"transferTo_address": "41dcf2957055e8fcde477a077ecfeddf5ab2357547",` +
		`"callValueInfo": [{"callValue": 1}],"note": "63616c6c"}]}`

	liveReceiptWithMap = `{"id": "db8667ffea27109131fd5c113abcadd535c896590814a50dbf9eda253a8d19da","blockNumber": 165,` +
		`"blockTimeStamp": 1790865510000,"contractResult": [""],"receipt": {"net_usage": 250},` +
		`"cancel_unfreezeV2_amount": [{"key": "ENERGY","value": 10000000},{"key": "TRON_POWER","value": 0},` +
		`{"key": "BANDWIDTH","value": 0}]}`

	liveBlockTxID  = "db8667ffea27109131fd5c113abcadd535c896590814a50dbf9eda253a8d19da"
	liveBlockRawTx = "0a0200a42208b0ef6ad98a02403e40b8e1e8d38f345a57083b12530a38747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e43616e63656c416c6c556e667265657a655632436f6e747261637412170a1541dcf2957055e8fcde477a077ecfeddf5ab235754770a1899cbf8f34"
	liveBlock      = `{"blockID":"00000000000000a568a725352ccd7fbad4e65f56ed11d420a505c21f2230145d","block_header":{"raw_data":` +
		`{"timestamp":1790865510000,"txTrieRoot":"c85f5050cb4a05c58222017f823191d5ae1ca07b7bd7f090f493a35c10f03491",` +
		`"parentHash":"00000000000000a4b0ef6ad98a02403ebb86833716c2350b6338a6ee79df2dfc","number":165,` +
		`"witness_address":"417e5f4552091a69125d5dfcb7b8c2659029395bdf","version":38},"witness_signature":` +
		`"9a36b7a129a3a673d326e9cb5c6974a332cca68b4c1a5ab531327da89849239147d032ebfead62d75ceadb9d8482eba53507392658bde4b8aea902ca543101cf00"},` +
		`"transactions":[{"raw_data":{"ref_block_bytes":"00a4","ref_block_hash":"b0ef6ad98a02403e","expiration":1790908707000,` +
		`"contract":[{"parameter":{"value":{"owner_address":"41dcf2957055e8fcde477a077ecfeddf5ab2357547"},` +
		`"type_url":"type.googleapis.com/protocol.CancelAllUnfreezeV2Contract"},"type":"CancelAllUnfreezeV2Contract"}],` +
		`"timestamp":1790865507489},"signature":["755e1552868a1734c90c28dacb66e58f85eaa5fc2845593280e3b6ee1cf1b7af793cc378d8c3ad144542502270ab852095665abfffdea306200a9942a67e161401"],` +
		`"ret":[{"contractRet":"SUCCESS"}],"raw_data_hex":"` + liveBlockRawTx + `","txID":"` + liveBlockTxID + `"}]}`
)

func mustHexBytes(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	require.NoError(t, err)
	return b
}

// The receipt id is what Client.GetTransactionInfoByHash matches against the
// hash it asked for. Read as base64, a 64-digit hex id became 48 bytes of
// noise, so over HTTP every receipt - of a transaction long since in a block -
// came back as ErrTransactionInfoNotFound.
func TestHTTPGetTransactionInfoByHashFindsTheReceipt(t *testing.T) {
	tr, lastReq := newStubTransportAtPath(t, "/wallet/gettransactioninfobyid", http.StatusOK, liveReceiptWithLog)
	c := &Client{transport: tr}

	const txid = "0ca17a0da65cb2df52be4e5edd8654358f7ad2b6f083180e719784ff8ff1c8ae"
	info, err := c.GetTransactionInfoByHash(t.Context(), txid)
	require.NoError(t, err)
	require.Equal(t, txid, (*lastReq)["value"])

	require.Equal(t, mustHexBytes(t, txid), info.GetId())
	require.Equal(t, int64(156), info.GetBlockNumber())
	require.Equal(t, mustHexBytes(t, "41d53adb8bb7b45de60d0dc3d03ba0c187093b919d"), info.GetContractAddress())
	require.Equal(t, int64(29_650), info.GetReceipt().GetEnergyUsageTotal())
	require.Equal(t, core.Transaction_Result_SUCCESS, info.GetReceipt().GetResult())
	require.Equal(t, [][]byte{make([]byte, 32)}, info.GetContractResult())

	require.Len(t, info.GetLog(), 1)
	log := info.GetLog()[0]
	require.Len(t, log.GetAddress(), 20, "a log address has no 0x41 prefix")
	require.Len(t, log.GetTopics(), 3)
	require.Equal(t, mustHexBytes(t, "ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"), log.GetTopics()[0])
	require.Equal(t, int64(1_000_000), new(big256).setBytes(log.GetData()))
}

// big256 reads a 32-byte big-endian word that fits an int64.
type big256 struct{}

func (big256) setBytes(b []byte) int64 {
	var n int64
	for _, x := range b {
		n = n<<8 | int64(x)
	}
	return n
}

func TestHTTPGetTransactionInfoKeepsInternalTransactions(t *testing.T) {
	tr, _ := newStubTransport(t, http.StatusOK, liveReceiptWithInternalTx)

	info, err := tr.GetTransactionInfoById(t.Context(), mustHexBytes(t, "7c44d197b2dcdc7452663f1d7fab7636e06aeb29393f7cd32074a39dde9d4139"))
	require.NoError(t, err)

	require.Equal(t, [][]byte{{}}, info.GetContractResult(), "an empty result is still one result")
	require.Len(t, info.GetInternalTransactions(), 1)
	internal := info.GetInternalTransactions()[0]
	require.Len(t, internal.GetHash(), 32)
	require.Equal(t, mustHexBytes(t, "41a4de5410785ffffc7c210f126f7534d868c2ecc9"), internal.GetCallerAddress())
	require.Equal(t, mustHexBytes(t, "41dcf2957055e8fcde477a077ecfeddf5ab2357547"), internal.GetTransferToAddress())
	require.Equal(t, int64(1), internal.GetCallValueInfo()[0].GetCallValue())
	require.Equal(t, "call", string(internal.GetNote()))
}

// A map is an array of key/value objects in java-tron's JSON. protojson
// refuses that shape, so the receipt of every CancelAllUnfreezeV2 failed to
// decode at all.
func TestHTTPGetTransactionInfoReadsMaps(t *testing.T) {
	tr, _ := newStubTransport(t, http.StatusOK, liveReceiptWithMap)

	info, err := tr.GetTransactionInfoById(t.Context(), mustHexBytes(t, liveBlockTxID))
	require.NoError(t, err)
	require.Equal(t, map[string]int64{"ENERGY": 10_000_000, "TRON_POWER": 0, "BANDWIDTH": 0}, info.GetCancelUnfreezeV2Amount())
}

// "Not found" is an empty object, and must read as the empty message gRPC
// returns, which the client turns into ErrTransactionInfoNotFound.
func TestHTTPGetTransactionInfoByHashNotFound(t *testing.T) {
	tr, _ := newStubTransport(t, http.StatusOK, `{}`)
	c := &Client{transport: tr}

	_, err := c.GetTransactionInfoByHash(t.Context(), liveBlockTxID)
	require.ErrorIs(t, err, ErrTransactionInfoNotFound)
}

// The transaction comes from raw_data_hex - the bytes that were hashed and
// signed - not from the JSON rendering of raw_data, which the old field-name
// rewrite got wrong (ref_block_hash and the Any parameter were never decoded).
func TestHTTPBlockKeepsTransactionsExact(t *testing.T) {
	tr, _ := newStubTransportAtPath(t, "/wallet/getblockbynum", http.StatusOK, liveBlock)

	block, err := tr.GetBlockByNum(t.Context(), 165)
	require.NoError(t, err)

	require.Equal(t, mustHexBytes(t, "00000000000000a568a725352ccd7fbad4e65f56ed11d420a505c21f2230145d"), block.GetBlockid())
	header := block.GetBlockHeader().GetRawData()
	require.Equal(t, int64(165), header.GetNumber())
	require.Equal(t, mustHexBytes(t, "417e5f4552091a69125d5dfcb7b8c2659029395bdf"), header.GetWitnessAddress())
	require.Len(t, block.GetBlockHeader().GetWitnessSignature(), 65)

	require.Len(t, block.GetTransactions(), 1)
	ext := block.GetTransactions()[0]
	require.Equal(t, mustHexBytes(t, liveBlockTxID), ext.GetTxid())
	// java-tron marks every transaction of a block this way over gRPC.
	require.True(t, ext.GetResult().GetResult())

	raw, err := proto.Marshal(ext.GetTransaction().GetRawData())
	require.NoError(t, err)
	require.Equal(t, liveBlockRawTx, hex.EncodeToString(raw))
	sum := sha256.Sum256(raw)
	require.Equal(t, ext.GetTxid(), sum[:])

	cancel := &core.CancelAllUnfreezeV2Contract{}
	require.NoError(t, ext.GetTransaction().GetRawData().GetContract()[0].GetParameter().UnmarshalTo(cancel))
	require.Equal(t, mustHexBytes(t, "41dcf2957055e8fcde477a077ecfeddf5ab2357547"), cancel.GetOwnerAddress())
	require.Len(t, ext.GetTransaction().GetSignature()[0], 65)
	require.Equal(t, core.Transaction_Result_SUCCESS, ext.GetTransaction().GetRet()[0].GetContractRet())
}

// A raw_data_hex that disagrees with raw_data wins: it is what the txid is
// the hash of.
func TestDecodeTronTransactionPrefersRawDataHex(t *testing.T) {
	body := `{"raw_data":{"timestamp":1,"ref_block_hash":"ffff"},"raw_data_hex":"` + liveBlockRawTx + `"}`

	tx := &core.Transaction{}
	require.NoError(t, decodeTronJSON([]byte(body), tx))

	raw, err := proto.Marshal(tx.GetRawData())
	require.NoError(t, err)
	require.Equal(t, liveBlockRawTx, hex.EncodeToString(raw))
}

func TestHTTPBlockListAndEmptyRange(t *testing.T) {
	tr := newRoutedTransport(t, map[string]string{
		"/wallet/getblockbylimitnext": `{"block":[` + liveBlock + `,` + liveBlock + `]}`,
		"/wallet/getblockbylatestnum": `{}`,
	})

	list, err := tr.GetBlockByLimitNext(t.Context(), 165, 167)
	require.NoError(t, err)
	require.Len(t, list.GetBlock(), 2)
	require.Equal(t, mustHexBytes(t, liveBlockTxID), list.GetBlock()[1].GetTransactions()[0].GetTxid())

	empty, err := tr.GetBlockByLatestNum(t.Context(), 1)
	require.NoError(t, err)
	require.Empty(t, empty.GetBlock())
}

func TestHTTPTransactionInfoByBlockNumUnwrapsTheArray(t *testing.T) {
	tr := newRoutedTransport(t, map[string]string{
		"/wallet/gettransactioninfobyblocknum": `[` + liveReceiptWithLog + `,` + liveReceiptWithMap + `]`,
	})

	list, err := tr.GetTransactionInfoByBlockNum(t.Context(), 156)
	require.NoError(t, err)
	require.Len(t, list.GetTransactionInfo(), 2)
	require.Len(t, list.GetTransactionInfo()[0].GetId(), 32)
	require.Equal(t, int64(10_000_000), list.GetTransactionInfo()[1].GetCancelUnfreezeV2Amount()["ENERGY"])

	tr = newRoutedTransport(t, map[string]string{"/wallet/gettransactioninfobyblocknum": `{}`})
	list, err = tr.GetTransactionInfoByBlockNum(t.Context(), 1)
	require.NoError(t, err)
	require.Empty(t, list.GetTransactionInfo())

	tr = newRoutedTransport(t, map[string]string{"/wallet/gettransactioninfobyblocknum": `{"id":"00"}`})
	_, err = tr.GetTransactionInfoByBlockNum(t.Context(), 1)
	require.ErrorContains(t, err, "expected a JSON array")
}

// A balance above 2^53 SUN is not exact as a float64, which is what a
// generic JSON decode turns every number into.
func TestDecodeTronKeepsLargeIntegersExact(t *testing.T) {
	acc := &core.Account{}
	require.NoError(t, decodeTronJSON([]byte(`{"balance": 98999198714315521}`), acc))
	require.Equal(t, int64(98_999_198_714_315_521), acc.GetBalance())
}

// /wallet/getaccount re-prints asset_issued_ID as text after the hex pass
// (java-tron's Util.convertOutput); read as hex, an 8-digit id would decode
// into four other bytes.
func TestDecodeTronReadsAssetIssuedIDAsText(t *testing.T) {
	acc := &core.Account{}
	require.NoError(t, decodeTronJSON([]byte(`{"asset_issued_ID":"10000010","asset_issued_name":"4c41"}`), acc))
	require.Equal(t, "10000010", string(acc.GetAssetIssued_ID()))
	require.Equal(t, "LA", string(acc.GetAssetIssuedName()))
}

func TestDecodeTronReturnMessage(t *testing.T) {
	for name, tc := range map[string]struct{ body, want string }{
		"hex":        {`{"result":{"message":"5245564552"}}`, "REVER"},
		"plain text": {`{"result":{"message":"REVERT opcode executed"}}`, "REVERT opcode executed"},
	} {
		t.Run(name, func(t *testing.T) {
			ext := &api.TransactionExtention{}
			require.NoError(t, decodeTronJSON([]byte(tc.body), ext))
			require.Equal(t, tc.want, string(ext.GetResult().GetMessage()))
		})
	}
}

// The constant-call servlets drop the extention's txid and keep only the
// transaction's own.
func TestDecodeTronTransactionExtentionTakesTheTransactionTxID(t *testing.T) {
	ext := &api.TransactionExtention{}
	require.NoError(t, decodeTronJSON([]byte(`{"transaction":{"txID":"`+liveBlockTxID+`","raw_data_hex":"`+liveBlockRawTx+`"}}`), ext))
	require.Equal(t, mustHexBytes(t, liveBlockTxID), ext.GetTxid())

	ext = &api.TransactionExtention{}
	require.NoError(t, decodeTronJSON([]byte(`{"txid":"aa","transaction":{"txID":"`+liveBlockTxID+`"}}`), ext))
	require.Equal(t, []byte{0xaa}, ext.GetTxid(), "an explicit txid is kept")
}

func TestDecodeTronAny(t *testing.T) {
	t.Run("value as JSON", func(t *testing.T) {
		tx := &core.Transaction{}
		require.NoError(t, decodeTronJSON([]byte(`{"raw_data":{"contract":[{"type":"TransferContract","parameter":`+
			`{"type_url":"type.googleapis.com/protocol.TransferContract","value":`+
			`{"owner_address":"41dcf2957055e8fcde477a077ecfeddf5ab2357547","amount":5}}}]}}`), tx))

		transfer := &core.TransferContract{}
		require.NoError(t, tx.GetRawData().GetContract()[0].GetParameter().UnmarshalTo(transfer))
		require.Equal(t, int64(5), transfer.GetAmount())
		require.Len(t, transfer.GetOwnerAddress(), 21)
	})

	t.Run("value as hex", func(t *testing.T) {
		inner, err := anypb.New(&core.TransferContract{Amount: 9})
		require.NoError(t, err)
		body, err := json.Marshal(map[string]any{"type_url": inner.GetTypeUrl(), "value": hex.EncodeToString(inner.GetValue())})
		require.NoError(t, err)

		got := &anypb.Any{}
		require.NoError(t, decodeTronJSON(body, got))
		require.True(t, proto.Equal(inner, got))
	})

	t.Run("unknown type", func(t *testing.T) {
		got := &anypb.Any{}
		err := decodeTronJSON([]byte(`{"type_url":"type.googleapis.com/protocol.Nope","value":{}}`), got)
		require.ErrorContains(t, err, "protocol.Nope")
	})
}

func TestDecodeTronEnums(t *testing.T) {
	for name, tc := range map[string]struct {
		body string
		want core.ResourceCode
	}{
		"name":           {`{"type":"ENERGY"}`, core.ResourceCode_ENERGY},
		"number":         {`{"type":1}`, core.ResourceCode_ENERGY},
		"newer by digit": {`{"type":"7"}`, core.ResourceCode(7)},
	} {
		t.Run(name, func(t *testing.T) {
			f := &core.Account_FreezeV2{}
			require.NoError(t, decodeTronJSON([]byte(tc.body), f))
			require.Equal(t, tc.want, f.GetType())
		})
	}

	// An unknown name must not fall back to the zero value: for a permission
	// that is Owner, which authorizes everything.
	err := decodeTronJSON([]byte(`{"type":"Delegated"}`), &core.Permission{})
	require.ErrorContains(t, err, `unknown protocol.Permission.PermissionType value "Delegated"`)
}

func TestDecodeTronMaps(t *testing.T) {
	t.Run("object form", func(t *testing.T) {
		res := &api.AccountResourceMessage{}
		require.NoError(t, decodeTronJSON([]byte(`{"assetNetUsed":{"1000001":4}}`), res))
		require.Equal(t, map[string]int64{"1000001": 4}, res.GetAssetNetUsed())
	})
	t.Run("zero value left out", func(t *testing.T) {
		res := &api.AccountResourceMessage{}
		require.NoError(t, decodeTronJSON([]byte(`{"assetNetUsed":[{"key":"1000001"}]}`), res))
		require.Equal(t, map[string]int64{"1000001": 0}, res.GetAssetNetUsed())
	})
	t.Run("integer keys", func(t *testing.T) {
		p := &core.Proposal{}
		require.NoError(t, decodeTronJSON([]byte(`{"parameters":[{"key":77,"value":1}]}`), p))
		require.Equal(t, map[int64]int64{77: 1}, p.GetParameters())
	})
	t.Run("not a map", func(t *testing.T) {
		err := decodeTronJSON([]byte(`{"assetNetUsed":7}`), &api.AccountResourceMessage{})
		require.ErrorContains(t, err, "key/value")
	})
}

func TestDecodeTronRejectsMalformedValues(t *testing.T) {
	for name, tc := range map[string]struct {
		body string
		msg  proto.Message
		want string
	}{
		"not json":         {`{`, &core.Account{}, "parse json"},
		"not an object":    {`[1]`, &core.Account{}, "expected a JSON object"},
		"bytes not hex":    {`{"address":"TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g"}`, &core.Account{}, "Account.address"},
		"int as string":    {`{"balance":"x"}`, &core.Account{}, "Account.balance"},
		"int overflow":     {`{"balance":99999999999999999999}`, &core.Account{}, "Account.balance"},
		"bool as number":   {`{"is_witness":1}`, &core.Account{}, "Account.is_witness"},
		"list not array":   {`{"votes":{}}`, &core.Account{}, "expected a JSON array"},
		"list element":     {`{"votes":[{"vote_count":"x"}]}`, &core.Account{}, "[0]"},
		"string as number": {`{"permission_name":1}`, &core.Permission{}, "Permission.permission_name"},
		"raw_data_hex":     {`{"raw_data_hex":"zz"}`, &core.Transaction{}, "raw_data_hex"},
		"raw_data proto":   {`{"raw_data_hex":"ff"}`, &core.Transaction{}, "unmarshal raw_data_hex"},
	} {
		t.Run(name, func(t *testing.T) {
			require.ErrorContains(t, decodeTronJSON([]byte(tc.body), tc.msg), tc.want)
		})
	}
}

// Unknown keys are what the node adds next to the proto fields, and null is
// how it says nothing; neither is an error.
func TestDecodeTronSkipsUnknownKeysAndNull(t *testing.T) {
	acc := &core.Account{}
	require.NoError(t, decodeTronJSON([]byte(`{"visible":false,"balance":3,"account_resource":null}`), acc))
	require.Equal(t, int64(3), acc.GetBalance())
	require.Nil(t, acc.GetAccountResource())
}

// DeployContractServlet wraps "abi" in {"entrys": …} itself. Sending the
// whole ABI message nested it twice, and the contract was stored with one
// empty entry - deployed, but uncallable from anything that reads its ABI.
func TestHTTPDeployContractSendsTheEntriesArray(t *testing.T) {
	tr, lastReq := newStubTransportAtPath(t, "/wallet/deploycontract", http.StatusOK, liveFreezeResponse)

	owner := mustDecode(t, testAddr)
	contractABI := &core.SmartContract_ABI{Entrys: []*core.SmartContract_ABI_Entry{
		{
			Name: "transfer", Type: core.SmartContract_ABI_Entry_Function,
			StateMutability: core.SmartContract_ABI_Entry_Nonpayable,
			Inputs: []*core.SmartContract_ABI_Entry_Param{
				{Name: "to", Type: "address"}, {Name: "value", Type: "uint256"},
			},
			Outputs: []*core.SmartContract_ABI_Entry_Param{{Type: "bool"}},
		},
		{Name: "Transfer", Type: core.SmartContract_ABI_Entry_Event, Inputs: []*core.SmartContract_ABI_Entry_Param{
			{Indexed: true, Name: "from", Type: "address"},
		}},
	}}

	_, err := tr.DeployContract(t.Context(), &core.CreateSmartContract{
		OwnerAddress:   owner,
		CallTokenValue: 4,
		TokenId:        1_000_001,
		NewContract: &core.SmartContract{
			Abi: contractABI, Bytecode: []byte{0x60, 0x80}, Name: "T", CallValue: 3,
			ConsumeUserResourcePercent: 30, OriginEnergyLimit: 9,
		},
	})
	require.NoError(t, err)

	req := *lastReq
	entries, ok := req["abi"].([]any)
	require.True(t, ok, "abi must be the array of entries, got %T", req["abi"])
	require.Len(t, entries, 2)

	first := entries[0].(map[string]any)
	require.Equal(t, "transfer", first["name"])
	require.Equal(t, "Function", first["type"])
	require.Equal(t, "Nonpayable", first["stateMutability"])
	require.Len(t, first["inputs"], 2)
	require.Equal(t, true, entries[1].(map[string]any)["inputs"].([]any)[0].(map[string]any)["indexed"])

	require.Equal(t, "6080", req["bytecode"])
	require.EqualValues(t, 3, req["call_value"])
	require.EqualValues(t, 4, req["call_token_value"])
	require.EqualValues(t, 1_000_001, req["token_id"])
	require.EqualValues(t, 30, req["consume_user_resource_percent"])
	require.EqualValues(t, 9, req["origin_energy_limit"])
}

func TestHTTPDeployContractWithoutABI(t *testing.T) {
	tr, lastReq := newStubTransport(t, http.StatusOK, liveFreezeResponse)

	_, err := tr.DeployContract(t.Context(), &core.CreateSmartContract{
		OwnerAddress: mustDecode(t, testAddr),
		NewContract:  &core.SmartContract{Bytecode: []byte{1}, OriginEnergyLimit: 1},
	})
	require.NoError(t, err)
	require.NotContains(t, *lastReq, "abi")
	require.NotContains(t, *lastReq, "call_value")
	require.NotContains(t, *lastReq, "token_id")
}

// The constant-call requests carry the TRC10 fields gRPC carries; without
// them a call that sends a token was priced as one that does not.
func TestHTTPConstantCallSendsTokenFields(t *testing.T) {
	tr, lastReq := newStubTransport(t, http.StatusOK, `{"result":{"result":true}}`)

	_, err := tr.EstimateEnergy(t.Context(), &core.TriggerSmartContract{
		OwnerAddress: mustDecode(t, testAddr), ContractAddress: mustDecode(t, testAddr),
		CallValue: 2, CallTokenValue: 5, TokenId: 1_000_001,
	})
	require.NoError(t, err)
	require.EqualValues(t, 2, (*lastReq)["call_value"])
	require.EqualValues(t, 5, (*lastReq)["call_token_value"])
	require.EqualValues(t, 1_000_001, (*lastReq)["token_id"])
	require.NotContains(t, *lastReq, "visible")
}

func TestDecodeTronNodeInfoScalars(t *testing.T) {
	info := &core.NodeInfo{}
	require.NoError(t, decodeTronJSON([]byte(`{"machineInfo":{"cpuRate":0.25,"cpuCount":8},`+
		`"cheatWitnessInfoMap":[{"key":"a","value":"b"},{"key":"c"}],`+
		`"configNodeInfo":{"discoverEnable":"true","minTimeRatio":1e-1}}`), info))

	require.InDelta(t, 0.25, info.GetMachineInfo().GetCpuRate(), 0)
	require.Equal(t, int32(8), info.GetMachineInfo().GetCpuCount())
	require.Equal(t, map[string]string{"a": "b", "c": ""}, info.GetCheatWitnessInfoMap())
	require.True(t, info.GetConfigNodeInfo().GetDiscoverEnable())
	require.InDelta(t, 0.1, info.GetConfigNodeInfo().GetMinTimeRatio(), 1e-12)

	for name, body := range map[string]string{
		"double as string":    `{"machineInfo":{"cpuRate":"fast"}}`,
		"bool as text":        `{"configNodeInfo":{"discoverEnable":"yes"}}`,
		"int32 overflow":      `{"machineInfo":{"cpuCount":4294967296}}`,
		"enum overflow":       `{"type":4294967296}`,
		"map entry not obj":   `{"cheatWitnessInfoMap":[1]}`,
		"map value mismatch":  `{"cheatWitnessInfoMap":[{"key":"a","value":1}]}`,
		"map key mismatch":    `{"cheatWitnessInfoMap":{"a":1}}`,
		"map int key invalid": `{"parameters":[{"key":"x","value":1}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			var msg proto.Message = &core.NodeInfo{}
			switch name {
			case "enum overflow":
				msg = &core.Account_FreezeV2{}
			case "map int key invalid":
				msg = &core.Proposal{}
			}
			require.Error(t, decodeTronJSON([]byte(body), msg))
		})
	}
}
