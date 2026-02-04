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
		headers:    cfg.HTTPHeaders,
	}, nil
}

// Close closes the HTTP transport (no-op for HTTP)
func (t *HTTPTransport) Close() error {
	return nil
}

// doRequestRaw performs an HTTP POST request and returns raw JSON response
func (t *HTTPTransport) doRequestRaw(ctx context.Context, endpoint string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	} else {
		bodyReader = bytes.NewReader([]byte("{}"))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.baseURL+endpoint, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	for key, value := range t.headers {
		req.Header.Set(key, value)
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
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
	if err := t.doRequest(ctx, "/wallet/getnowblock", nil, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) GetBlockByNum(ctx context.Context, num int64) (*api.BlockExtention, error) {
	reqBody := map[string]interface{}{
		"num": num,
	}

	result := &api.BlockExtention{}
	if err := t.doRequest(ctx, "/wallet/getblockbynum", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) GetBlockById(ctx context.Context, id []byte) (*core.Block, error) {
	reqBody := map[string]interface{}{
		"value": hex.EncodeToString(id),
	}

	result := &core.Block{}
	if err := t.doRequest(ctx, "/wallet/getblockbyid", reqBody, result); err != nil {
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
	if err := t.doRequest(ctx, "/wallet/getblockbylimitnext", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) GetBlockByLatestNum(ctx context.Context, num int64) (*api.BlockListExtention, error) {
	reqBody := map[string]interface{}{
		"num": num,
	}

	result := &api.BlockListExtention{}
	if err := t.doRequest(ctx, "/wallet/getblockbylatestnum", reqBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *HTTPTransport) GetTransactionInfoByBlockNum(ctx context.Context, num int64) (*api.TransactionInfoList, error) {
	reqBody := map[string]interface{}{
		"num": num,
	}

	result := &api.TransactionInfoList{}
	if err := t.doRequest(ctx, "/wallet/gettransactioninfobyblocknum", reqBody, result); err != nil {
		return nil, err
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
