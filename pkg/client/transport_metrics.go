package client

import (
	"context"
	"time"

	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

// MetricsTransport wraps a Transport and records metrics for each call.
type MetricsTransport struct {
	transport  Transport
	metrics    MetricsCollector
	blockchain string
}

// NewMetricsTransport creates a new metrics-collecting transport wrapper.
func NewMetricsTransport(transport Transport, metrics MetricsCollector, blockchain string) *MetricsTransport {
	if blockchain == "" {
		blockchain = "tron"
	}
	return &MetricsTransport{
		transport:  transport,
		metrics:    metrics,
		blockchain: blockchain,
	}
}

func (t *MetricsTransport) after(method string, start time.Time, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}
	t.metrics.RecordRequest(t.blockchain, method, status, time.Since(start))
}

// Close closes the underlying transport.
func (t *MetricsTransport) Close() error {
	return t.transport.Close()
}

// Account operations

func (t *MetricsTransport) GetAccount(ctx context.Context, account *core.Account) (*core.Account, error) {
	start := time.Now()
	result, err := t.transport.GetAccount(ctx, account)
	t.after("GetAccount", start, err)
	return result, err
}

func (t *MetricsTransport) GetAccountResource(ctx context.Context, account *core.Account) (*api.AccountResourceMessage, error) {
	start := time.Now()
	result, err := t.transport.GetAccountResource(ctx, account)
	t.after("GetAccountResource", start, err)
	return result, err
}

func (t *MetricsTransport) CreateAccount(ctx context.Context, contract *core.AccountCreateContract) (*api.TransactionExtention, error) {
	start := time.Now()
	result, err := t.transport.CreateAccount(ctx, contract)
	t.after("CreateAccount", start, err)
	return result, err
}

// Block operations

func (t *MetricsTransport) GetNowBlock(ctx context.Context) (*api.BlockExtention, error) {
	start := time.Now()
	result, err := t.transport.GetNowBlock(ctx)
	t.after("GetNowBlock", start, err)
	return result, err
}

func (t *MetricsTransport) GetBlockByNum(ctx context.Context, num int64) (*api.BlockExtention, error) {
	start := time.Now()
	result, err := t.transport.GetBlockByNum(ctx, num)
	t.after("GetBlockByNum", start, err)
	return result, err
}

func (t *MetricsTransport) GetBlockById(ctx context.Context, id []byte) (*core.Block, error) {
	start := time.Now()
	result, err := t.transport.GetBlockById(ctx, id)
	t.after("GetBlockById", start, err)
	return result, err
}

func (t *MetricsTransport) GetBlockByLimitNext(ctx context.Context, startBlock, end int64) (*api.BlockListExtention, error) {
	start := time.Now()
	result, err := t.transport.GetBlockByLimitNext(ctx, startBlock, end)
	t.after("GetBlockByLimitNext", start, err)
	return result, err
}

func (t *MetricsTransport) GetBlockByLatestNum(ctx context.Context, num int64) (*api.BlockListExtention, error) {
	start := time.Now()
	result, err := t.transport.GetBlockByLatestNum(ctx, num)
	t.after("GetBlockByLatestNum", start, err)
	return result, err
}

func (t *MetricsTransport) GetTransactionInfoByBlockNum(ctx context.Context, num int64) (*api.TransactionInfoList, error) {
	start := time.Now()
	result, err := t.transport.GetTransactionInfoByBlockNum(ctx, num)
	t.after("GetTransactionInfoByBlockNum", start, err)
	return result, err
}

// Transaction operations

func (t *MetricsTransport) GetTransactionById(ctx context.Context, id []byte) (*core.Transaction, error) {
	start := time.Now()
	result, err := t.transport.GetTransactionById(ctx, id)
	t.after("GetTransactionById", start, err)
	return result, err
}

func (t *MetricsTransport) GetTransactionInfoById(ctx context.Context, id []byte) (*core.TransactionInfo, error) {
	start := time.Now()
	result, err := t.transport.GetTransactionInfoById(ctx, id)
	t.after("GetTransactionInfoById", start, err)
	return result, err
}

func (t *MetricsTransport) BroadcastTransaction(ctx context.Context, tx *core.Transaction) (*api.Return, error) {
	start := time.Now()
	result, err := t.transport.BroadcastTransaction(ctx, tx)
	t.after("BroadcastTransaction", start, err)
	return result, err
}

func (t *MetricsTransport) CreateTransaction(ctx context.Context, contract *core.TransferContract) (*api.TransactionExtention, error) {
	start := time.Now()
	result, err := t.transport.CreateTransaction(ctx, contract)
	t.after("CreateTransaction", start, err)
	return result, err
}

// Contract operations

func (t *MetricsTransport) TriggerContract(ctx context.Context, contract *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	start := time.Now()
	result, err := t.transport.TriggerContract(ctx, contract)
	t.after("TriggerContract", start, err)
	return result, err
}

func (t *MetricsTransport) TriggerConstantContract(ctx context.Context, contract *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	start := time.Now()
	result, err := t.transport.TriggerConstantContract(ctx, contract)
	t.after("TriggerConstantContract", start, err)
	return result, err
}

func (t *MetricsTransport) EstimateEnergy(ctx context.Context, contract *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
	start := time.Now()
	result, err := t.transport.EstimateEnergy(ctx, contract)
	t.after("EstimateEnergy", start, err)
	return result, err
}

