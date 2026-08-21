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

// apiError reports a refusal the node returned as HTTP 200 with an "Error"
// field, which is how the /wallet endpoints answer a request they would not
// process at all - a malformed address, a value out of range.
//
// Without it such a refusal is indistinguishable from an empty answer:
// protojson drops the field as unknown and leaves the zero message behind, so
// "this request was wrong" arrives as "there is nothing there".
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

// fetchJSON decodes an answer with encoding/json instead of protojson, for the
// endpoints whose JSON protojson cannot read.
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

// doRequest performs an HTTP POST request to the Tron API
func (t *HTTPTransport) doRequest(ctx context.Context, endpoint string, body any, result proto.Message) error {
	respBody, err := t.fetch(ctx, endpoint, body)
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

// doRequestTransformed performs an HTTP POST request and transforms Tron's
// non-standard JSON format to standard protobuf JSON before unmarshaling.
// This is needed for endpoints that return protobuf Any types.
func (t *HTTPTransport) doRequestTransformed(ctx context.Context, endpoint string, body any, result proto.Message) error {
	respBody, err := t.fetch(ctx, endpoint, body)
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
	respBody, err := t.fetch(ctx, endpoint, body)
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
	respBody, err := t.fetch(ctx, endpoint, body)
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
	IsWitness             bool                 `json:"is_witness"`
	AccountResource       *httpAccountResource `json:"account_resource"`
	OwnerPermission       *httpPermission      `json:"owner_permission"`
	WitnessPermission     *httpPermission      `json:"witness_permission"`
	ActivePermission      []httpPermission     `json:"active_permission"`
	FrozenV2              []httpFreezeV2       `json:"frozenV2"`
	UnfrozenV2            []httpUnFreezeV2     `json:"unfrozenV2"`
	AssetV2               []httpAssetBalance   `json:"assetV2"`
	FreeAssetNetUsageV2   []httpAssetBalance   `json:"free_asset_net_usageV2"`
	AssetOptimized        bool                 `json:"asset_optimized"`
}

// httpAssetBalance is one entry of a TRC10 id -> amount map. Tron renders a
// protobuf map as an array of key/value objects, which protojson would not read
// as a map either.
type httpAssetBalance struct {
	Key   string `json:"key"`
	Value int64  `json:"value"`
}

func assetMap(entries []httpAssetBalance) map[string]int64 {
	if len(entries) == 0 {
		return nil
	}

	out := make(map[string]int64, len(entries))
	for _, e := range entries {
		out[e.Key] = e.Value
	}

	return out
}

// httpPermission is one entry of an account's multisig permission set. "type"
// is omitted for Owner (the zero enum), and the key addresses are base58
// because the request asks for visible ones.
type httpPermission struct {
	Type           string `json:"type"`
	ID             int32  `json:"id"`
	PermissionName string `json:"permission_name"`
	Threshold      int64  `json:"threshold"`
	ParentID       int32  `json:"parent_id"`
	Operations     string `json:"operations"`
	Keys           []struct {
		Address string `json:"address"`
		Weight  int64  `json:"weight"`
	} `json:"keys"`
}

func (p httpPermission) toProto() (*core.Permission, error) {
	operations, err := hex.DecodeString(p.Operations)
	if err != nil {
		return nil, fmt.Errorf("decode permission operations %q: %w", p.Operations, err)
	}

	// Tron omits "type" for the owner permission because Owner is the protobuf
	// zero value. Every other value must resolve: a map miss would silently
	// become Owner, which PermissionAllows treats as authorizing everything.
	permissionType := core.Permission_Owner
	if p.Type != "" {
		value, ok := core.Permission_PermissionType_value[p.Type]
		if !ok {
			return nil, fmt.Errorf("%w: unknown permission type %q", ErrInvalidPermission, p.Type)
		}

		permissionType = core.Permission_PermissionType(value)
	}

	out := &core.Permission{
		Type:           permissionType,
		Id:             p.ID,
		PermissionName: p.PermissionName,
		Threshold:      p.Threshold,
		ParentId:       p.ParentID,
		Operations:     operations,
	}

	for _, k := range p.Keys {
		address, err := decodeAddress("permission key", k.Address)
		if err != nil {
			return nil, err
		}

		out.Keys = append(out.Keys, &core.Key{Address: address, Weight: k.Weight})
	}

	return out, nil
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

// httpAccountResource mirrors core.Account_AccountResource field for field, in
// its declaration order, so a missing one is visible side by side. Stake 1.0's
// frozen_balance_for_energy is the only omission: it is a nested message the
// chain no longer writes.
type httpAccountResource struct {
	EnergyUsage                               int64 `json:"energy_usage"`
	LatestConsumeTimeForEnergy                int64 `json:"latest_consume_time_for_energy"`
	AcquiredDelegatedFrozenBalanceForEnergy   int64 `json:"acquired_delegated_frozen_balance_for_energy"`
	DelegatedFrozenBalanceForEnergy           int64 `json:"delegated_frozen_balance_for_energy"`
	StorageLimit                              int64 `json:"storage_limit"`
	StorageUsage                              int64 `json:"storage_usage"`
	LatestExchangeStorageTime                 int64 `json:"latest_exchange_storage_time"`
	EnergyWindowSize                          int64 `json:"energy_window_size"`
	DelegatedFrozenV2BalanceForEnergy         int64 `json:"delegated_frozenV2_balance_for_energy"`
	AcquiredDelegatedFrozenV2BalanceForEnergy int64 `json:"acquired_delegated_frozenV2_balance_for_energy"`
	EnergyWindowOptimized                     bool  `json:"energy_window_optimized"`
}

func (r httpAccountResource) toProto() *core.Account_AccountResource {
	return &core.Account_AccountResource{
		EnergyUsage:                               r.EnergyUsage,
		LatestConsumeTimeForEnergy:                r.LatestConsumeTimeForEnergy,
		AcquiredDelegatedFrozenBalanceForEnergy:   r.AcquiredDelegatedFrozenBalanceForEnergy,
		DelegatedFrozenBalanceForEnergy:           r.DelegatedFrozenBalanceForEnergy,
		StorageLimit:                              r.StorageLimit,
		StorageUsage:                              r.StorageUsage,
		LatestExchangeStorageTime:                 r.LatestExchangeStorageTime,
		EnergyWindowSize:                          r.EnergyWindowSize,
		DelegatedFrozenV2BalanceForEnergy:         r.DelegatedFrozenV2BalanceForEnergy,
		AcquiredDelegatedFrozenV2BalanceForEnergy: r.AcquiredDelegatedFrozenV2BalanceForEnergy,
		EnergyWindowOptimized:                     r.EnergyWindowOptimized,
	}
}

// Account operations

func (t *HTTPTransport) GetAccount(ctx context.Context, account *core.Account) (*core.Account, error) {
	reqBody := map[string]any{
		"address": tronutils.EncodeCheck(account.Address),
		"visible": true,
	}

	// Parse into helper struct to handle incompatible JSON format
	var httpAcc httpAccount
	if err := t.fetchJSON(ctx, "/wallet/getaccount", reqBody, &httpAcc); err != nil {
		return nil, err
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
		IsWitness:             httpAcc.IsWitness,
		AssetOptimized:        httpAcc.AssetOptimized,
		AssetV2:               assetMap(httpAcc.AssetV2),
		FreeAssetNetUsageV2:   assetMap(httpAcc.FreeAssetNetUsageV2),
	}

	// An unreadable address is refused rather than dropped: Client.GetAccount
	// compares it against the one it asked for, so a nil here reports a funded
	// account as ErrAccountNotFound.
	if httpAcc.Address != "" {
		address, err := decodeAddress("address", httpAcc.Address)
		if err != nil {
			return nil, t.wrapErr("/wallet/getaccount", err)
		}

		result.Address = address
	}

	if httpAcc.AccountResource != nil {
		result.AccountResource = httpAcc.AccountResource.toProto()
	}

	if httpAcc.OwnerPermission != nil {
		owner, err := httpAcc.OwnerPermission.toProto()
		if err != nil {
			return nil, t.wrapErr("/wallet/getaccount", err)
		}

		result.OwnerPermission = owner
	}

	if httpAcc.WitnessPermission != nil {
		witness, err := httpAcc.WitnessPermission.toProto()
		if err != nil {
			return nil, t.wrapErr("/wallet/getaccount", err)
		}

		result.WitnessPermission = witness
	}

	for _, item := range httpAcc.ActivePermission {
		permission, err := item.toProto()
		if err != nil {
			return nil, t.wrapErr("/wallet/getaccount", err)
		}

		result.ActivePermission = append(result.ActivePermission, permission)
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
	FreeNetLimit      int64              `json:"freeNetLimit"`
	FreeNetUsed       int64              `json:"freeNetUsed"`
	NetLimit          int64              `json:"NetLimit"`
	NetUsed           int64              `json:"NetUsed"`
	TotalNetLimit     int64              `json:"TotalNetLimit"`
	TotalNetWeight    int64              `json:"TotalNetWeight"`
	EnergyLimit       int64              `json:"EnergyLimit"`
	EnergyUsed        int64              `json:"EnergyUsed"`
	TotalEnergyLimit  int64              `json:"TotalEnergyLimit"`
	TotalEnergyWeight int64              `json:"TotalEnergyWeight"`
	AssetNetUsed      []httpAssetBalance `json:"assetNetUsed"`
	AssetNetLimit     []httpAssetBalance `json:"assetNetLimit"`

	TotalTronPowerWeight int64 `json:"TotalTronPowerWeight"`
	TronPowerUsed        int64 `json:"tronPowerUsed"`
	TronPowerLimit       int64 `json:"tronPowerLimit"`
	StorageUsed          int64 `json:"storageUsed"`
	StorageLimit         int64 `json:"storageLimit"`
}

func (t *HTTPTransport) GetAccountResource(ctx context.Context, account *core.Account) (*api.AccountResourceMessage, error) {
	reqBody := map[string]any{
		"address": tronutils.EncodeCheck(account.Address),
		"visible": true,
	}

	// Parse into helper struct to handle incompatible JSON format
	var httpRes httpAccountResourceMessage
	if err := t.fetchJSON(ctx, "/wallet/getaccountresource", reqBody, &httpRes); err != nil {
		return nil, err
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

		AssetNetUsed:  assetMap(httpRes.AssetNetUsed),
		AssetNetLimit: assetMap(httpRes.AssetNetLimit),

		TotalTronPowerWeight: httpRes.TotalTronPowerWeight,
		TronPowerUsed:        httpRes.TronPowerUsed,
		TronPowerLimit:       httpRes.TronPowerLimit,
		StorageUsed:          httpRes.StorageUsed,
		StorageLimit:         httpRes.StorageLimit,
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

	respBody, err := t.fetch(ctx, "/wallet/gettransactioninfobyblocknum", reqBody)
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
		"owner_address": tronutils.EncodeCheck(contract.OwnerAddress),
		"data":          hex.EncodeToString(contract.Data),
		"visible":       true,
	}

	// An empty contract address is how a deployment is expressed: the node then
	// reads data as creation bytecode and runs the constructor. The field has to
	// be left out entirely - EncodeCheck of nothing is a short but perfectly
	// well-formed base58 string, and an explicit "" is rejected outright
	// ("invalid address for field ... contract_address").
	if len(contract.ContractAddress) > 0 {
		reqBody["contract_address"] = tronutils.EncodeCheck(contract.ContractAddress)
	}

	if contract.CallValue > 0 {
		reqBody["call_value"] = contract.CallValue
	}

	// Parse into helper struct to handle constant_result as string array
	var httpRes httpTriggerConstantContractResponse
	if err := t.fetchJSON(ctx, "/wallet/triggerconstantcontract", reqBody, &httpRes); err != nil {
		return nil, err
	}

	// Code and message must be carried over, not just the boolean: a revert
	// arrives as result.result = true with the failure only in message, so
	// dropping them made every failed call indistinguishable from a successful
	// one over HTTP while gRPC reported it.
	result := &api.TransactionExtention{
		Result: &api.Return{
			Result:  httpRes.Result.Result,
			Code:    api.ReturnResponseCode(api.ReturnResponseCode_value[httpRes.Result.Code]),
			Message: []byte(httpRes.Result.Message),
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
		"owner_address": tronutils.EncodeCheck(contract.OwnerAddress),
		"data":          hex.EncodeToString(contract.Data),
		"visible":       true,
	}

	// Omitted rather than empty for a deployment - see TriggerConstantContract.
	if len(contract.ContractAddress) > 0 {
		reqBody["contract_address"] = tronutils.EncodeCheck(contract.ContractAddress)
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
	// Every bytes field here - origin_address, contract_address, bytecode,
	// code_hash - arrives as hex, so the response needs the hex-to-base64
	// transform before protojson sees it. Plain doRequest fed the hex straight
	// to protojson, which base64-decoded it: a 21-byte address came back as 31
	// bytes of nonsense, so whoever read origin_address (who pays a call's
	// energy, who may update the contract) got a garbage account and no error.
	//
	// visible:true would defeat this in the other direction, making the node
	// answer in base58 for the transform to mangle as if it were hex.
	reqBody := map[string]any{
		"value": hex.EncodeToString(address),
	}

	result := &core.SmartContract{}
	if err := t.doRequestTransformed(ctx, "/wallet/getcontract", reqBody, result); err != nil {
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

// httpDelegatedResource is one record of /wallet/getdelegatedresource[v2].
//
// It exists because the request asks for visible addresses and protojson cannot
// read one: a bytes field is base64 there, and a base58 address is made of
// base64 characters, so it decodes without complaint into a different account.
// The delegation then names an address nobody holds and every later lookup for
// it comes back empty.
type httpDelegatedResource struct {
	From                      string `json:"from"`
	To                        string `json:"to"`
	FrozenBalanceForBandwidth int64  `json:"frozen_balance_for_bandwidth"`
	FrozenBalanceForEnergy    int64  `json:"frozen_balance_for_energy"`
	ExpireTimeForBandwidth    int64  `json:"expire_time_for_bandwidth"`
	ExpireTimeForEnergy       int64  `json:"expire_time_for_energy"`
}

// delegatedResources performs one of the two delegation-record endpoints, which
// differ only in their path.
func (t *HTTPTransport) delegatedResources(ctx context.Context, endpoint string, msg *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
	reqBody := map[string]any{
		"fromAddress": tronutils.EncodeCheck(msg.FromAddress),
		"toAddress":   tronutils.EncodeCheck(msg.ToAddress),
		"visible":     true,
	}

	var parsed struct {
		DelegatedResource []httpDelegatedResource `json:"delegatedResource"`
	}
	if err := t.fetchJSON(ctx, endpoint, reqBody, &parsed); err != nil {
		return nil, err
	}

	result := &api.DelegatedResourceList{
		DelegatedResource: make([]*core.DelegatedResource, 0, len(parsed.DelegatedResource)),
	}

	for _, item := range parsed.DelegatedResource {
		from, err := decodeAddress("from", item.From)
		if err != nil {
			return nil, t.wrapErr(endpoint, err)
		}

		to, err := decodeAddress("to", item.To)
		if err != nil {
			return nil, t.wrapErr(endpoint, err)
		}

		result.DelegatedResource = append(result.DelegatedResource, &core.DelegatedResource{
			From:                      from,
			To:                        to,
			FrozenBalanceForBandwidth: item.FrozenBalanceForBandwidth,
			FrozenBalanceForEnergy:    item.FrozenBalanceForEnergy,
			ExpireTimeForBandwidth:    item.ExpireTimeForBandwidth,
			ExpireTimeForEnergy:       item.ExpireTimeForEnergy,
		})
	}

	return result, nil
}

// delegationIndex performs one of the two account-index endpoints, whose
// addresses are base58 for the same reason.
func (t *HTTPTransport) delegationIndex(ctx context.Context, endpoint string, address []byte) (*core.DelegatedResourceAccountIndex, error) {
	reqBody := map[string]any{
		"value":   tronutils.EncodeCheck(address),
		"visible": true,
	}

	var parsed struct {
		Account      string   `json:"account"`
		FromAccounts []string `json:"fromAccounts"`
		ToAccounts   []string `json:"toAccounts"`
		Timestamp    int64    `json:"timestamp"`
	}
	if err := t.fetchJSON(ctx, endpoint, reqBody, &parsed); err != nil {
		return nil, err
	}

	result := &core.DelegatedResourceAccountIndex{Timestamp: parsed.Timestamp}

	// The queried account is echoed back; an empty index omits it entirely, and
	// DecodeCheck rejects the empty string.
	if parsed.Account != "" {
		account, err := decodeAddress("account", parsed.Account)
		if err != nil {
			return nil, t.wrapErr(endpoint, err)
		}

		result.Account = account
	}

	var err error
	if result.FromAccounts, err = decodeAddresses("fromAccounts", parsed.FromAccounts); err != nil {
		return nil, t.wrapErr(endpoint, err)
	}

	if result.ToAccounts, err = decodeAddresses("toAccounts", parsed.ToAccounts); err != nil {
		return nil, t.wrapErr(endpoint, err)
	}

	return result, nil
}

// decodeAddresses is decodeAddress over a list, refusing the whole list if any
// entry is malformed.
func decodeAddresses(field string, list []string) ([][]byte, error) {
	if len(list) == 0 {
		return nil, nil
	}

	out := make([][]byte, 0, len(list))
	for _, addr := range list {
		decoded, err := decodeAddress(field, addr)
		if err != nil {
			return nil, err
		}

		out = append(out, decoded)
	}

	return out, nil
}

// decodeAddress turns one base58 address of a response into its byte form,
// naming the field it came from. A malformed one is refused rather than kept:
// EncodeCheck turns any bytes back into a plausible-looking address, so a record
// built from a half-decoded one names an account that does not exist, and an
// index silently short of a receiver reads as a delegation that was reclaimed.
func decodeAddress(field, addr string) ([]byte, error) {
	decoded, err := tronutils.DecodeCheck(addr)
	if err != nil {
		return nil, fmt.Errorf("%w: %s %q: %w", ErrInvalidAddress, field, addr, err)
	}

	return decoded, nil
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
	var resp struct {
		Witnesses []httpWitness `json:"witnesses"`
	}
	if err := t.fetchJSON(ctx, "/wallet/listwitnesses", nil, &resp); err != nil {
		return nil, err
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

// httpAssetIssue is one TRC10 asset as /wallet/getassetissue* renders it.
//
// Every bytes field arrives hex-encoded, and protojson reads a bytes field as
// base64: an even-length hex string is valid base64, so it decoded without
// complaint into other bytes entirely. The name of BitTorrent came back as
// "\xe3n\xbd\xef…" and its 21-byte owner as 31 bytes of nonsense - while gRPC,
// which carries the same field as raw bytes, returned it intact.
//
// doRequestTransformed is not usable here: its hex-to-base64 pass is driven by
// the global bytesFields allow-list, and "name" cannot go in it - core.Witness
// and core.SmartContract both have a string field by that name, so any of them
// that happened to spell a hex string would be mangled in turn.
// The fields are in core.AssetIssueContract's own declaration order so that the
// two can be read side by side and a missing one is visible; the bytes ones are
// strings here and decoded below.
type httpAssetIssue struct {
	ID           string `json:"id"`
	OwnerAddress string `json:"owner_address"`
	Name         string `json:"name"`
	Abbr         string `json:"abbr"`
	TotalSupply  int64  `json:"total_supply"`
	FrozenSupply []struct {
		FrozenAmount int64 `json:"frozen_amount"`
		FrozenDays   int64 `json:"frozen_days"`
	} `json:"frozen_supply"`
	TrxNum                  int32  `json:"trx_num"`
	Precision               int32  `json:"precision"`
	Num                     int32  `json:"num"`
	StartTime               int64  `json:"start_time"`
	EndTime                 int64  `json:"end_time"`
	Order                   int64  `json:"order"`
	VoteScore               int32  `json:"vote_score"`
	Description             string `json:"description"`
	URL                     string `json:"url"`
	FreeAssetNetLimit       int64  `json:"free_asset_net_limit"`
	PublicFreeAssetNetLimit int64  `json:"public_free_asset_net_limit"`
	PublicFreeAssetNetUsage int64  `json:"public_free_asset_net_usage"`
	PublicLatestFreeNetTime int64  `json:"public_latest_free_net_time"`
}

func (a httpAssetIssue) toProto() (*core.AssetIssueContract, error) {
	out := &core.AssetIssueContract{
		Id:                      a.ID,
		TotalSupply:             a.TotalSupply,
		TrxNum:                  a.TrxNum,
		Precision:               a.Precision,
		Num:                     a.Num,
		StartTime:               a.StartTime,
		EndTime:                 a.EndTime,
		Order:                   a.Order,
		VoteScore:               a.VoteScore,
		FreeAssetNetLimit:       a.FreeAssetNetLimit,
		PublicFreeAssetNetLimit: a.PublicFreeAssetNetLimit,
		PublicFreeAssetNetUsage: a.PublicFreeAssetNetUsage,
		PublicLatestFreeNetTime: a.PublicLatestFreeNetTime,
	}

	// The issuer is an address, so a caller matches its failure the way they
	// match one from any other read.
	if a.OwnerAddress != "" {
		owner, err := hex.DecodeString(a.OwnerAddress)
		if err != nil {
			return nil, fmt.Errorf("%w: owner_address %q: %w", ErrInvalidAddress, a.OwnerAddress, err)
		}

		out.OwnerAddress = owner
	}

	for _, field := range []struct {
		name string
		hex  string
		out  *[]byte
	}{
		{"name", a.Name, &out.Name},
		{"abbr", a.Abbr, &out.Abbr},
		{"description", a.Description, &out.Description},
		{"url", a.URL, &out.Url},
	} {
		decoded, err := hex.DecodeString(field.hex)
		if err != nil {
			return nil, fmt.Errorf("decode %s %q: %w", field.name, field.hex, err)
		}

		if len(decoded) > 0 {
			*field.out = decoded
		}
	}

	for _, f := range a.FrozenSupply {
		out.FrozenSupply = append(out.FrozenSupply, &core.AssetIssueContract_FrozenSupply{
			FrozenAmount: f.FrozenAmount,
			FrozenDays:   f.FrozenDays,
		})
	}

	return out, nil
}

// GetAssetIssueById looks up a TRC10 asset by its id, a decimal string that is
// sent as it stands - unlike the name GetAssetIssueListByName takes.
func (t *HTTPTransport) GetAssetIssueById(ctx context.Context, id []byte) (*core.AssetIssueContract, error) {
	reqBody := map[string]any{
		"value": string(id),
	}

	var parsed httpAssetIssue
	if err := t.fetchJSON(ctx, "/wallet/getassetissuebyid", reqBody, &parsed); err != nil {
		return nil, err
	}

	result, err := parsed.toProto()
	if err != nil {
		return nil, t.wrapErr("/wallet/getassetissuebyid", err)
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

	var parsed struct {
		AssetIssue []httpAssetIssue `json:"assetIssue"`
	}
	if err := t.fetchJSON(ctx, "/wallet/getassetissuelistbyname", reqBody, &parsed); err != nil {
		return nil, err
	}

	result := &api.AssetIssueList{AssetIssue: make([]*core.AssetIssueContract, 0, len(parsed.AssetIssue))}
	for _, item := range parsed.AssetIssue {
		asset, err := item.toProto()
		if err != nil {
			return nil, t.wrapErr("/wallet/getassetissuelistbyname", err)
		}

		result.AssetIssue = append(result.AssetIssue, asset)
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
