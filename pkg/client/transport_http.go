package client

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// HTTPTransport implements Transport using HTTP REST API
type HTTPTransport struct {
	baseURL    string
	httpClient *http.Client
	headers    map[string]string
}

// NewHTTPTransport creates a new HTTP transport
func NewHTTPTransport(cfg NodeConfig) (*HTTPTransport, error) {
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	baseURL := strings.TrimSuffix(cfg.Address, "/")

	return &HTTPTransport{
		baseURL:    baseURL,
		httpClient: httpClient,
		headers:    cfg.Headers,
	}, nil
}

// Close closes the HTTP transport (no-op for HTTP)
func (t *HTTPTransport) Close() error {
	return nil
}

// doRequestRaw performs an HTTP POST request and returns raw JSON response
func (t *HTTPTransport) doRequestRaw(ctx context.Context, endpoint string, body any) ([]byte, error) {
	var bodyReader io.Reader

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, t.wrapErr(endpoint, fmt.Errorf("marshal request body: %w", err))
		}
		bodyReader = bytes.NewReader(jsonBody)
	} else {
		bodyReader = bytes.NewReader([]byte("{}"))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.baseURL+endpoint, bodyReader)
	if err != nil {
		return nil, t.wrapErr(endpoint, fmt.Errorf("create request: %w", err))
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	for key, value := range t.headers {
		req.Header.Set(key, value)
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, t.wrapErr(endpoint, fmt.Errorf("http request: %w", err))
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, t.wrapErr(endpoint, fmt.Errorf("read response: %w", err))
	}

	if resp.StatusCode != http.StatusOK {
		return nil, t.wrapErr(endpoint, &HTTPStatusError{Code: resp.StatusCode, Body: string(respBody)})
	}

	return respBody, nil
}

func (t *HTTPTransport) wrapErr(method string, err error) error {
	return &TransportError{
		Host:     t.baseURL,
		Protocol: "http",
		Method:   method,
		Err:      err,
	}
}

// apiError reports a refusal the node returned as HTTP 200 with an "Error"
// field, which is how the /wallet endpoints answer a request they would not
// process at all - a malformed address, a value out of range.
//
// Without it such a refusal is indistinguishable from an empty answer: no
// message has the field, so the decoder skips it and leaves the zero message
// behind, and "this request was wrong" arrives as "there is nothing there".
//
// A body that is not a JSON object - an array, a bare number - carries no such
// field and is left to the caller's own decoding.
//
// The substring test is not just a fast path for the block bodies this runs on,
// where a full parse would be the third one of the same megabyte: encoding/json
// matches field names case-insensitively, so without it a gateway that added a
// lowercase "error" key of its own alongside a valid payload would fail every
// read.
func apiError(body []byte) error {
	if !bytes.Contains(body, []byte(`"Error"`)) {
		return nil
	}

	var probe struct {
		Error string `json:"Error"`
	}

	if err := json.Unmarshal(body, &probe); err != nil || probe.Error == "" {
		return nil
	}

	return fmt.Errorf("%w: %s", ErrNodeRefusedRequest, probe.Error)
}

// fetch is doRequestRaw plus that check, which every decoding path below needs.
//
// The transaction-creating endpoints are the exception and stay on
// doRequestRaw: they report the same refusal, but parseTxResponse gives it a
// type so callers can match it the way they match the gRPC equivalent.
func (t *HTTPTransport) fetch(ctx context.Context, endpoint string, body any) ([]byte, error) {
	respBody, err := t.doRequestRaw(ctx, endpoint, body)
	if err != nil {
		return nil, err
	}

	if err := apiError(respBody); err != nil {
		return nil, t.wrapErr(endpoint, err)
	}

	return respBody, nil
}

