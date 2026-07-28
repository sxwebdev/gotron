package client

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
	"google.golang.org/protobuf/proto"
)

const (
	triggerOwnerAddr    = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
	triggerContractAddr = "TXLAQ63Xg1NAzckPwKHvzw7CSEmLMEqcdj"
	triggerCallData     = "a9059cbb000000000000000000000000d80b8e8b1f6d3b6c5f5e5b0d5e5c5a595857565500000000000000000000000000000000000000000000000000000000000f4240"
)

// Node-shaped /wallet/triggersmartcontract response: the transaction is nested under
// "transaction", the contract parameter is a Tron-style Any ({"value":..,"type_url":..})
// and every bytes field inside raw_data is hex, not base64.
const liveTriggerResponse = `{"result":{"result":true},"transaction":{"visible":true,
	"txID":"ba9266e0f269bd3cc0b769e8ed1fa3d545790115bc6eac849015b041b0076165",
	"raw_data":{"ref_block_bytes":"cc02","ref_block_hash":"c1171222b68f923e","expiration":1785238218000,
		"contract":[{"parameter":{"value":{"data":"` + triggerCallData + `","owner_address":"` + triggerOwnerAddr + `","contract_address":"` + triggerContractAddr + `"},"type_url":"type.googleapis.com/protocol.TriggerSmartContract"},"type":"TriggerSmartContract"}],
		"timestamp":1785238160928},
	"raw_data_hex":"0a02cc022208c1171222b68f923e4090caf5c3fa335aae01081f12a9010a31747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e54726967676572536d617274436f6e747261637412740a1541a614f803b6fd780986a42c78ec9c7f77e6ded13c121541ea51342dabbb928ae1e576bd39eff8aaf070a8c62244a9059cbb000000000000000000000000d80b8e8b1f6d3b6c5f5e5b0d5e5c5a595857565500000000000000000000000000000000000000000000000000000000000f424070a08cf2c3fa33"}}`

func TestHTTPTriggerContractParsesWrappedTransaction(t *testing.T) {
	tr, lastReq := newStubTransport(t, http.StatusOK, liveTriggerResponse)

	owner, err := tronutils.DecodeCheck(triggerOwnerAddr)
	require.NoError(t, err)
	contractAddr, err := tronutils.DecodeCheck(triggerContractAddr)
	require.NoError(t, err)
	callData, err := hex.DecodeString(triggerCallData)
	require.NoError(t, err)

	tx, err := tr.TriggerContract(t.Context(), &core.TriggerSmartContract{
		OwnerAddress:    owner,
		ContractAddress: contractAddr,
		Data:            callData,
	})
	require.NoError(t, err)

	rawData := tx.GetTransaction().GetRawData()
	require.NotNil(t, rawData)

	// Bytes fields inside raw_data are hex in Tron's JSON; decoding them as base64
	// silently corrupts the TAPOS reference.
	require.Equal(t, "cc02", hex.EncodeToString(rawData.GetRefBlockBytes()))
	require.Equal(t, "c1171222b68f923e", hex.EncodeToString(rawData.GetRefBlockHash()))

	// The contract payload must survive: recipient and amount live in Data.
	require.Len(t, rawData.GetContract(), 1)
	require.Equal(t, core.Transaction_Contract_TriggerSmartContract, rawData.GetContract()[0].GetType())

	var got core.TriggerSmartContract
	require.NoError(t, rawData.GetContract()[0].GetParameter().UnmarshalTo(&got))
	require.Equal(t, owner, got.GetOwnerAddress())
	require.Equal(t, contractAddr, got.GetContractAddress())
	require.Equal(t, callData, got.GetData())

	// Re-serializing raw_data must reproduce the node's txID, i.e. the transaction is
	// byte-identical to what the node built and is safe to sign.
	rawBytes, err := proto.Marshal(rawData)
	require.NoError(t, err)
	sum := sha256.Sum256(rawBytes)
	require.Equal(t, hex.EncodeToString(sum[:]), hex.EncodeToString(tx.GetTxid()))

	// Request shape.
	require.Equal(t, triggerOwnerAddr, (*lastReq)["owner_address"])
	require.Equal(t, triggerContractAddr, (*lastReq)["contract_address"])
	require.Equal(t, triggerCallData, (*lastReq)["data"])
	require.Equal(t, true, (*lastReq)["visible"])
}

