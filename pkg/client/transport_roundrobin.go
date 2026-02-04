package client

import (
	"context"
	"sync/atomic"

	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

// RoundRobinTransport implements Transport using round-robin load balancing
// across multiple underlying transports
type RoundRobinTransport struct {
	transports []Transport
	counter    atomic.Uint64
}

// NewRoundRobinTransport creates a new round-robin transport from multiple transports
func NewRoundRobinTransport(transports []Transport) *RoundRobinTransport {
	return &RoundRobinTransport{
		transports: transports,
	}
}

// next returns the next transport using round-robin selection
func (t *RoundRobinTransport) next() Transport {
	idx := t.counter.Add(1) - 1
	return t.transports[idx%uint64(len(t.transports))]
}

// Close closes all underlying transports
func (t *RoundRobinTransport) Close() error {
	var lastErr error
	for _, transport := range t.transports {
		if err := transport.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// Account operations

func (t *RoundRobinTransport) GetAccount(ctx context.Context, account *core.Account) (*core.Account, error) {
	return t.next().GetAccount(ctx, account)
}

func (t *RoundRobinTransport) GetAccountResource(ctx context.Context, account *core.Account) (*api.AccountResourceMessage, error) {
	return t.next().GetAccountResource(ctx, account)
}

func (t *RoundRobinTransport) CreateAccount(ctx context.Context, contract *core.AccountCreateContract) (*api.TransactionExtention, error) {
	return t.next().CreateAccount(ctx, contract)
}

// Block operations

func (t *RoundRobinTransport) GetNowBlock(ctx context.Context) (*api.BlockExtention, error) {
	return t.next().GetNowBlock(ctx)
}

func (t *RoundRobinTransport) GetBlockByNum(ctx context.Context, num int64) (*api.BlockExtention, error) {
	return t.next().GetBlockByNum(ctx, num)
}

func (t *RoundRobinTransport) GetBlockById(ctx context.Context, id []byte) (*core.Block, error) {
	return t.next().GetBlockById(ctx, id)
}

func (t *RoundRobinTransport) GetBlockByLimitNext(ctx context.Context, start, end int64) (*api.BlockListExtention, error) {
	return t.next().GetBlockByLimitNext(ctx, start, end)
}

func (t *RoundRobinTransport) GetBlockByLatestNum(ctx context.Context, num int64) (*api.BlockListExtention, error) {
	return t.next().GetBlockByLatestNum(ctx, num)
}

func (t *RoundRobinTransport) GetTransactionInfoByBlockNum(ctx context.Context, num int64) (*api.TransactionInfoList, error) {
	return t.next().GetTransactionInfoByBlockNum(ctx, num)
}

// Transaction operations

func (t *RoundRobinTransport) GetTransactionById(ctx context.Context, id []byte) (*core.Transaction, error) {
	return t.next().GetTransactionById(ctx, id)
}

func (t *RoundRobinTransport) GetTransactionInfoById(ctx context.Context, id []byte) (*core.TransactionInfo, error) {
	return t.next().GetTransactionInfoById(ctx, id)
}

func (t *RoundRobinTransport) BroadcastTransaction(ctx context.Context, tx *core.Transaction) (*api.Return, error) {
	return t.next().BroadcastTransaction(ctx, tx)
}

func (t *RoundRobinTransport) CreateTransaction(ctx context.Context, contract *core.TransferContract) (*api.TransactionExtention, error) {
	return t.next().CreateTransaction(ctx, contract)
}

// Contract operations

func (t *RoundRobinTransport) TriggerContract(ctx context.Context, contract *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	return t.next().TriggerContract(ctx, contract)
}

func (t *RoundRobinTransport) TriggerConstantContract(ctx context.Context, contract *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	return t.next().TriggerConstantContract(ctx, contract)
}

func (t *RoundRobinTransport) EstimateEnergy(ctx context.Context, contract *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
	return t.next().EstimateEnergy(ctx, contract)
}

func (t *RoundRobinTransport) DeployContract(ctx context.Context, contract *core.CreateSmartContract) (*api.TransactionExtention, error) {
	return t.next().DeployContract(ctx, contract)
}

func (t *RoundRobinTransport) GetContract(ctx context.Context, address []byte) (*core.SmartContract, error) {
	return t.next().GetContract(ctx, address)
}

func (t *RoundRobinTransport) UpdateSetting(ctx context.Context, contract *core.UpdateSettingContract) (*api.TransactionExtention, error) {
	return t.next().UpdateSetting(ctx, contract)
}

func (t *RoundRobinTransport) UpdateEnergyLimit(ctx context.Context, contract *core.UpdateEnergyLimitContract) (*api.TransactionExtention, error) {
	return t.next().UpdateEnergyLimit(ctx, contract)
}

// Resource operations

func (t *RoundRobinTransport) GetAccountResourceMessage(ctx context.Context, account *core.Account) (*api.AccountResourceMessage, error) {
	return t.next().GetAccountResourceMessage(ctx, account)
}

func (t *RoundRobinTransport) GetDelegatedResource(ctx context.Context, msg *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
	return t.next().GetDelegatedResource(ctx, msg)
}

func (t *RoundRobinTransport) GetDelegatedResourceV2(ctx context.Context, msg *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
	return t.next().GetDelegatedResourceV2(ctx, msg)
}

func (t *RoundRobinTransport) GetDelegatedResourceAccountIndex(ctx context.Context, address []byte) (*core.DelegatedResourceAccountIndex, error) {
	return t.next().GetDelegatedResourceAccountIndex(ctx, address)
}

func (t *RoundRobinTransport) GetDelegatedResourceAccountIndexV2(ctx context.Context, address []byte) (*core.DelegatedResourceAccountIndex, error) {
	return t.next().GetDelegatedResourceAccountIndexV2(ctx, address)
}

func (t *RoundRobinTransport) GetCanDelegatedMaxSize(ctx context.Context, msg *api.CanDelegatedMaxSizeRequestMessage) (*api.CanDelegatedMaxSizeResponseMessage, error) {
	return t.next().GetCanDelegatedMaxSize(ctx, msg)
}

func (t *RoundRobinTransport) DelegateResource(ctx context.Context, contract *core.DelegateResourceContract) (*api.TransactionExtention, error) {
	return t.next().DelegateResource(ctx, contract)
}

func (t *RoundRobinTransport) UnDelegateResource(ctx context.Context, contract *core.UnDelegateResourceContract) (*api.TransactionExtention, error) {
	return t.next().UnDelegateResource(ctx, contract)
}

// Asset operations

func (t *RoundRobinTransport) GetAssetIssueById(ctx context.Context, id []byte) (*core.AssetIssueContract, error) {
	return t.next().GetAssetIssueById(ctx, id)
}

func (t *RoundRobinTransport) GetAssetIssueListByName(ctx context.Context, name []byte) (*api.AssetIssueList, error) {
	return t.next().GetAssetIssueListByName(ctx, name)
}

// Network operations

func (t *RoundRobinTransport) ListNodes(ctx context.Context) (*api.NodeList, error) {
	return t.next().ListNodes(ctx)
}

func (t *RoundRobinTransport) GetChainParameters(ctx context.Context) (*core.ChainParameters, error) {
	return t.next().GetChainParameters(ctx)
}

func (t *RoundRobinTransport) GetNextMaintenanceTime(ctx context.Context) (*api.NumberMessage, error) {
	return t.next().GetNextMaintenanceTime(ctx)
}

func (t *RoundRobinTransport) TotalTransaction(ctx context.Context) (*api.NumberMessage, error) {
	return t.next().TotalTransaction(ctx)
}