// fetchJSON decodes an answer with encoding/json, for the few endpoints whose
// answer is not a protobuf message at all (getReward, getBrokerage,
// broadcasthex).
func (t *HTTPTransport) fetchJSON(ctx context.Context, endpoint string, body, result any) error {
	respBody, err := t.fetch(ctx, endpoint, body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(respBody, result); err != nil {
		return t.wrapErr(endpoint, fmt.Errorf("unmarshal response: %w (body: %s)", err, string(respBody)))
	}

	return nil
}

// fetchTron performs a read and decodes the answer with decodeTronJSON. The
// request must not set "visible" - see decodeTronJSON.
func (t *HTTPTransport) fetchTron(ctx context.Context, endpoint string, body any, result proto.Message) error {
	respBody, err := t.fetch(ctx, endpoint, body)
	if err != nil {
		return err
	}

	if err := decodeTronJSON(respBody, result); err != nil {
		return t.wrapErr(endpoint, fmt.Errorf("unmarshal response: %w (body: %s)", err, truncateBody(respBody)))
	}

	return nil
}

// truncateBody keeps an error message readable when the body is a block.
func truncateBody(body []byte) string {
	const limit = 512
	if len(body) <= limit {
		return string(body)
	}
	return string(body[:limit]) + "…"
}

// doTxRequest performs an HTTP POST request to a transaction-creating endpoint.
//
// Such endpoints return the raw transaction at the top level instead of the
// TransactionExtention shape, and report contract validation failures as HTTP 200
// with an "Error" field. The transaction is rebuilt from raw_data_hex, which is the
// protobuf-serialized TransactionRaw - unlike raw_data, it needs no JSON translation
// and carries addresses in their canonical byte form regardless of "visible".
func (t *HTTPTransport) doTxRequest(ctx context.Context, endpoint string, body any) (*api.TransactionExtention, error) {
	respBody, err := t.doRequestRaw(ctx, endpoint, body)
	if err != nil {
		return nil, err
	}

	return t.parseTxResponse(endpoint, respBody)
}

// doTxRequestWrapped is doTxRequest for transaction-creating endpoints that nest the
// transaction under "transaction" and report the outcome in "result" instead of "Error"
// (e.g. /wallet/triggersmartcontract).
func (t *HTTPTransport) doTxRequestWrapped(ctx context.Context, endpoint string, body any) (*api.TransactionExtention, error) {
	respBody, err := t.doRequestRaw(ctx, endpoint, body)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Result struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"result"`
		Transaction json.RawMessage `json:"transaction"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, t.wrapErr(endpoint, fmt.Errorf("unmarshal response: %w (body: %s)", err, string(respBody)))
	}

	if resp.Result.Code != "" && resp.Result.Code != api.Return_SUCCESS.String() {
		// java-tron serializes Return.message (bytes) as hex, so decode it when possible
		// to surface the human-readable validation error.
		message := resp.Result.Message
		if decoded, err := hex.DecodeString(message); err == nil {
			message = string(decoded)
		}
		return nil, t.wrapErr(endpoint, &ContractValidateError{
			Code:    api.ReturnResponseCode(api.ReturnResponseCode_value[resp.Result.Code]),
			Message: message,
		})
	}

	if len(resp.Transaction) == 0 {
		return nil, t.wrapErr(endpoint, fmt.Errorf("%w: no transaction in response (body: %s)", ErrInvalidTransaction, string(respBody)))
	}

	return t.parseTxResponse(endpoint, resp.Transaction)
}