func TestHTTPTriggerContractSurfacesNodeError(t *testing.T) {
	// Tron reports contract validation failures with HTTP 200 and a hex-encoded message.
	const body = `{"result":{"code":"CONTRACT_VALIDATE_ERROR","message":"6163636f756e74206465732076616c6964617465206572726f72"}}`

	tr, _ := newStubTransport(t, http.StatusOK, body)

	owner, err := tronutils.DecodeCheck(triggerOwnerAddr)
	require.NoError(t, err)

	_, err = tr.TriggerContract(t.Context(), &core.TriggerSmartContract{OwnerAddress: owner, ContractAddress: owner})
	require.ErrorIs(t, err, ErrInvalidTransaction)
	require.ErrorContains(t, err, "CONTRACT_VALIDATE_ERROR")
	require.ErrorContains(t, err, "account des validate error")

	// Typed, so a caller can tell a refused request from an unwell node without
	// reading the message.
	var cve *ContractValidateError
	require.ErrorAs(t, err, &cve)
}

// publicnode returns the same failure with a plain-text message rather than hex, so
// the hex fallback must not mangle it.
func TestHTTPTriggerContractSurfacesPlainTextNodeError(t *testing.T) {
	// Verbatim response from tron-rpc.publicnode.com.
	const body = `{"result":{"code":"CONTRACT_VALIDATE_ERROR","message":"No contract or not a valid smart contract"}}`

	tr, _ := newStubTransport(t, http.StatusOK, body)

	owner, err := tronutils.DecodeCheck(triggerOwnerAddr)
	require.NoError(t, err)

	_, err = tr.TriggerContract(t.Context(), &core.TriggerSmartContract{OwnerAddress: owner, ContractAddress: owner})
	require.ErrorIs(t, err, ErrInvalidTransaction)
	require.ErrorContains(t, err, "No contract or not a valid smart contract")
}

// A reverted constant call, verbatim from tron-rpc.publicnode.com. The node
// reports result=true with code absent (i.e. SUCCESS) and puts the failure only
// in message, so dropping code and message - which this transport used to do -
// leaves a revert indistinguishable from a successful call over HTTP while gRPC
// reports it. Every caller that trusts the result then prices the energy burned
// before the revert as the cost of the whole call.
const liveConstantRevertResponse = `{"result":{"result":true,"message":"REVERT opcode executed"},
	"constant_result":[""],"energy_used":8624,"energy_penalty":5859,
	"transaction":{"ret":[{"ret":"FAILED"}],"visible":true,
		"txID":"0c32fa4b4ea21f8eb5d65daf01c62ac1c252a509a20875788f02d4c93211e1a8"}}`

func TestHTTPTriggerConstantContractCarriesTheFailure(t *testing.T) {
	tr, _ := newStubTransport(t, http.StatusOK, liveConstantRevertResponse)

	owner, err := tronutils.DecodeCheck(triggerOwnerAddr)
	require.NoError(t, err)

	tx, err := tr.TriggerConstantContract(t.Context(), &core.TriggerSmartContract{
		OwnerAddress:    owner,
		ContractAddress: owner,
	})
	// The transport itself does not judge the call; it must simply not lose the
	// evidence.
	require.NoError(t, err)

	require.Equal(t, "REVERT opcode executed", string(tx.GetResult().GetMessage()))
	require.True(t, tx.GetResult().GetResult(), "a revert still answers result=true")
	require.Equal(t, int64(8624), tx.GetEnergyUsed())
	require.Equal(t, int64(5859), tx.GetEnergyPenalty())
}

