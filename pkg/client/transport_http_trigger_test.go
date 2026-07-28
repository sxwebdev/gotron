package client

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/tronutils"
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