// parseTxResponse rebuilds a transaction from a Tron transaction JSON object.
func (t *HTTPTransport) parseTxResponse(endpoint string, respBody []byte) (*api.TransactionExtention, error) {
	var resp struct {
		Error      string `json:"Error"`
		TxID       string `json:"txID"`
		RawDataHex string `json:"raw_data_hex"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, t.wrapErr(endpoint, fmt.Errorf("unmarshal transaction: %w (body: %s)", err, string(respBody)))
	}

	// The node answers 200 with this field when it refuses to build the
	// transaction. It is the same class of failure gRPC reports through
	// Result.Code, so it gets the same type - otherwise which errors a caller
	// can match on would depend on the transport.
	if resp.Error != "" {
		return nil, t.wrapErr(endpoint, &ContractValidateError{Message: resp.Error})
	}

	if resp.RawDataHex == "" {
		return nil, t.wrapErr(endpoint, fmt.Errorf("%w: no raw_data_hex in response (body: %s)", ErrInvalidTransaction, string(respBody)))
	}

	rawBytes, err := hex.DecodeString(resp.RawDataHex)
	if err != nil {
		return nil, t.wrapErr(endpoint, fmt.Errorf("decode raw_data_hex: %w", err))
	}

	rawData := &core.TransactionRaw{}
	if err := proto.Unmarshal(rawBytes, rawData); err != nil {
		return nil, t.wrapErr(endpoint, fmt.Errorf("unmarshal raw_data_hex: %w", err))
	}

	txid, err := hex.DecodeString(resp.TxID)
	if err != nil {
		return nil, t.wrapErr(endpoint, fmt.Errorf("decode txID: %w", err))
	}

	return &api.TransactionExtention{
		Transaction: &core.Transaction{RawData: rawData},
		Txid:        txid,
		Result:      &api.Return{Result: true, Code: api.Return_SUCCESS},
	}, nil
}

// Account operations

// hexAddress renders an address the way a request without "visible" takes it.
func hexAddress(address []byte) string {
	return hex.EncodeToString(address)
}

// GetAccount reads the whole core.Account, every field gRPC returns. The
// hand-written struct this used to parse into had a subset of them, so votes,
// account_name, net_usage, the TRC10 an account issued and its bandwidth
// delegation totals all read as zero over HTTP.
func (t *HTTPTransport) GetAccount(ctx context.Context, account *core.Account) (*core.Account, error) {
	reqBody := map[string]any{
		"address": hexAddress(account.Address),
	}

	result := &core.Account{}
	if err := t.fetchTron(ctx, "/wallet/getaccount", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) GetAccountResource(ctx context.Context, account *core.Account) (*api.AccountResourceMessage, error) {
	reqBody := map[string]any{
		"address": hexAddress(account.Address),
	}

	result := &api.AccountResourceMessage{}
	if err := t.fetchTron(ctx, "/wallet/getaccountresource", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) CreateAccount(ctx context.Context, contract *core.AccountCreateContract) (*api.TransactionExtention, error) {
	reqBody := map[string]any{
		"owner_address":   tronutils.EncodeCheck(contract.OwnerAddress),
		"account_address": tronutils.EncodeCheck(contract.AccountAddress),
		"visible":         true,
	}

	return t.doTxRequest(ctx, "/wallet/createaccount", reqBody)
}

type httpPermissionKeyRequest struct {
	Address string `json:"address"`
	Weight  int64  `json:"weight"`
}

type httpPermissionRequest struct {
	Type           int32                      `json:"type"`
	ID             int32                      `json:"id,omitempty"`
	PermissionName string                     `json:"permission_name"`
	Threshold      int64                      `json:"threshold"`
	ParentID       int32                      `json:"parent_id,omitempty"`
	Operations     string                     `json:"operations,omitempty"`
	Keys           []httpPermissionKeyRequest `json:"keys"`
}

func permissionRequest(p *core.Permission) httpPermissionRequest {
	keys := make([]httpPermissionKeyRequest, 0, len(p.GetKeys()))
	for _, key := range p.GetKeys() {
		keys = append(keys, httpPermissionKeyRequest{
			Address: tronutils.EncodeCheck(key.GetAddress()),
			Weight:  key.GetWeight(),
		})
	}
	return httpPermissionRequest{
		Type:           int32(p.GetType()),
		ID:             p.GetId(),
		PermissionName: p.GetPermissionName(),
		Threshold:      p.GetThreshold(),
		ParentID:       p.GetParentId(),
		Operations:     hex.EncodeToString(p.GetOperations()),
		Keys:           keys,
	}
}

func (t *HTTPTransport) AccountPermissionUpdate(ctx context.Context, contract *core.AccountPermissionUpdateContract) (*api.TransactionExtention, error) {
	actives := make([]httpPermissionRequest, 0, len(contract.GetActives()))
	for _, permission := range contract.GetActives() {
		actives = append(actives, permissionRequest(permission))
	}
	reqBody := map[string]any{
		"owner_address": tronutils.EncodeCheck(contract.GetOwnerAddress()),
		"owner":         permissionRequest(contract.GetOwner()),
		"actives":       actives,
		"visible":       true,
	}
	if contract.GetWitness() != nil {
		reqBody["witness"] = permissionRequest(contract.GetWitness())
	}
	return t.doTxRequest(ctx, "/wallet/accountpermissionupdate", reqBody)
}

// Block operations

// The block endpoints answer with core.Block's shape, while the interface
// returns BlockExtention, which gRPC builds on the node: the block id next to
// the header, and each transaction wrapped with its id and a Result that
// java-tron sets to true for every transaction of a block. The same wrapping
// is done here, so a block reads the same over both transports.

func (t *HTTPTransport) fetchBlock(ctx context.Context, endpoint string, body any) (*api.BlockExtention, error) {
	respBody, err := t.fetch(ctx, endpoint, body)
	if err != nil {
		return nil, err
	}

	v, err := parseTronJSON(respBody)
	if err != nil {
		return nil, t.wrapErr(endpoint, err)
	}

	block, err := decodeTronBlockExtention(v)
	if err != nil {
		return nil, t.wrapErr(endpoint, fmt.Errorf("unmarshal block: %w (body: %s)", err, truncateBody(respBody)))
	}

	return block, nil
}

func (t *HTTPTransport) fetchBlockList(ctx context.Context, endpoint string, body any) (*api.BlockListExtention, error) {
	respBody, err := t.fetch(ctx, endpoint, body)
	if err != nil {
		return nil, err
	}

	v, err := parseTronJSON(respBody)
	if err != nil {
		return nil, t.wrapErr(endpoint, err)
	}

	// {"block": [...]}, or {} when the range holds no block.
	obj, ok := v.(map[string]any)
	if !ok {
		return nil, t.wrapErr(endpoint, fmt.Errorf("unmarshal block list: expected a JSON object (body: %s)", truncateBody(respBody)))
	}
	blocks, _ := obj["block"].([]any)

	result := &api.BlockListExtention{}
	for i, item := range blocks {
		block, err := decodeTronBlockExtention(item)
		if err != nil {
			return nil, t.wrapErr(endpoint, fmt.Errorf("unmarshal block %d: %w", i, err))
		}
		result.Block = append(result.Block, block)
	}

	return result, nil
}

func (t *HTTPTransport) GetNowBlock(ctx context.Context) (*api.BlockExtention, error) {
	return t.fetchBlock(ctx, "/wallet/getnowblock", nil)
}

func (t *HTTPTransport) GetBlockByNum(ctx context.Context, num int64) (*api.BlockExtention, error) {
	return t.fetchBlock(ctx, "/wallet/getblockbynum", map[string]any{"num": num})
}

func (t *HTTPTransport) GetBlockById(ctx context.Context, id []byte) (*core.Block, error) {
	reqBody := map[string]any{
		"value": hex.EncodeToString(id),
	}

	result := &core.Block{}
	if err := t.fetchTron(ctx, "/wallet/getblockbyid", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) GetBlockByLimitNext(ctx context.Context, start, end int64) (*api.BlockListExtention, error) {
	return t.fetchBlockList(ctx, "/wallet/getblockbylimitnext", map[string]any{"startNum": start, "endNum": end})
}

func (t *HTTPTransport) GetBlockByLatestNum(ctx context.Context, num int64) (*api.BlockListExtention, error) {
	return t.fetchBlockList(ctx, "/wallet/getblockbylatestnum", map[string]any{"num": num})
}

// GetTransactionInfoByBlockNum unwraps the bare JSON array the node answers
// with into TransactionInfoList.
func (t *HTTPTransport) GetTransactionInfoByBlockNum(ctx context.Context, num int64) (*api.TransactionInfoList, error) {
	const endpoint = "/wallet/gettransactioninfobyblocknum"

	respBody, err := t.fetch(ctx, endpoint, map[string]any{"num": num})
	if err != nil {
		return nil, err
	}

	v, err := parseTronJSON(respBody)
	if err != nil {
		return nil, t.wrapErr(endpoint, err)
	}

	result := &api.TransactionInfoList{}
	items, ok := v.([]any)
	if !ok {
		// A block without transactions is answered with {} rather than [].
		if obj, isObj := v.(map[string]any); isObj && len(obj) == 0 {
			return result, nil
		}
		return nil, t.wrapErr(endpoint, fmt.Errorf("unmarshal response: expected a JSON array (body: %s)", truncateBody(respBody)))
	}

	for i, item := range items {
		info := &core.TransactionInfo{}
		if err := decodeTronMessage(item, info.ProtoReflect()); err != nil {
			return nil, t.wrapErr(endpoint, fmt.Errorf("unmarshal transaction info %d: %w", i, err))
		}
		result.TransactionInfo = append(result.TransactionInfo, info)
	}

	return result, nil
}

// Transaction operations

func (t *HTTPTransport) GetTransactionById(ctx context.Context, id []byte) (*core.Transaction, error) {
	reqBody := map[string]any{
		"value": hex.EncodeToString(id),
	}

	result := &core.Transaction{}
	if err := t.fetchTron(ctx, "/wallet/gettransactionbyid", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetTransactionInfoById reads a receipt. Every bytes field of it - the id
// Client.GetTransactionInfoByHash matches against the hash it asked for,
// contract_address, resMessage, the logs - arrives as hex, which protojson used
// to read as base64: the id never matched, so every receipt reached the caller
// as ErrTransactionInfoNotFound.
func (t *HTTPTransport) GetTransactionInfoById(ctx context.Context, id []byte) (*core.TransactionInfo, error) {
	reqBody := map[string]any{
		"value": hex.EncodeToString(id),
	}

	result := &core.TransactionInfo{}
	if err := t.fetchTron(ctx, "/wallet/gettransactioninfobyid", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

// httpBroadcastResponse is a helper struct for parsing HTTP API broadcast responses.
//
// api.Return.Message is a protobuf bytes field, which protojson insists on decoding
// as base64, but the node sends a plain-text sentence there ("Validate signature
// error: ..."). Unmarshaling straight into api.Return therefore fails outright, so
// every rejection reached the caller as an opaque protojson error instead of a
// BroadcastError carrying the response code.
type httpBroadcastResponse struct {
	Result  bool   `json:"result"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// BroadcastTransaction submits a signed transaction over /wallet/broadcasthex.
//
// The hex endpoint takes the marshalled protobuf, so the signed bytes reach the
// node exactly as they were signed. /wallet/broadcasttransaction cannot offer
// that: it rebuilds the transaction from the JSON "raw_data" object - it ignores
// "raw_data_hex" and "txID" entirely - which means every byte field, every
// address and every contract-specific Any would have to be re-rendered in
// Tron's own JSON dialect, and any discrepancy would change the hash the
// signature covers. protojson cannot produce that dialect at all: it emits
// base64 where Tron reads hex and "@type" where Tron reads "type_url"/"value",
// and a node handed such a body answers "class java.lang.NullPointerException".
func (t *HTTPTransport) BroadcastTransaction(ctx context.Context, tx *core.Transaction) (*api.Return, error) {
	txBytes, err := proto.Marshal(tx)
	if err != nil {
		return nil, fmt.Errorf("marshal transaction: %w", err)
	}

	reqBody := map[string]any{
		"transaction": hex.EncodeToString(txBytes),
	}

	var resp httpBroadcastResponse
	if err := t.fetchJSON(ctx, "/wallet/broadcasthex", reqBody, &resp); err != nil {
		return nil, err
	}

	return &api.Return{
		Result:  resp.Result,
		Code:    api.ReturnResponseCode(api.ReturnResponseCode_value[resp.Code]),
		Message: []byte(resp.Message),
	}, nil
}

func (t *HTTPTransport) CreateTransaction(ctx context.Context, contract *core.TransferContract) (*api.TransactionExtention, error) {
	reqBody := map[string]any{
		"owner_address": tronutils.EncodeCheck(contract.OwnerAddress),
		"to_address":    tronutils.EncodeCheck(contract.ToAddress),
		"amount":        contract.Amount,
		"visible":       true,
	}

	return t.doTxRequest(ctx, "/wallet/createtransaction", reqBody)
}

// Contract operations

func (t *HTTPTransport) TriggerContract(ctx context.Context, contract *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	reqBody := map[string]any{
		"owner_address":    tronutils.EncodeCheck(contract.OwnerAddress),
		"contract_address": tronutils.EncodeCheck(contract.ContractAddress),
		"data":             hex.EncodeToString(contract.Data),
		"visible":          true,
	}

	if contract.CallValue > 0 {
		reqBody["call_value"] = contract.CallValue
	}
	if contract.CallTokenValue > 0 {
		reqBody["call_token_value"] = contract.CallTokenValue
		reqBody["token_id"] = contract.TokenId
	}

	return t.doTxRequestWrapped(ctx, "/wallet/triggersmartcontract", reqBody)
}

// constantCallRequest is the body of the two endpoints that run a call without
// broadcasting it.
func constantCallRequest(contract *core.TriggerSmartContract) map[string]any {
	reqBody := map[string]any{
		"owner_address": hexAddress(contract.OwnerAddress),
		"data":          hex.EncodeToString(contract.Data),
	}

	// An empty contract address is how a deployment is expressed: the node then
	// reads data as creation bytecode and runs the constructor. The field has to
	// be left out entirely - an explicit "" is rejected outright ("invalid
	// address for field ... contract_address").
	if len(contract.ContractAddress) > 0 {
		reqBody["contract_address"] = hexAddress(contract.ContractAddress)
	}

	if contract.CallValue > 0 {
		reqBody["call_value"] = contract.CallValue
	}
	if contract.CallTokenValue > 0 {
		reqBody["call_token_value"] = contract.CallTokenValue
		reqBody["token_id"] = contract.TokenId
	}

	return reqBody
}

// TriggerConstantContract decodes the whole TransactionExtention the node
// answers with. Code and message matter as much as constant_result: a revert
// arrives as result.result = true with the failure only in message.
func (t *HTTPTransport) TriggerConstantContract(ctx context.Context, contract *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	result := &api.TransactionExtention{}
	if err := t.fetchTron(ctx, "/wallet/triggerconstantcontract", constantCallRequest(contract), result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) EstimateEnergy(ctx context.Context, contract *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
	result := &api.EstimateEnergyMessage{}
	if err := t.fetchTron(ctx, "/wallet/estimateenergy", constantCallRequest(contract), result); err != nil {
		return nil, err
	}

	return result, nil
}

// DeployContract sends the ABI as the bare array of entries. The node's
// servlet wraps whatever "abi" holds in {"entrys": …} itself, so the whole ABI
// message - which is what this used to send - became {"entrys":{"entrys":[…]}}
// and the contract was stored with a single empty entry: deployed, but not
// callable from anything that reads its ABI.
//
// The entries are rendered by protojson with proto field names; the node's
// parser takes those, and the enum names ("Function", "Nonpayable") are the
// protobuf ones it expects.
func (t *HTTPTransport) DeployContract(ctx context.Context, contract *core.CreateSmartContract) (*api.TransactionExtention, error) {
	const endpoint = "/wallet/deploycontract"

	newContract := contract.GetNewContract()
	reqBody := map[string]any{
		"owner_address":                 tronutils.EncodeCheck(contract.OwnerAddress),
		"name":                          newContract.GetName(),
		"bytecode":                      hex.EncodeToString(newContract.GetBytecode()),
		"consume_user_resource_percent": newContract.GetConsumeUserResourcePercent(),
		"origin_energy_limit":           newContract.GetOriginEnergyLimit(),
		"visible":                       true,
	}

	if newContract.GetCallValue() > 0 {
		reqBody["call_value"] = newContract.GetCallValue()
	}
	if contract.GetCallTokenValue() > 0 {
		reqBody["call_token_value"] = contract.GetCallTokenValue()
		reqBody["token_id"] = contract.GetTokenId()
	}

	if newContract.GetAbi() != nil {
		entries, err := abiEntriesJSON(newContract.GetAbi())
		if err != nil {
			return nil, t.wrapErr(endpoint, err)
		}
		reqBody["abi"] = entries
	}

	return t.doTxRequest(ctx, endpoint, reqBody)
}

// abiEntriesJSON renders the entries of an ABI as the JSON array the
// deployment endpoint expects in "abi".
func abiEntriesJSON(contractABI *core.SmartContract_ABI) (json.RawMessage, error) {
	entries := make([]json.RawMessage, 0, len(contractABI.GetEntrys()))
	for i, entry := range contractABI.GetEntrys() {
		b, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(entry)
		if err != nil {
			return nil, fmt.Errorf("marshal abi entry %d: %w", i, err)
		}
		entries = append(entries, b)
	}

	b, err := json.Marshal(entries)
	if err != nil {
		return nil, fmt.Errorf("marshal abi: %w", err)
	}

	return b, nil
}

func (t *HTTPTransport) GetContract(ctx context.Context, address []byte) (*core.SmartContract, error) {
	reqBody := map[string]any{
		"value": hexAddress(address),
	}

	result := &core.SmartContract{}
	if err := t.fetchTron(ctx, "/wallet/getcontract", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) UpdateSetting(ctx context.Context, contract *core.UpdateSettingContract) (*api.TransactionExtention, error) {
	reqBody := map[string]any{
		"owner_address":                 tronutils.EncodeCheck(contract.OwnerAddress),
		"contract_address":              tronutils.EncodeCheck(contract.ContractAddress),
		"consume_user_resource_percent": contract.ConsumeUserResourcePercent,
		"visible":                       true,
	}

	return t.doTxRequest(ctx, "/wallet/updatesetting", reqBody)
}

func (t *HTTPTransport) UpdateEnergyLimit(ctx context.Context, contract *core.UpdateEnergyLimitContract) (*api.TransactionExtention, error) {
	reqBody := map[string]any{
		"owner_address":       tronutils.EncodeCheck(contract.OwnerAddress),
		"contract_address":    tronutils.EncodeCheck(contract.ContractAddress),
		"origin_energy_limit": contract.OriginEnergyLimit,
		"visible":             true,
	}

	return t.doTxRequest(ctx, "/wallet/updateenergylimit", reqBody)
}

// Resource operations

func (t *HTTPTransport) GetAccountResourceMessage(ctx context.Context, account *core.Account) (*api.AccountResourceMessage, error) {
	return t.GetAccountResource(ctx, account)
}

// delegatedResources performs one of the two delegation-record endpoints, which
// differ only in their path.
func (t *HTTPTransport) delegatedResources(ctx context.Context, endpoint string, msg *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
	reqBody := map[string]any{
		"fromAddress": hexAddress(msg.FromAddress),
		"toAddress":   hexAddress(msg.ToAddress),
	}

	result := &api.DelegatedResourceList{}
	if err := t.fetchTron(ctx, endpoint, reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

// delegationIndex performs one of the two account-index endpoints.
func (t *HTTPTransport) delegationIndex(ctx context.Context, endpoint string, address []byte) (*core.DelegatedResourceAccountIndex, error) {
	reqBody := map[string]any{
		"value": hexAddress(address),
	}

	result := &core.DelegatedResourceAccountIndex{}
	if err := t.fetchTron(ctx, endpoint, reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) GetDelegatedResource(ctx context.Context, msg *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
	return t.delegatedResources(ctx, "/wallet/getdelegatedresource", msg)
}

func (t *HTTPTransport) GetDelegatedResourceV2(ctx context.Context, msg *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
	return t.delegatedResources(ctx, "/wallet/getdelegatedresourcev2", msg)
}

func (t *HTTPTransport) GetDelegatedResourceAccountIndex(ctx context.Context, address []byte) (*core.DelegatedResourceAccountIndex, error) {
	return t.delegationIndex(ctx, "/wallet/getdelegatedresourceaccountindex", address)
}

func (t *HTTPTransport) GetDelegatedResourceAccountIndexV2(ctx context.Context, address []byte) (*core.DelegatedResourceAccountIndex, error) {
	return t.delegationIndex(ctx, "/wallet/getdelegatedresourceaccountindexv2", address)
}

func (t *HTTPTransport) GetCanDelegatedMaxSize(ctx context.Context, msg *api.CanDelegatedMaxSizeRequestMessage) (*api.CanDelegatedMaxSizeResponseMessage, error) {
	reqBody := map[string]any{
		"owner_address": hexAddress(msg.OwnerAddress),
		"type":          msg.Type,
	}

	result := &api.CanDelegatedMaxSizeResponseMessage{}
	if err := t.fetchTron(ctx, "/wallet/getcandelegatedmaxsize", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) DelegateResource(ctx context.Context, contract *core.DelegateResourceContract) (*api.TransactionExtention, error) {
	reqBody := map[string]any{
		"owner_address":    tronutils.EncodeCheck(contract.OwnerAddress),
		"receiver_address": tronutils.EncodeCheck(contract.ReceiverAddress),
		"balance":          contract.Balance,
		"resource":         contract.Resource.String(),
		"lock":             contract.Lock,
		"visible":          true,
	}

	if contract.LockPeriod > 0 {
		reqBody["lock_period"] = contract.LockPeriod
	}

	return t.doTxRequest(ctx, "/wallet/delegateresource", reqBody)
}

func (t *HTTPTransport) UnDelegateResource(ctx context.Context, contract *core.UnDelegateResourceContract) (*api.TransactionExtention, error) {
	reqBody := map[string]any{
		"owner_address":    tronutils.EncodeCheck(contract.OwnerAddress),
		"receiver_address": tronutils.EncodeCheck(contract.ReceiverAddress),
		"balance":          contract.Balance,
		"resource":         contract.Resource.String(),
		"visible":          true,
	}

	return t.doTxRequest(ctx, "/wallet/undelegateresource", reqBody)
}

// Staking operations (Stake 2.0)

func (t *HTTPTransport) FreezeBalanceV2(ctx context.Context, contract *core.FreezeBalanceV2Contract) (*api.TransactionExtention, error) {
	reqBody := map[string]any{
		"owner_address":  tronutils.EncodeCheck(contract.OwnerAddress),
		"frozen_balance": contract.FrozenBalance,
		"resource":       contract.Resource.String(),
		"visible":        true,
	}

	return t.doTxRequest(ctx, "/wallet/freezebalancev2", reqBody)
}

func (t *HTTPTransport) UnfreezeBalanceV2(ctx context.Context, contract *core.UnfreezeBalanceV2Contract) (*api.TransactionExtention, error) {
	reqBody := map[string]any{
		"owner_address":    tronutils.EncodeCheck(contract.OwnerAddress),
		"unfreeze_balance": contract.UnfreezeBalance,
		"resource":         contract.Resource.String(),
		"visible":          true,
	}

	return t.doTxRequest(ctx, "/wallet/unfreezebalancev2", reqBody)
}

func (t *HTTPTransport) WithdrawExpireUnfreeze(ctx context.Context, contract *core.WithdrawExpireUnfreezeContract) (*api.TransactionExtention, error) {
	reqBody := map[string]any{
		"owner_address": tronutils.EncodeCheck(contract.OwnerAddress),
		"visible":       true,
	}

	return t.doTxRequest(ctx, "/wallet/withdrawexpireunfreeze", reqBody)
}

func (t *HTTPTransport) CancelAllUnfreezeV2(ctx context.Context, contract *core.CancelAllUnfreezeV2Contract) (*api.TransactionExtention, error) {
	reqBody := map[string]any{
		"owner_address": tronutils.EncodeCheck(contract.OwnerAddress),
		"visible":       true,
	}

	return t.doTxRequest(ctx, "/wallet/cancelallunfreezev2", reqBody)
}

func (t *HTTPTransport) GetAvailableUnfreezeCount(ctx context.Context, msg *api.GetAvailableUnfreezeCountRequestMessage) (*api.GetAvailableUnfreezeCountResponseMessage, error) {
	reqBody := map[string]any{
		"owner_address": hexAddress(msg.OwnerAddress),
	}

	result := &api.GetAvailableUnfreezeCountResponseMessage{}
	if err := t.fetchTron(ctx, "/wallet/getavailableunfreezecount", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) GetCanWithdrawUnfreezeAmount(ctx context.Context, msg *api.CanWithdrawUnfreezeAmountRequestMessage) (*api.CanWithdrawUnfreezeAmountResponseMessage, error) {
	reqBody := map[string]any{
		"owner_address": hexAddress(msg.OwnerAddress),
		"timestamp":     msg.Timestamp,
	}

	result := &api.CanWithdrawUnfreezeAmountResponseMessage{}
	if err := t.fetchTron(ctx, "/wallet/getcanwithdrawunfreezeamount", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

// Witness operations

func (t *HTTPTransport) VoteWitnessAccount(ctx context.Context, contract *core.VoteWitnessContract) (*api.TransactionExtention, error) {
	votes := make([]map[string]any, len(contract.Votes))
	for i, vote := range contract.Votes {
		votes[i] = map[string]any{
			"vote_address": tronutils.EncodeCheck(vote.VoteAddress),
			"vote_count":   vote.VoteCount,
		}
	}

	reqBody := map[string]any{
		"owner_address": tronutils.EncodeCheck(contract.OwnerAddress),
		"votes":         votes,
		"visible":       true,
	}

	return t.doTxRequest(ctx, "/wallet/votewitnessaccount", reqBody)
}

func (t *HTTPTransport) WithdrawBalance(ctx context.Context, contract *core.WithdrawBalanceContract) (*api.TransactionExtention, error) {
	reqBody := map[string]any{
		"owner_address": tronutils.EncodeCheck(contract.OwnerAddress),
		"visible":       true,
	}

	return t.doTxRequest(ctx, "/wallet/withdrawbalance", reqBody)
}

func (t *HTTPTransport) ListWitnesses(ctx context.Context) (*api.WitnessList, error) {
	result := &api.WitnessList{}
	if err := t.fetchTron(ctx, "/wallet/listwitnesses", nil, result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetRewardInfo uses the camelCase /wallet/getReward endpoint (the lowercase path
// returns HTTP 405) and its response field is "reward", not NumberMessage's "num".
func (t *HTTPTransport) GetRewardInfo(ctx context.Context, address []byte) (*api.NumberMessage, error) {
	reqBody := map[string]any{
		"address": tronutils.EncodeCheck(address),
		"visible": true,
	}

	var resp struct {
		Reward int64 `json:"reward"`
	}
	if err := t.fetchJSON(ctx, "/wallet/getReward", reqBody, &resp); err != nil {
		return nil, err
	}

	return &api.NumberMessage{Num: resp.Reward}, nil
}

// GetBrokerageInfo uses the camelCase /wallet/getBrokerage endpoint (the lowercase
// path returns HTTP 405) and its response field is "brokerage", not "num".
func (t *HTTPTransport) GetBrokerageInfo(ctx context.Context, address []byte) (*api.NumberMessage, error) {
	reqBody := map[string]any{
		"address": tronutils.EncodeCheck(address),
		"visible": true,
	}

	var resp struct {
		Brokerage int64 `json:"brokerage"`
	}
	if err := t.fetchJSON(ctx, "/wallet/getBrokerage", reqBody, &resp); err != nil {
		return nil, err
	}

	return &api.NumberMessage{Num: resp.Brokerage}, nil
}

// Asset operations

// GetAssetIssueById looks up a TRC10 asset by its id, a decimal string that is
// sent as it stands - unlike the name GetAssetIssueListByName takes.
func (t *HTTPTransport) GetAssetIssueById(ctx context.Context, id []byte) (*core.AssetIssueContract, error) {
	reqBody := map[string]any{
		"value": string(id),
	}

	result := &core.AssetIssueContract{}
	if err := t.fetchTron(ctx, "/wallet/getassetissuebyid", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetAssetIssueListByName looks up TRC10 assets by name.
//
// The name goes over the wire hex-encoded: this endpoint takes a bytes field,
// and the plain name is rejected outright with "invalid characters encountered
// in Hex string".
func (t *HTTPTransport) GetAssetIssueListByName(ctx context.Context, name []byte) (*api.AssetIssueList, error) {
	reqBody := map[string]any{
		"value": hex.EncodeToString(name),
	}

	result := &api.AssetIssueList{}
	if err := t.fetchTron(ctx, "/wallet/getassetissuelistbyname", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

// Network operations

func (t *HTTPTransport) ListNodes(ctx context.Context) (*api.NodeList, error) {
	result := &api.NodeList{}
	if err := t.fetchTron(ctx, "/wallet/listnodes", nil, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) GetNodeInfo(ctx context.Context) (*core.NodeInfo, error) {
	result := &core.NodeInfo{}
	if err := t.fetchTron(ctx, "/wallet/getnodeinfo", nil, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) GetChainParameters(ctx context.Context) (*core.ChainParameters, error) {
	result := &core.ChainParameters{}
	if err := t.fetchTron(ctx, "/wallet/getchainparameters", nil, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) GetNextMaintenanceTime(ctx context.Context) (*api.NumberMessage, error) {
	result := &api.NumberMessage{}
	if err := t.fetchTron(ctx, "/wallet/getnextmaintenancetime", nil, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) TotalTransaction(ctx context.Context) (*api.NumberMessage, error) {
	result := &api.NumberMessage{}
	if err := t.fetchTron(ctx, "/wallet/totaltransaction", nil, result); err != nil {
		return nil, err
	}

	return result, nil
}