func TestHTTPTriggerConstantContractMapsTheResultCode(t *testing.T) {
	const body = `{"result":{"result":false,"code":"CONTRACT_VALIDATE_ERROR","message":"boom"},"constant_result":[]}`

	tr, _ := newStubTransport(t, http.StatusOK, body)

	owner, err := tronutils.DecodeCheck(triggerOwnerAddr)
	require.NoError(t, err)

	tx, err := tr.TriggerConstantContract(t.Context(), &core.TriggerSmartContract{
		OwnerAddress:    owner,
		ContractAddress: owner,
	})
	require.NoError(t, err)

	// The code arrives as a name, and the enum it maps to is what callers
	// compare against - leaving it zero reads as SUCCESS.
	require.Equal(t, api.Return_CONTRACT_VALIDATE_ERROR, tx.GetResult().GetCode())
	require.Equal(t, "boom", string(tx.GetResult().GetMessage()))
}

func TestHTTPTriggerConstantContractSuccessStaysClean(t *testing.T) {
	const body = `{"result":{"result":true},
		"constant_result":["0000000000000000000000000000000000000000000000000000000000000001"],
		"energy_used":64285}`

	tr, _ := newStubTransport(t, http.StatusOK, body)

	owner, err := tronutils.DecodeCheck(triggerOwnerAddr)
	require.NoError(t, err)

	tx, err := tr.TriggerConstantContract(t.Context(), &core.TriggerSmartContract{
		OwnerAddress:    owner,
		ContractAddress: owner,
	})
	require.NoError(t, err)

	// An absent code must stay SUCCESS and an absent message must stay empty,
	// or the client layer would report every successful read as a failure.
	require.Equal(t, api.Return_SUCCESS, tx.GetResult().GetCode())
	require.Empty(t, tx.GetResult().GetMessage())
	require.Len(t, tx.GetConstantResult(), 1)
}

// Verbatim /wallet/getcontract response (addresses hex, as the node sends them
// when visible is not set). Every bytes field here has to survive: origin_address
// decides who pays a call's energy.
const liveGetContractResponse = `{
	"origin_address":"414698ca96dd198ae04e6c45b199516c17c31dbc95",
	"contract_address":"41ea51342dabbb928ae1e576bd39eff8aaf070a8c6",
	"abi":{"entrys":[{"name":"transfer","stateMutability":"Nonpayable","type":"Function",
		"inputs":[{"name":"to","type":"address"},{"name":"value","type":"uint256"}],
		"outputs":[{"name":"","type":"bool"}]}]},
	"bytecode":"608060405260043610",
	"name":"TetherToken",
	"origin_energy_limit":1000000000,
	"code_hash":"a1b2c3d4",
	"consume_user_resource_percent":25
}`

