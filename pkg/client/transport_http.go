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
func transformTronJSON(data interface{}) interface{} {
	return transformTronJSONWithKey(data, "")
}

func transformTronJSONWithKey(data interface{}, fieldName string) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		// Check if this is a Tron-style Any type (has type_url and value)
		typeURL, hasTypeURL := v["type_url"].(string)
		valueObj, hasValue := v["value"].(map[string]interface{})

		if hasTypeURL && hasValue && len(v) == 2 {
			// Transform to standard protojson Any format
			result := map[string]interface{}{
				"@type": typeURL,
			}
			// Merge value fields into result
			for k, val := range valueObj {
				result[k] = transformTronJSONWithKey(val, k)
			}
			return result
		}

		// Recursively transform all values with field name normalization
		result := make(map[string]interface{}, len(v))
		for k, val := range v {
			// Normalize field names
			normalizedKey := normalizeFieldName(k)
			result[normalizedKey] = transformTronJSONWithKey(val, k)
		}
		return result

	case []interface{}:
		// Recursively transform array elements
		result := make([]interface{}, len(v))
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
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
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
func transformBlockJSON(data interface{}) interface{} {
	blockMap, ok := data.(map[string]interface{})
	if !ok {
		return transformTronJSON(data)
	}

	result := make(map[string]interface{})

	for k, v := range blockMap {
		normalizedKey := normalizeFieldName(k)

		if normalizedKey == "transactions" {
			// Transform transactions array: wrap each Transaction in TransactionExtention
			txs, ok := v.([]interface{})
			if ok {
				transformedTxs := make([]interface{}, len(txs))
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
func wrapTransactionExtention(tx interface{}) interface{} {
	txMap, ok := tx.(map[string]interface{})
	if !ok {
		return transformTronJSON(tx)
	}

	// Extract txID for the txid field and convert hex to base64
	var txid interface{}
	if txID, ok := txMap["txID"].(string); ok {
		if isHexString(txID) {
			txid = hexToBase64(txID)
		} else {
			txid = txID
		}
	}

	// The rest of the fields belong to the nested transaction object
	transactionFields := make(map[string]interface{})
	for k, v := range txMap {
		if k == "txID" {
			continue // txID goes to txid at TransactionExtention level
		}
		transactionFields[k] = v
	}

	// Build TransactionExtention structure
	result := map[string]interface{}{
		"transaction": transformTronJSON(transactionFields),
	}
	if txid != nil {
		result["txid"] = txid
	}

	return result
}

// transformBlockListJSON transforms block list response to match BlockListExtention proto structure.
func transformBlockListJSON(data interface{}) interface{} {
	// The HTTP API returns an array of blocks directly, but BlockListExtention
	// expects an object with "block" field
	if blocks, ok := data.([]interface{}); ok {
		transformedBlocks := make([]interface{}, len(blocks))
		for i, block := range blocks {
			transformedBlocks[i] = transformBlockJSON(block)
		}
		return map[string]interface{}{
			"block": transformedBlocks,
		}
	}

	// If it's already an object, check for "block" field
	if obj, ok := data.(map[string]interface{}); ok {
		if blocks, ok := obj["block"].([]interface{}); ok {
			transformedBlocks := make([]interface{}, len(blocks))
			for i, block := range blocks {
				transformedBlocks[i] = transformBlockJSON(block)
			}
			result := make(map[string]interface{})
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
func (t *HTTPTransport) doRequestRaw(ctx context.Context, endpoint string, body interface{}) ([]byte, error) {
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
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, t.wrapErr(endpoint, fmt.Errorf("read response: %w", err))
	}

	if resp.StatusCode != http.StatusOK {
		return nil, t.wrapErr(endpoint, fmt.Errorf("http status %d: %s", resp.StatusCode, string(respBody)))
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
func (t *HTTPTransport) doRequest(ctx context.Context, endpoint string, body interface{}, result proto.Message) error {
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

// doRequestTransformed performs an HTTP POST request and transforms Tron's
// non-standard JSON format to standard protobuf JSON before unmarshaling.
// This is needed for endpoints that return protobuf Any types.
func (t *HTTPTransport) doRequestTransformed(ctx context.Context, endpoint string, body interface{}, result proto.Message) error {
	respBody, err := t.doRequestRaw(ctx, endpoint, body)
	if err != nil {
		return err
	}

	if result != nil {
		// Parse JSON into generic structure
		var data interface{}
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
func (t *HTTPTransport) doBlockRequest(ctx context.Context, endpoint string, body interface{}, result proto.Message) error {
	respBody, err := t.doRequestRaw(ctx, endpoint, body)
	if err != nil {
		return err
	}

	if result != nil {
		// Parse JSON into generic structure
		var data interface{}
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
func (t *HTTPTransport) doBlockListRequest(ctx context.Context, endpoint string, body interface{}, result proto.Message) error {
	respBody, err := t.doRequestRaw(ctx, endpoint, body)
	if err != nil {
		return err
	}

	if result != nil {
		// Parse JSON into generic structure
		var data interface{}
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
	Address                       string `json:"address"`
	Balance                       int64  `json:"balance"`
	CreateTime                    int64  `json:"create_time"`
	LatestOprationTime            int64  `json:"latest_opration_time"`
	LatestConsumeTime             int64  `json:"latest_consume_time"`
	LatestConsumeFreeTime         int64  `json:"latest_consume_free_time"`
	NetWindowSize                 int64  `json:"net_window_size"`
	NetWindowOptimized            bool   `json:"net_window_optimized"`
	AccountResource               *httpAccountResource `json:"account_resource"`
	OwnerPermission               json.RawMessage `json:"owner_permission"`
	ActivePermission              json.RawMessage `json:"active_permission"`
	FrozenV2                      json.RawMessage `json:"frozenV2"`
	AssetV2                       json.RawMessage `json:"assetV2"`
	FreeAssetNetUsageV2           json.RawMessage `json:"free_asset_net_usageV2"`
	AssetOptimized                bool   `json:"asset_optimized"`
}

type httpAccountResource struct {
	LatestConsumeTimeForEnergy                   int64 `json:"latest_consume_time_for_energy"`
	EnergyWindowSize                             int64 `json:"energy_window_size"`
	AcquiredDelegatedFrozenV2BalanceForEnergy    int64 `json:"acquired_delegated_frozenV2_balance_for_energy"`
	EnergyWindowOptimized                        bool  `json:"energy_window_optimized"`
}

// Account operations

func (t *HTTPTransport) GetAccount(ctx context.Context, account *core.Account) (*core.Account, error) {
	reqBody := map[string]interface{}{
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

	return result, nil
}

// httpAccountResourceMessage is a helper struct for parsing HTTP API account resource response
type httpAccountResourceMessage struct {
	FreeNetLimit      int64 `json:"freeNetLimit"`
	FreeNetUsed       int64 `json:"freeNetUsed"`
	NetLimit          int64 `json:"NetLimit"`
	NetUsed           int64 `json:"NetUsed"`
	TotalNetLimit     int64 `json:"TotalNetLimit"`
	TotalNetWeight    int64 `json:"TotalNetWeight"`
	EnergyLimit       int64 `json:"EnergyLimit"`
	EnergyUsed        int64 `json:"EnergyUsed"`
	TotalEnergyLimit  int64 `json:"TotalEnergyLimit"`
	TotalEnergyWeight int64 `json:"TotalEnergyWeight"`
	AssetNetUsed      json.RawMessage `json:"assetNetUsed"`
	AssetNetLimit     json.RawMessage `json:"assetNetLimit"`
}

func (t *HTTPTransport) GetAccountResource(ctx context.Context, account *core.Account) (*api.AccountResourceMessage, error) {
	reqBody := map[string]interface{}{
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
	reqBody := map[string]interface{}{
		"owner_address":   tronutils.EncodeCheck(contract.OwnerAddress),
		"account_address": tronutils.EncodeCheck(contract.AccountAddress),
		"visible":         true,
	}

	result := &api.TransactionExtention{}
	if err := t.doRequest(ctx, "/wallet/createaccount", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
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
	reqBody := map[string]interface{}{
		"num": num,
	}

	result := &api.BlockExtention{}
	if err := t.doBlockRequest(ctx, "/wallet/getblockbynum", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) GetBlockById(ctx context.Context, id []byte) (*core.Block, error) {
	reqBody := map[string]interface{}{
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
	reqBody := map[string]interface{}{
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
	reqBody := map[string]interface{}{
		"num": num,
	}

	result := &api.BlockListExtention{}
	if err := t.doBlockListRequest(ctx, "/wallet/getblockbylatestnum", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) GetTransactionInfoByBlockNum(ctx context.Context, num int64) (*api.TransactionInfoList, error) {
	reqBody := map[string]interface{}{
		"num": num,
	}

	respBody, err := t.doRequestRaw(ctx, "/wallet/gettransactioninfobyblocknum", reqBody)
	if err != nil {
		return nil, err
	}

	// HTTP API returns a JSON array directly, but TransactionInfoList expects an object
	// with "transactionInfo" field. Parse, transform (hex->base64), and wrap.
	var data []interface{}
	if err := json.Unmarshal(respBody, &data); err != nil {
		return nil, fmt.Errorf("parse json: %w (body: %s)", err, string(respBody))
	}

	// Transform each TransactionInfo to convert hex fields to base64
	transformedData := make([]interface{}, len(data))
	for i, item := range data {
		transformedData[i] = transformTronJSON(item)
	}

	// Wrap in expected format
	wrapped := map[string]interface{}{
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
	reqBody := map[string]interface{}{
		"value": hex.EncodeToString(id),
	}

	result := &core.Transaction{}
	if err := t.doRequest(ctx, "/wallet/gettransactionbyid", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) GetTransactionInfoById(ctx context.Context, id []byte) (*core.TransactionInfo, error) {
	reqBody := map[string]interface{}{
		"value": hex.EncodeToString(id),
	}

	result := &core.TransactionInfo{}
	if err := t.doRequest(ctx, "/wallet/gettransactioninfobyid", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) BroadcastTransaction(ctx context.Context, tx *core.Transaction) (*api.Return, error) {
	// Convert transaction to JSON using protojson
	txJSON, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(tx)
	if err != nil {
		return nil, fmt.Errorf("marshal transaction: %w", err)
	}

	var reqBody map[string]interface{}
	if err := json.Unmarshal(txJSON, &reqBody); err != nil {
		return nil, fmt.Errorf("unmarshal transaction json: %w", err)
	}
	reqBody["visible"] = true

	result := &api.Return{}
	if err := t.doRequest(ctx, "/wallet/broadcasttransaction", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) CreateTransaction(ctx context.Context, contract *core.TransferContract) (*api.TransactionExtention, error) {
	reqBody := map[string]interface{}{
		"owner_address": tronutils.EncodeCheck(contract.OwnerAddress),
		"to_address":    tronutils.EncodeCheck(contract.ToAddress),
		"amount":        contract.Amount,
		"visible":       true,
	}

	result := &api.TransactionExtention{}
	if err := t.doRequest(ctx, "/wallet/createtransaction", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

// Contract operations

func (t *HTTPTransport) TriggerContract(ctx context.Context, contract *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	reqBody := map[string]interface{}{
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

	result := &api.TransactionExtention{}
	if err := t.doRequest(ctx, "/wallet/triggersmartcontract", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

// httpTriggerConstantContractResponse is a helper struct for parsing HTTP API response
type httpTriggerConstantContractResponse struct {
	Result struct {
		Result  bool   `json:"result"`
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"result"`
	ConstantResult  []string `json:"constant_result"`
	EnergyUsed      int64    `json:"energy_used"`
	EnergyPenalty   int64    `json:"energy_penalty"`
	Transaction     json.RawMessage `json:"transaction"`
}

func (t *HTTPTransport) TriggerConstantContract(ctx context.Context, contract *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	reqBody := map[string]interface{}{
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
	reqBody := map[string]interface{}{
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
	reqBody := map[string]interface{}{
		"owner_address":                  tronutils.EncodeCheck(contract.OwnerAddress),
		"name":                           contract.NewContract.Name,
		"bytecode":                       hex.EncodeToString(contract.NewContract.Bytecode),
		"consume_user_resource_percent":  contract.NewContract.ConsumeUserResourcePercent,
		"origin_energy_limit":            contract.NewContract.OriginEnergyLimit,
		"visible":                        true,
	}

	if contract.NewContract.Abi != nil {
		abiJSON, err := protojson.Marshal(contract.NewContract.Abi)
		if err == nil {
			var abiMap interface{}
			if json.Unmarshal(abiJSON, &abiMap) == nil {
				reqBody["abi"] = abiMap
			}
		}
	}

	result := &api.TransactionExtention{}
	if err := t.doRequest(ctx, "/wallet/deploycontract", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) GetContract(ctx context.Context, address []byte) (*core.SmartContract, error) {
	reqBody := map[string]interface{}{
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
	reqBody := map[string]interface{}{
		"owner_address":                  tronutils.EncodeCheck(contract.OwnerAddress),
		"contract_address":               tronutils.EncodeCheck(contract.ContractAddress),
		"consume_user_resource_percent":  contract.ConsumeUserResourcePercent,
		"visible":                        true,
	}

	result := &api.TransactionExtention{}
	if err := t.doRequest(ctx, "/wallet/updatesetting", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) UpdateEnergyLimit(ctx context.Context, contract *core.UpdateEnergyLimitContract) (*api.TransactionExtention, error) {
	reqBody := map[string]interface{}{
		"owner_address":        tronutils.EncodeCheck(contract.OwnerAddress),
		"contract_address":     tronutils.EncodeCheck(contract.ContractAddress),
		"origin_energy_limit":  contract.OriginEnergyLimit,
		"visible":              true,
	}

	result := &api.TransactionExtention{}
	if err := t.doRequest(ctx, "/wallet/updateenergylimit", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

// Resource operations

func (t *HTTPTransport) GetAccountResourceMessage(ctx context.Context, account *core.Account) (*api.AccountResourceMessage, error) {
	return t.GetAccountResource(ctx, account)
}

func (t *HTTPTransport) GetDelegatedResource(ctx context.Context, msg *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
	reqBody := map[string]interface{}{
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
	reqBody := map[string]interface{}{
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
	reqBody := map[string]interface{}{
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
	reqBody := map[string]interface{}{
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
	reqBody := map[string]interface{}{
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
	reqBody := map[string]interface{}{
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

	result := &api.TransactionExtention{}
	if err := t.doRequest(ctx, "/wallet/delegateresource", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) UnDelegateResource(ctx context.Context, contract *core.UnDelegateResourceContract) (*api.TransactionExtention, error) {
	reqBody := map[string]interface{}{
		"owner_address":    tronutils.EncodeCheck(contract.OwnerAddress),
		"receiver_address": tronutils.EncodeCheck(contract.ReceiverAddress),
		"balance":          contract.Balance,
		"resource":         contract.Resource.String(),
		"visible":          true,
	}

	result := &api.TransactionExtention{}
	if err := t.doRequest(ctx, "/wallet/undelegateresource", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

// Asset operations

func (t *HTTPTransport) GetAssetIssueById(ctx context.Context, id []byte) (*core.AssetIssueContract, error) {
	reqBody := map[string]interface{}{
		"value": string(id),
	}

	result := &core.AssetIssueContract{}
	if err := t.doRequest(ctx, "/wallet/getassetissuebyid", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) GetAssetIssueListByName(ctx context.Context, name []byte) (*api.AssetIssueList, error) {
	reqBody := map[string]interface{}{
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
