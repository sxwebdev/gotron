package client

import (
	"bytes"
	"context"
	"encoding/base64"
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

// transformTronJSON converts Tron's non-standard JSON format to standard protobuf JSON format.
// It handles:
// 1. Any types: {"type_url": "...", "value": {...}} -> {"@type": "...", ...fields...}
// 2. Field name normalization (e.g., blockID -> blockid, txID -> txid)
// 3. Hex to base64 conversion for bytes fields
func transformTronJSON(data any) any {
	return transformTronJSONWithKey(data, "")
}

func transformTronJSONWithKey(data any, fieldName string) any {
	switch v := data.(type) {
	case map[string]any:
		// Check if this is a Tron-style Any type (has type_url and value)
		typeURL, hasTypeURL := v["type_url"].(string)
		valueObj, hasValue := v["value"].(map[string]any)

		if hasTypeURL && hasValue && len(v) == 2 {
			// Transform to standard protojson Any format
			result := map[string]any{
				"@type": typeURL,
			}
			// Merge value fields into result
			for k, val := range valueObj {
				result[k] = transformTronJSONWithKey(val, k)
			}
			return result
		}

		// Recursively transform all values with field name normalization
		result := make(map[string]any, len(v))
		for k, val := range v {
			// Normalize field names
			normalizedKey := normalizeFieldName(k)
			result[normalizedKey] = transformTronJSONWithKey(val, k)
		}
		return result

	case []any:
		// Recursively transform array elements
		result := make([]any, len(v))
		for i, val := range v {
			result[i] = transformTronJSONWithKey(val, fieldName)
		}
		return result

	case string:
		// Convert hex strings to base64 for known bytes fields
		if bytesFields[fieldName] && isHexString(v) {
			return hexToBase64(v)
		}
		return v

	default:
		return data
	}
}

// normalizeFieldName converts Tron HTTP API field names to protobuf JSON field names
func normalizeFieldName(name string) string {
	// Map of Tron HTTP API field names to protobuf field names
	fieldMap := map[string]string{
		"blockID": "blockid",
		"txID":    "txid",
	}

	if mapped, ok := fieldMap[name]; ok {
		return mapped
	}
	return name
}

// hexToBase64 converts a hex string to base64 string (for protojson bytes fields)
func hexToBase64(hexStr string) string {
	data, err := hex.DecodeString(hexStr)
	if err != nil {
		return hexStr // Return original if not valid hex
	}
	return base64.StdEncoding.EncodeToString(data)
}

// isHexString checks if a string looks like a hex-encoded bytes value
func isHexString(s string) bool {
	if len(s) == 0 || len(s)%2 != 0 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}

// bytesFields contains field names that should be treated as bytes (hex -> base64)
var bytesFields = map[string]bool{
	// Transaction/block identifiers
	"txid":       true,
	"txID":       true,
	"blockid":    true,
	"blockID":    true,
	"id":         true,
	"parentHash": true,
	"txTrieRoot": true,

	// Address fields (used in various contracts)
	"owner_address":             true,
	"ownerAddress":              true,
	"to_address":                true,
	"toAddress":                 true,
	"contract_address":          true,
	"contractAddress":           true,
	"receiver_address":          true,
	"receiverAddress":           true,
	"resource_receiver_address": true,
	"resourceReceiverAddress":   true,
	"origin_address":            true,
	"originAddress":             true,
	"caller_address":            true,
	"callerAddress":             true,
	"transferTo_address":        true,
	"transferToAddress":         true,
	"account_address":           true,
	"accountAddress":            true,
	"witness_address":           true,
	"witnessAddress":            true,
	"frozen_address":            true,
	"frozenAddress":             true,

	// Signature and data fields
	"witness_signature": true,
	"witnessSignature":  true,
	"signature":         true,
	"data":              true,
	"bytecode":          true,
	"code_hash":         true,
	"codeHash":          true,
	"asset_name":        true,
	"assetName":         true,
	"url":               true,
	"description":       true,

	// Log/event fields
	"address": true,
	"topics":  true,

	// Internal transaction fields
	"hash":            true,
	"note":            true,
	"token_info":      true,
	"callValueInfo":   true,
	"extra":           true,
	"contractResult":  true,
	"resMessage":      true,
	"contract_result": true,
}

// transformBlockJSON transforms block response to match BlockExtention proto structure.
// The HTTP API returns transactions as plain Transaction objects, but BlockExtention
// expects TransactionExtention objects with nested transaction field.
func transformBlockJSON(data any) any {
	blockMap, ok := data.(map[string]any)
	if !ok {
		return transformTronJSON(data)
	}

	result := make(map[string]any)

	for k, v := range blockMap {
		normalizedKey := normalizeFieldName(k)

		if normalizedKey == "transactions" {
			// Transform transactions array: wrap each Transaction in TransactionExtention
			txs, ok := v.([]any)
			if ok {
				transformedTxs := make([]any, len(txs))
				for i, tx := range txs {
					transformedTxs[i] = wrapTransactionExtention(tx)
				}
				result[normalizedKey] = transformedTxs
			} else {
				result[normalizedKey] = transformTronJSON(v)
			}
		} else {
			result[normalizedKey] = transformTronJSON(v)
		}
	}

	return result
}

// wrapTransactionExtention wraps a Transaction JSON into TransactionExtention structure
func wrapTransactionExtention(tx any) any {
	txMap, ok := tx.(map[string]any)
	if !ok {
		return transformTronJSON(tx)
	}

	// Extract txID for the txid field and convert hex to base64
	var txid any
	if txID, ok := txMap["txID"].(string); ok {
		if isHexString(txID) {
			txid = hexToBase64(txID)
		} else {
			txid = txID
		}
	}

	// The rest of the fields belong to the nested transaction object
	transactionFields := make(map[string]any)
	for k, v := range txMap {
		if k == "txID" {
			continue // txID goes to txid at TransactionExtention level
		}
		transactionFields[k] = v
	}

	// Build TransactionExtention structure
	result := map[string]any{
		"transaction": transformTronJSON(transactionFields),
	}
	if txid != nil {
		result["txid"] = txid
	}

	return result
}

// transformBlockListJSON transforms block list response to match BlockListExtention proto structure.
func transformBlockListJSON(data any) any {
	// The HTTP API returns an array of blocks directly, but BlockListExtention
	// expects an object with "block" field
	if blocks, ok := data.([]any); ok {
		transformedBlocks := make([]any, len(blocks))
		for i, block := range blocks {
			transformedBlocks[i] = transformBlockJSON(block)
		}
		return map[string]any{
			"block": transformedBlocks,
		}
	}

	// If it's already an object, check for "block" field
	if obj, ok := data.(map[string]any); ok {
		if blocks, ok := obj["block"].([]any); ok {
			transformedBlocks := make([]any, len(blocks))
			for i, block := range blocks {
				transformedBlocks[i] = transformBlockJSON(block)
			}
			result := make(map[string]any)
			for k, v := range obj {
				if k == "block" {
					result[k] = transformedBlocks
				} else {
					result[normalizeFieldName(k)] = transformTronJSON(v)
				}
			}
			return result
		}
	}

	return transformTronJSON(data)
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

// doRequest performs an HTTP POST request to the Tron API
func (t *HTTPTransport) doRequest(ctx context.Context, endpoint string, body any, result proto.Message) error {
	respBody, err := t.doRequestRaw(ctx, endpoint, body)
	if err != nil {
		return err
	}

	if result != nil {
		opts := protojson.UnmarshalOptions{DiscardUnknown: true}
		if err := opts.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unmarshal response: %w (body: %s)", err, string(respBody))
		}
	}

	return nil
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
		return nil, t.wrapErr(endpoint, fmt.Errorf("%w: %s: %s", ErrInvalidTransaction, resp.Result.Code, message))
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

	if resp.Error != "" {
		return nil, t.wrapErr(endpoint, fmt.Errorf("%s", resp.Error))
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

// doRequestTransformed performs an HTTP POST request and transforms Tron's
// non-standard JSON format to standard protobuf JSON before unmarshaling.
// This is needed for endpoints that return protobuf Any types.
func (t *HTTPTransport) doRequestTransformed(ctx context.Context, endpoint string, body any, result proto.Message) error {
	respBody, err := t.doRequestRaw(ctx, endpoint, body)
	if err != nil {
		return err
	}

	if result != nil {
		// Parse JSON into generic structure
		var data any
		if err := json.Unmarshal(respBody, &data); err != nil {
			return fmt.Errorf("parse json: %w (body: %s)", err, string(respBody))
		}

		// Transform Tron's JSON format to standard protobuf JSON
		transformed := transformTronJSON(data)

		// Marshal back to JSON
		transformedJSON, err := json.Marshal(transformed)
		if err != nil {
			return fmt.Errorf("marshal transformed json: %w", err)
		}

		// Unmarshal with protojson
		opts := protojson.UnmarshalOptions{DiscardUnknown: true}
		if err := opts.Unmarshal(transformedJSON, result); err != nil {
			return fmt.Errorf("unmarshal response: %w (body: %s)", err, string(transformedJSON))
		}
	}

	return nil
}

// doBlockRequest performs an HTTP POST request for block endpoints and transforms
// the response to match BlockExtention proto structure.
func (t *HTTPTransport) doBlockRequest(ctx context.Context, endpoint string, body any, result proto.Message) error {
	respBody, err := t.doRequestRaw(ctx, endpoint, body)
	if err != nil {
		return err
	}

	if result != nil {
		// Parse JSON into generic structure
		var data any
		if err := json.Unmarshal(respBody, &data); err != nil {
			return fmt.Errorf("parse json: %w (body: %s)", err, string(respBody))
		}

		// Transform block JSON to match protobuf structure
		transformed := transformBlockJSON(data)

		// Marshal back to JSON
		transformedJSON, err := json.Marshal(transformed)
		if err != nil {
			return fmt.Errorf("marshal transformed json: %w", err)
		}

		// Unmarshal with protojson
		opts := protojson.UnmarshalOptions{DiscardUnknown: true}
		if err := opts.Unmarshal(transformedJSON, result); err != nil {
			return fmt.Errorf("unmarshal response: %w (body: %s)", err, string(transformedJSON))
		}
	}

	return nil
}

// doBlockListRequest performs an HTTP POST request for block list endpoints and transforms
// the response to match BlockListExtention proto structure.
func (t *HTTPTransport) doBlockListRequest(ctx context.Context, endpoint string, body any, result proto.Message) error {
	respBody, err := t.doRequestRaw(ctx, endpoint, body)
	if err != nil {
		return err
	}

	if result != nil {
		// Parse JSON into generic structure
		var data any
		if err := json.Unmarshal(respBody, &data); err != nil {
			return fmt.Errorf("parse json: %w (body: %s)", err, string(respBody))
		}

		// Transform block list JSON to match protobuf structure
		transformed := transformBlockListJSON(data)

		// Marshal back to JSON
		transformedJSON, err := json.Marshal(transformed)
		if err != nil {
			return fmt.Errorf("marshal transformed json: %w", err)
		}

		// Unmarshal with protojson
		opts := protojson.UnmarshalOptions{DiscardUnknown: true}
		if err := opts.Unmarshal(transformedJSON, result); err != nil {
			return fmt.Errorf("unmarshal response: %w (body: %s)", err, string(transformedJSON))
		}
	}

	return nil
}

// httpAccount is a helper struct for parsing HTTP API account response
type httpAccount struct {
	Address               string               `json:"address"`
	Balance               int64                `json:"balance"`
	CreateTime            int64                `json:"create_time"`
	LatestOprationTime    int64                `json:"latest_opration_time"`
	LatestConsumeTime     int64                `json:"latest_consume_time"`
	LatestConsumeFreeTime int64                `json:"latest_consume_free_time"`
	NetWindowSize         int64                `json:"net_window_size"`
	NetWindowOptimized    bool                 `json:"net_window_optimized"`
	AccountResource       *httpAccountResource `json:"account_resource"`
	OwnerPermission       json.RawMessage      `json:"owner_permission"`
	ActivePermission      json.RawMessage      `json:"active_permission"`
	FrozenV2              []httpFreezeV2       `json:"frozenV2"`
	UnfrozenV2            []httpUnFreezeV2     `json:"unfrozenV2"`
	AssetV2               json.RawMessage      `json:"assetV2"`
	FreeAssetNetUsageV2   json.RawMessage      `json:"free_asset_net_usageV2"`
	AssetOptimized        bool                 `json:"asset_optimized"`
}

// httpFreezeV2 is a Stake 2.0 staked-balance entry. Tron omits "type" for
// BANDWIDTH (the zero enum) and omits "amount" when it is zero.
type httpFreezeV2 struct {
	Type   string `json:"type"`
	Amount int64  `json:"amount"`
}

// httpUnFreezeV2 is a Stake 2.0 pending-unstake entry.
type httpUnFreezeV2 struct {
	Type               string `json:"type"`
	UnfreezeAmount     int64  `json:"unfreeze_amount"`
	UnfreezeExpireTime int64  `json:"unfreeze_expire_time"`
}

type httpAccountResource struct {
	LatestConsumeTimeForEnergy                int64 `json:"latest_consume_time_for_energy"`
	EnergyWindowSize                          int64 `json:"energy_window_size"`
	AcquiredDelegatedFrozenV2BalanceForEnergy int64 `json:"acquired_delegated_frozenV2_balance_for_energy"`
	EnergyWindowOptimized                     bool  `json:"energy_window_optimized"`
}

// Account operations

func (t *HTTPTransport) GetAccount(ctx context.Context, account *core.Account) (*core.Account, error) {
	reqBody := map[string]any{
		"address": tronutils.EncodeCheck(account.Address),
		"visible": true,
	}

	respBody, err := t.doRequestRaw(ctx, "/wallet/getaccount", reqBody)
	if err != nil {
		return nil, err
	}

	// Parse into helper struct to handle incompatible JSON format
	var httpAcc httpAccount
	if err := json.Unmarshal(respBody, &httpAcc); err != nil {
		return nil, fmt.Errorf("unmarshal account: %w", err)
	}

	// Convert to protobuf Account
	result := &core.Account{
		Balance:               httpAcc.Balance,
		CreateTime:            httpAcc.CreateTime,
		LatestOprationTime:    httpAcc.LatestOprationTime,
		LatestConsumeTime:     httpAcc.LatestConsumeTime,
		LatestConsumeFreeTime: httpAcc.LatestConsumeFreeTime,
		NetWindowSize:         httpAcc.NetWindowSize,
		NetWindowOptimized:    httpAcc.NetWindowOptimized,
	}

	// Decode address
	if httpAcc.Address != "" {
		result.Address, _ = tronutils.DecodeCheck(httpAcc.Address)
	}

	// Stake 2.0 balances
	for _, item := range httpAcc.FrozenV2 {
		result.FrozenV2 = append(result.FrozenV2, &core.Account_FreezeV2{
			Type:   core.ResourceCode(core.ResourceCode_value[item.Type]),
			Amount: item.Amount,
		})
	}
	for _, item := range httpAcc.UnfrozenV2 {
		result.UnfrozenV2 = append(result.UnfrozenV2, &core.Account_UnFreezeV2{
			Type:               core.ResourceCode(core.ResourceCode_value[item.Type]),
			UnfreezeAmount:     item.UnfreezeAmount,
			UnfreezeExpireTime: item.UnfreezeExpireTime,
		})
	}

	return result, nil
}

// httpAccountResourceMessage is a helper struct for parsing HTTP API account resource response
type httpAccountResourceMessage struct {
	FreeNetLimit      int64           `json:"freeNetLimit"`
	FreeNetUsed       int64           `json:"freeNetUsed"`
	NetLimit          int64           `json:"NetLimit"`
	NetUsed           int64           `json:"NetUsed"`
	TotalNetLimit     int64           `json:"TotalNetLimit"`
	TotalNetWeight    int64           `json:"TotalNetWeight"`
	EnergyLimit       int64           `json:"EnergyLimit"`
	EnergyUsed        int64           `json:"EnergyUsed"`
	TotalEnergyLimit  int64           `json:"TotalEnergyLimit"`
	TotalEnergyWeight int64           `json:"TotalEnergyWeight"`
	AssetNetUsed      json.RawMessage `json:"assetNetUsed"`
	AssetNetLimit     json.RawMessage `json:"assetNetLimit"`
}

func (t *HTTPTransport) GetAccountResource(ctx context.Context, account *core.Account) (*api.AccountResourceMessage, error) {
	reqBody := map[string]any{
		"address": tronutils.EncodeCheck(account.Address),
		"visible": true,
	}

	respBody, err := t.doRequestRaw(ctx, "/wallet/getaccountresource", reqBody)
	if err != nil {
		return nil, err
	}

	// Parse into helper struct to handle incompatible JSON format
	var httpRes httpAccountResourceMessage
	if err := json.Unmarshal(respBody, &httpRes); err != nil {
		return nil, fmt.Errorf("unmarshal account resource: %w", err)
	}

	// Convert to protobuf AccountResourceMessage
	result := &api.AccountResourceMessage{
		FreeNetLimit:      httpRes.FreeNetLimit,
		FreeNetUsed:       httpRes.FreeNetUsed,
		NetLimit:          httpRes.NetLimit,
		NetUsed:           httpRes.NetUsed,
		TotalNetLimit:     httpRes.TotalNetLimit,
		TotalNetWeight:    httpRes.TotalNetWeight,
		EnergyLimit:       httpRes.EnergyLimit,
		EnergyUsed:        httpRes.EnergyUsed,
		TotalEnergyLimit:  httpRes.TotalEnergyLimit,
		TotalEnergyWeight: httpRes.TotalEnergyWeight,
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

// Block operations

func (t *HTTPTransport) GetNowBlock(ctx context.Context) (*api.BlockExtention, error) {
	result := &api.BlockExtention{}
	if err := t.doBlockRequest(ctx, "/wallet/getnowblock", nil, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) GetBlockByNum(ctx context.Context, num int64) (*api.BlockExtention, error) {
	reqBody := map[string]any{
		"num": num,
	}

	result := &api.BlockExtention{}
	if err := t.doBlockRequest(ctx, "/wallet/getblockbynum", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) GetBlockById(ctx context.Context, id []byte) (*core.Block, error) {
	reqBody := map[string]any{
		"value": hex.EncodeToString(id),
	}

	result := &core.Block{}
	// core.Block uses Transaction directly (not TransactionExtention),
	// so we use doRequestTransformed instead of doBlockRequest
	if err := t.doRequestTransformed(ctx, "/wallet/getblockbyid", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) GetBlockByLimitNext(ctx context.Context, start, end int64) (*api.BlockListExtention, error) {
	reqBody := map[string]any{
		"startNum": start,
		"endNum":   end,
	}

	result := &api.BlockListExtention{}
	if err := t.doBlockListRequest(ctx, "/wallet/getblockbylimitnext", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) GetBlockByLatestNum(ctx context.Context, num int64) (*api.BlockListExtention, error) {
	reqBody := map[string]any{
		"num": num,
	}

	result := &api.BlockListExtention{}
	if err := t.doBlockListRequest(ctx, "/wallet/getblockbylatestnum", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) GetTransactionInfoByBlockNum(ctx context.Context, num int64) (*api.TransactionInfoList, error) {
	reqBody := map[string]any{
		"num": num,
	}

	respBody, err := t.doRequestRaw(ctx, "/wallet/gettransactioninfobyblocknum", reqBody)
	if err != nil {
		return nil, err
	}

	// HTTP API returns a JSON array directly, but TransactionInfoList expects an object
	// with "transactionInfo" field. Parse, transform (hex->base64), and wrap.
	var data []any
	if err := json.Unmarshal(respBody, &data); err != nil {
		return nil, fmt.Errorf("parse json: %w (body: %s)", err, string(respBody))
	}

	// Transform each TransactionInfo to convert hex fields to base64
	transformedData := make([]any, len(data))
	for i, item := range data {
		transformedData[i] = transformTronJSON(item)
	}

	// Wrap in expected format
	wrapped := map[string]any{
		"transactionInfo": transformedData,
	}

	wrappedJSON, err := json.Marshal(wrapped)
	if err != nil {
		return nil, fmt.Errorf("marshal transformed json: %w", err)
	}

	result := &api.TransactionInfoList{}
	opts := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err := opts.Unmarshal(wrappedJSON, result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w (body: %s)", err, string(wrappedJSON))
	}

	return result, nil
}

// Transaction operations

func (t *HTTPTransport) GetTransactionById(ctx context.Context, id []byte) (*core.Transaction, error) {
	reqBody := map[string]any{
		"value": hex.EncodeToString(id),
	}

	result := &core.Transaction{}
	if err := t.doRequest(ctx, "/wallet/gettransactionbyid", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) GetTransactionInfoById(ctx context.Context, id []byte) (*core.TransactionInfo, error) {
	reqBody := map[string]any{
		"value": hex.EncodeToString(id),
	}

	result := &core.TransactionInfo{}
	if err := t.doRequest(ctx, "/wallet/gettransactioninfobyid", reqBody, result); err != nil {
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
	Error   string `json:"Error"`
}

func (t *HTTPTransport) BroadcastTransaction(ctx context.Context, tx *core.Transaction) (*api.Return, error) {
	// Convert transaction to JSON using protojson
	txJSON, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(tx)
	if err != nil {
		return nil, fmt.Errorf("marshal transaction: %w", err)
	}

	var reqBody map[string]any
	if err := json.Unmarshal(txJSON, &reqBody); err != nil {
		return nil, fmt.Errorf("unmarshal transaction json: %w", err)
	}
	reqBody["visible"] = true

	respBody, err := t.doRequestRaw(ctx, "/wallet/broadcasttransaction", reqBody)
	if err != nil {
		return nil, err
	}

	var resp httpBroadcastResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, t.wrapErr("/wallet/broadcasttransaction", fmt.Errorf("unmarshal response: %w (body: %s)", err, string(respBody)))
	}

	if resp.Error != "" {
		return nil, t.wrapErr("/wallet/broadcasttransaction", fmt.Errorf("%s", resp.Error))
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

// httpTriggerConstantContractResponse is a helper struct for parsing HTTP API response
type httpTriggerConstantContractResponse struct {
	Result struct {
		Result  bool   `json:"result"`
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"result"`
	ConstantResult []string        `json:"constant_result"`
	EnergyUsed     int64           `json:"energy_used"`
	EnergyPenalty  int64           `json:"energy_penalty"`
	Transaction    json.RawMessage `json:"transaction"`
}

func (t *HTTPTransport) TriggerConstantContract(ctx context.Context, contract *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	reqBody := map[string]any{
		"owner_address":    tronutils.EncodeCheck(contract.OwnerAddress),
		"contract_address": tronutils.EncodeCheck(contract.ContractAddress),
		"data":             hex.EncodeToString(contract.Data),
		"visible":          true,
	}

	if contract.CallValue > 0 {
		reqBody["call_value"] = contract.CallValue
	}

	respBody, err := t.doRequestRaw(ctx, "/wallet/triggerconstantcontract", reqBody)
	if err != nil {
		return nil, err
	}

	// Parse into helper struct to handle constant_result as string array
	var httpRes httpTriggerConstantContractResponse
	if err := json.Unmarshal(respBody, &httpRes); err != nil {
		return nil, fmt.Errorf("unmarshal trigger constant contract response: %w", err)
	}

	// Convert constant_result from hex strings to bytes
	result := &api.TransactionExtention{
		Result: &api.Return{
			Result: httpRes.Result.Result,
		},
		EnergyUsed:    httpRes.EnergyUsed,
		EnergyPenalty: httpRes.EnergyPenalty,
	}

	// Convert constant_result hex strings to bytes
	for _, hexStr := range httpRes.ConstantResult {
		data, err := hex.DecodeString(hexStr)
		if err != nil {
			return nil, fmt.Errorf("decode constant_result hex: %w", err)
		}
		result.ConstantResult = append(result.ConstantResult, data)
	}

	return result, nil
}

func (t *HTTPTransport) EstimateEnergy(ctx context.Context, contract *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
	reqBody := map[string]any{
		"owner_address":    tronutils.EncodeCheck(contract.OwnerAddress),
		"contract_address": tronutils.EncodeCheck(contract.ContractAddress),
		"data":             hex.EncodeToString(contract.Data),
		"visible":          true,
	}

	if contract.CallValue > 0 {
		reqBody["call_value"] = contract.CallValue
	}

	result := &api.EstimateEnergyMessage{}
	if err := t.doRequest(ctx, "/wallet/estimateenergy", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) DeployContract(ctx context.Context, contract *core.CreateSmartContract) (*api.TransactionExtention, error) {
	reqBody := map[string]any{
		"owner_address":                 tronutils.EncodeCheck(contract.OwnerAddress),
		"name":                          contract.NewContract.Name,
		"bytecode":                      hex.EncodeToString(contract.NewContract.Bytecode),
		"consume_user_resource_percent": contract.NewContract.ConsumeUserResourcePercent,
		"origin_energy_limit":           contract.NewContract.OriginEnergyLimit,
		"visible":                       true,
	}

	if contract.NewContract.Abi != nil {
		abiJSON, err := protojson.Marshal(contract.NewContract.Abi)
		if err == nil {
			var abiMap any
			if json.Unmarshal(abiJSON, &abiMap) == nil {
				reqBody["abi"] = abiMap
			}
		}
	}

	return t.doTxRequest(ctx, "/wallet/deploycontract", reqBody)
}

func (t *HTTPTransport) GetContract(ctx context.Context, address []byte) (*core.SmartContract, error) {
	reqBody := map[string]any{
		"value":   tronutils.EncodeCheck(address),
		"visible": true,
	}

	result := &core.SmartContract{}
	if err := t.doRequest(ctx, "/wallet/getcontract", reqBody, result); err != nil {
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

func (t *HTTPTransport) GetDelegatedResource(ctx context.Context, msg *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
	reqBody := map[string]any{
		"fromAddress": tronutils.EncodeCheck(msg.FromAddress),
		"toAddress":   tronutils.EncodeCheck(msg.ToAddress),
		"visible":     true,
	}

	result := &api.DelegatedResourceList{}
	if err := t.doRequest(ctx, "/wallet/getdelegatedresource", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) GetDelegatedResourceV2(ctx context.Context, msg *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
	reqBody := map[string]any{
		"fromAddress": tronutils.EncodeCheck(msg.FromAddress),
		"toAddress":   tronutils.EncodeCheck(msg.ToAddress),
		"visible":     true,
	}

	result := &api.DelegatedResourceList{}
	if err := t.doRequest(ctx, "/wallet/getdelegatedresourcev2", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) GetDelegatedResourceAccountIndex(ctx context.Context, address []byte) (*core.DelegatedResourceAccountIndex, error) {
	reqBody := map[string]any{
		"value":   tronutils.EncodeCheck(address),
		"visible": true,
	}

	result := &core.DelegatedResourceAccountIndex{}
	if err := t.doRequest(ctx, "/wallet/getdelegatedresourceaccountindex", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) GetDelegatedResourceAccountIndexV2(ctx context.Context, address []byte) (*core.DelegatedResourceAccountIndex, error) {
	reqBody := map[string]any{
		"value":   tronutils.EncodeCheck(address),
		"visible": true,
	}

	result := &core.DelegatedResourceAccountIndex{}
	if err := t.doRequest(ctx, "/wallet/getdelegatedresourceaccountindexv2", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) GetCanDelegatedMaxSize(ctx context.Context, msg *api.CanDelegatedMaxSizeRequestMessage) (*api.CanDelegatedMaxSizeResponseMessage, error) {
	reqBody := map[string]any{
		"owner_address": tronutils.EncodeCheck(msg.OwnerAddress),
		"type":          msg.Type,
		"visible":       true,
	}

	result := &api.CanDelegatedMaxSizeResponseMessage{}
	if err := t.doRequest(ctx, "/wallet/getcandelegatedmaxsize", reqBody, result); err != nil {
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
		"owner_address": tronutils.EncodeCheck(msg.OwnerAddress),
		"visible":       true,
	}

	result := &api.GetAvailableUnfreezeCountResponseMessage{}
	if err := t.doRequest(ctx, "/wallet/getavailableunfreezecount", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) GetCanWithdrawUnfreezeAmount(ctx context.Context, msg *api.CanWithdrawUnfreezeAmountRequestMessage) (*api.CanWithdrawUnfreezeAmountResponseMessage, error) {
	reqBody := map[string]any{
		"owner_address": tronutils.EncodeCheck(msg.OwnerAddress),
		"timestamp":     msg.Timestamp,
		"visible":       true,
	}

	result := &api.CanWithdrawUnfreezeAmountResponseMessage{}
	if err := t.doRequest(ctx, "/wallet/getcanwithdrawunfreezeamount", reqBody, result); err != nil {
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

// httpWitness is a helper struct for parsing HTTP API witness list response.
// /wallet/listwitnesses ignores "visible" and always returns hex addresses.
type httpWitness struct {
	Address        string `json:"address"`
	VoteCount      int64  `json:"voteCount"`
	URL            string `json:"url"`
	TotalProduced  int64  `json:"totalProduced"`
	TotalMissed    int64  `json:"totalMissed"`
	LatestBlockNum int64  `json:"latestBlockNum"`
	LatestSlotNum  int64  `json:"latestSlotNum"`
	IsJobs         bool   `json:"isJobs"`
}

func (t *HTTPTransport) ListWitnesses(ctx context.Context) (*api.WitnessList, error) {
	respBody, err := t.doRequestRaw(ctx, "/wallet/listwitnesses", nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Witnesses []httpWitness `json:"witnesses"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal witness list: %w", err)
	}

	result := &api.WitnessList{Witnesses: make([]*core.Witness, 0, len(resp.Witnesses))}
	for _, w := range resp.Witnesses {
		addr, err := hex.DecodeString(w.Address)
		if err != nil {
			return nil, fmt.Errorf("decode witness address %q: %w", w.Address, err)
		}

		result.Witnesses = append(result.Witnesses, &core.Witness{
			Address:        addr,
			VoteCount:      w.VoteCount,
			Url:            w.URL,
			TotalProduced:  w.TotalProduced,
			TotalMissed:    w.TotalMissed,
			LatestBlockNum: w.LatestBlockNum,
			LatestSlotNum:  w.LatestSlotNum,
			IsJobs:         w.IsJobs,
		})
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

	respBody, err := t.doRequestRaw(ctx, "/wallet/getReward", reqBody)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Reward int64 `json:"reward"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal reward: %w", err)
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

	respBody, err := t.doRequestRaw(ctx, "/wallet/getBrokerage", reqBody)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Brokerage int64 `json:"brokerage"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal brokerage: %w", err)
	}

	return &api.NumberMessage{Num: resp.Brokerage}, nil
}

// Asset operations

func (t *HTTPTransport) GetAssetIssueById(ctx context.Context, id []byte) (*core.AssetIssueContract, error) {
	reqBody := map[string]any{
		"value": string(id),
	}

	result := &core.AssetIssueContract{}
	if err := t.doRequest(ctx, "/wallet/getassetissuebyid", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) GetAssetIssueListByName(ctx context.Context, name []byte) (*api.AssetIssueList, error) {
	reqBody := map[string]any{
		"value": string(name),
	}

	result := &api.AssetIssueList{}
	if err := t.doRequest(ctx, "/wallet/getassetissuelistbyname", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

// Network operations

func (t *HTTPTransport) ListNodes(ctx context.Context) (*api.NodeList, error) {
	result := &api.NodeList{}
	if err := t.doRequest(ctx, "/wallet/listnodes", nil, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) GetNodeInfo(ctx context.Context) (*core.NodeInfo, error) {
	result := &core.NodeInfo{}
	if err := t.doRequest(ctx, "/wallet/getnodeinfo", nil, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) GetChainParameters(ctx context.Context) (*core.ChainParameters, error) {
	result := &core.ChainParameters{}
	if err := t.doRequest(ctx, "/wallet/getchainparameters", nil, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) GetNextMaintenanceTime(ctx context.Context) (*api.NumberMessage, error) {
	result := &api.NumberMessage{}
	if err := t.doRequest(ctx, "/wallet/getnextmaintenancetime", nil, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) TotalTransaction(ctx context.Context) (*api.NumberMessage, error) {
	result := &api.NumberMessage{}
	if err := t.doRequest(ctx, "/wallet/totaltransaction", nil, result); err != nil {
		return nil, err
	}

	return result, nil
}