func TestHTTPGetContractDecodesTheByteFields(t *testing.T) {
	tr, lastReq := newStubTransport(t, http.StatusOK, liveGetContractResponse)

	addr, err := tronutils.DecodeCheck(triggerContractAddr)
	require.NoError(t, err)

	got, err := tr.GetContract(t.Context(), addr)
	require.NoError(t, err)

	// Handing the node's hex straight to protojson base64-decodes it: a
	// 21-byte address came back as 31 bytes of nonsense, with no error, and
	// callers billed the wrong account for a contract's energy.
	wantOrigin, err := hex.DecodeString("414698ca96dd198ae04e6c45b199516c17c31dbc95")
	require.NoError(t, err)
	require.Equal(t, wantOrigin, got.GetOriginAddress())
	require.Len(t, got.GetOriginAddress(), 21)

	wantContract, err := hex.DecodeString("41ea51342dabbb928ae1e576bd39eff8aaf070a8c6")
	require.NoError(t, err)
	require.Equal(t, wantContract, got.GetContractAddress())

	require.Equal(t, []byte{0x60, 0x80, 0x60, 0x40, 0x52, 0x60, 0x04, 0x36, 0x10}, got.GetBytecode())
	require.Equal(t, []byte{0xa1, 0xb2, 0xc3, 0xd4}, got.GetCodeHash())

	// The scalars that always worked must keep working.
	require.Equal(t, "TetherToken", got.GetName())
	require.Equal(t, int64(25), got.GetConsumeUserResourcePercent())
	require.Equal(t, int64(1_000_000_000), got.GetOriginEnergyLimit())

	// The ABI's strings must not be mistaken for bytes on the way through.
	require.Len(t, got.GetAbi().GetEntrys(), 1)
	require.Equal(t, "transfer", got.GetAbi().GetEntrys()[0].GetName())
	require.Equal(t, "address", got.GetAbi().GetEntrys()[0].GetInputs()[0].GetType())

	// visible must stay off: with it the node answers in base58, which the
	// same transform would then mangle as if it were hex.
	require.NotContains(t, *lastReq, "visible")
	require.Equal(t, hex.EncodeToString(addr), (*lastReq)["value"])
}

// A deployment is expressed as a TriggerSmartContract with no contract address.
// The field has to be left out of the JSON entirely: EncodeCheck of nothing is a
// short but well-formed base58 string that the node reads as a real address, and
// an explicit "" is refused outright ("invalid address for field ...
// contract_address"). Either way the node would price a call instead of a
// deployment, or fail.
func TestHTTPConstantCallOmitsAnEmptyContractAddress(t *testing.T) {
	const body = `{"result":{"result":true},"constant_result":[""],"energy_used":2021}`

	tr, lastReq := newStubTransport(t, http.StatusOK, body)

	owner, err := tronutils.DecodeCheck(triggerOwnerAddr)
	require.NoError(t, err)
	code, err := hex.DecodeString("600a600c600039600a6000f3602a60805260206080f3")
	require.NoError(t, err)

	tx, err := tr.TriggerConstantContract(t.Context(), &core.TriggerSmartContract{
		OwnerAddress: owner,
		Data:         code,
	})
	require.NoError(t, err)
	require.Equal(t, int64(2021), tx.GetEnergyUsed())

	require.NotContains(t, *lastReq, "contract_address")
	require.Equal(t, triggerOwnerAddr, (*lastReq)["owner_address"])
	require.Equal(t, hex.EncodeToString(code), (*lastReq)["data"])
}

func TestHTTPConstantCallKeepsARealContractAddress(t *testing.T) {
	const body = `{"result":{"result":true},"constant_result":[""],"energy_used":1}`

	tr, lastReq := newStubTransport(t, http.StatusOK, body)

	owner, err := tronutils.DecodeCheck(triggerOwnerAddr)
	require.NoError(t, err)
	contractAddr, err := tronutils.DecodeCheck(triggerContractAddr)
	require.NoError(t, err)

	_, err = tr.TriggerConstantContract(t.Context(), &core.TriggerSmartContract{
		OwnerAddress:    owner,
		ContractAddress: contractAddr,
		Data:            []byte{0x01},
	})
	require.NoError(t, err)
	require.Equal(t, triggerContractAddr, (*lastReq)["contract_address"])
}

func TestHTTPEstimateEnergyOmitsAnEmptyContractAddress(t *testing.T) {
	const body = `{"result":{"result":true},"energy_required":4042}`

	tr, lastReq := newStubTransport(t, http.StatusOK, body)

	owner, err := tronutils.DecodeCheck(triggerOwnerAddr)
	require.NoError(t, err)

	got, err := tr.EstimateEnergy(t.Context(), &core.TriggerSmartContract{
		OwnerAddress: owner,
		Data:         []byte{0x60},
	})
	require.NoError(t, err)
	require.Equal(t, int64(4042), got.GetEnergyRequired())
	require.NotContains(t, *lastReq, "contract_address")
}