func (t *MetricsTransport) DeployContract(ctx context.Context, contract *core.CreateSmartContract) (*api.TransactionExtention, error) {
	start := time.Now()
	result, err := t.transport.DeployContract(ctx, contract)
	t.after("DeployContract", start, err)
	return result, err
}

func (t *MetricsTransport) GetContract(ctx context.Context, address []byte) (*core.SmartContract, error) {
	start := time.Now()
	result, err := t.transport.GetContract(ctx, address)
	t.after("GetContract", start, err)
	return result, err
}

func (t *MetricsTransport) UpdateSetting(ctx context.Context, contract *core.UpdateSettingContract) (*api.TransactionExtention, error) {
	start := time.Now()
	result, err := t.transport.UpdateSetting(ctx, contract)
	t.after("UpdateSetting", start, err)
	return result, err
}

func (t *MetricsTransport) UpdateEnergyLimit(ctx context.Context, contract *core.UpdateEnergyLimitContract) (*api.TransactionExtention, error) {
	start := time.Now()
	result, err := t.transport.UpdateEnergyLimit(ctx, contract)
	t.after("UpdateEnergyLimit", start, err)
	return result, err
}

// Resource operations

func (t *MetricsTransport) GetAccountResourceMessage(ctx context.Context, account *core.Account) (*api.AccountResourceMessage, error) {
	start := time.Now()
	result, err := t.transport.GetAccountResourceMessage(ctx, account)
	t.after("GetAccountResourceMessage", start, err)
	return result, err
}

func (t *MetricsTransport) GetDelegatedResource(ctx context.Context, msg *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
	start := time.Now()
	result, err := t.transport.GetDelegatedResource(ctx, msg)
	t.after("GetDelegatedResource", start, err)
	return result, err
}

func (t *MetricsTransport) GetDelegatedResourceV2(ctx context.Context, msg *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
	start := time.Now()
	result, err := t.transport.GetDelegatedResourceV2(ctx, msg)
	t.after("GetDelegatedResourceV2", start, err)
	return result, err
}

func (t *MetricsTransport) GetDelegatedResourceAccountIndex(ctx context.Context, address []byte) (*core.DelegatedResourceAccountIndex, error) {
	start := time.Now()
	result, err := t.transport.GetDelegatedResourceAccountIndex(ctx, address)
	t.after("GetDelegatedResourceAccountIndex", start, err)
	return result, err
}

func (t *MetricsTransport) GetDelegatedResourceAccountIndexV2(ctx context.Context, address []byte) (*core.DelegatedResourceAccountIndex, error) {
	start := time.Now()
	result, err := t.transport.GetDelegatedResourceAccountIndexV2(ctx, address)
	t.after("GetDelegatedResourceAccountIndexV2", start, err)
	return result, err
}

func (t *MetricsTransport) GetCanDelegatedMaxSize(ctx context.Context, msg *api.CanDelegatedMaxSizeRequestMessage) (*api.CanDelegatedMaxSizeResponseMessage, error) {
	start := time.Now()
	result, err := t.transport.GetCanDelegatedMaxSize(ctx, msg)
	t.after("GetCanDelegatedMaxSize", start, err)
	return result, err
}

func (t *MetricsTransport) DelegateResource(ctx context.Context, contract *core.DelegateResourceContract) (*api.TransactionExtention, error) {
	start := time.Now()
	result, err := t.transport.DelegateResource(ctx, contract)
	t.after("DelegateResource", start, err)
	return result, err
}

func (t *MetricsTransport) UnDelegateResource(ctx context.Context, contract *core.UnDelegateResourceContract) (*api.TransactionExtention, error) {
	start := time.Now()
	result, err := t.transport.UnDelegateResource(ctx, contract)
	t.after("UnDelegateResource", start, err)
	return result, err
}

// Asset operations

func (t *MetricsTransport) GetAssetIssueById(ctx context.Context, id []byte) (*core.AssetIssueContract, error) {
	start := time.Now()
	result, err := t.transport.GetAssetIssueById(ctx, id)
	t.after("GetAssetIssueById", start, err)
	return result, err
}

func (t *MetricsTransport) GetAssetIssueListByName(ctx context.Context, name []byte) (*api.AssetIssueList, error) {
	start := time.Now()
	result, err := t.transport.GetAssetIssueListByName(ctx, name)
	t.after("GetAssetIssueListByName", start, err)
	return result, err
}

// Network operations

func (t *MetricsTransport) ListNodes(ctx context.Context) (*api.NodeList, error) {
	start := time.Now()
	result, err := t.transport.ListNodes(ctx)
	t.after("ListNodes", start, err)
	return result, err
}

func (t *MetricsTransport) GetChainParameters(ctx context.Context) (*core.ChainParameters, error) {
	start := time.Now()
	result, err := t.transport.GetChainParameters(ctx)
	t.after("GetChainParameters", start, err)
	return result, err
}

func (t *MetricsTransport) GetNextMaintenanceTime(ctx context.Context) (*api.NumberMessage, error) {
	start := time.Now()
	result, err := t.transport.GetNextMaintenanceTime(ctx)
	t.after("GetNextMaintenanceTime", start, err)
	return result, err
}

func (t *MetricsTransport) TotalTransaction(ctx context.Context) (*api.NumberMessage, error) {
	start := time.Now()
	result, err := t.transport.TotalTransaction(ctx)
	t.after("TotalTransaction", start, err)
	return result, err
}
