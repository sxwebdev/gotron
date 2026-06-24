package client

import (
	"context"

	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

// fakeTransport is an in-memory Transport for deterministic, network-free tests.
// Set the hook for the method under test; unset methods return zero values.
type fakeTransport struct {
	getAccount               func(ctx context.Context, account *core.Account) (*core.Account, error)
	getAccountResource       func(ctx context.Context, account *core.Account) (*api.AccountResourceMessage, error)
	getAccountResourceMsg    func(ctx context.Context, account *core.Account) (*api.AccountResourceMessage, error)
	createAccount            func(ctx context.Context, c *core.AccountCreateContract) (*api.TransactionExtention, error)
	createTransaction        func(ctx context.Context, c *core.TransferContract) (*api.TransactionExtention, error)
	triggerContract          func(ctx context.Context, c *core.TriggerSmartContract) (*api.TransactionExtention, error)
	triggerConstantContract  func(ctx context.Context, c *core.TriggerSmartContract) (*api.TransactionExtention, error)
	estimateEnergy           func(ctx context.Context, c *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error)
	getChainParameters       func(ctx context.Context) (*core.ChainParameters, error)
	broadcastTransaction     func(ctx context.Context, tx *core.Transaction) (*api.Return, error)
	getContract              func(ctx context.Context, address []byte) (*core.SmartContract, error)
	getNowBlock              func(ctx context.Context) (*api.BlockExtention, error)
	getBlockByNum            func(ctx context.Context, num int64) (*api.BlockExtention, error)
	getTransactionById       func(ctx context.Context, id []byte) (*core.Transaction, error)
	getTransactionInfoById   func(ctx context.Context, id []byte) (*core.TransactionInfo, error)
	delegateResource         func(ctx context.Context, c *core.DelegateResourceContract) (*api.TransactionExtention, error)
	unDelegateResource       func(ctx context.Context, c *core.UnDelegateResourceContract) (*api.TransactionExtention, error)
	closeFn                  func() error

	closeCalls int
}

func (f *fakeTransport) GetAccount(ctx context.Context, account *core.Account) (*core.Account, error) {
	if f.getAccount != nil {
		return f.getAccount(ctx, account)
	}
	return nil, nil
}

func (f *fakeTransport) GetAccountResource(ctx context.Context, account *core.Account) (*api.AccountResourceMessage, error) {
	if f.getAccountResource != nil {
		return f.getAccountResource(ctx, account)
	}
	return nil, nil
}

func (f *fakeTransport) GetAccountResourceMessage(ctx context.Context, account *core.Account) (*api.AccountResourceMessage, error) {
	if f.getAccountResourceMsg != nil {
		return f.getAccountResourceMsg(ctx, account)
	}
	return nil, nil
}

func (f *fakeTransport) CreateAccount(ctx context.Context, c *core.AccountCreateContract) (*api.TransactionExtention, error) {
	if f.createAccount != nil {
		return f.createAccount(ctx, c)
	}
	return nil, nil
}

func (f *fakeTransport) CreateTransaction(ctx context.Context, c *core.TransferContract) (*api.TransactionExtention, error) {
	if f.createTransaction != nil {
		return f.createTransaction(ctx, c)
	}
	return nil, nil
}

func (f *fakeTransport) TriggerContract(ctx context.Context, c *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	if f.triggerContract != nil {
		return f.triggerContract(ctx, c)
	}
	return nil, nil
}

func (f *fakeTransport) TriggerConstantContract(ctx context.Context, c *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	if f.triggerConstantContract != nil {
		return f.triggerConstantContract(ctx, c)
	}
	return nil, nil
}

func (f *fakeTransport) EstimateEnergy(ctx context.Context, c *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
	if f.estimateEnergy != nil {
		return f.estimateEnergy(ctx, c)
	}
	return nil, nil
}

func (f *fakeTransport) GetChainParameters(ctx context.Context) (*core.ChainParameters, error) {
	if f.getChainParameters != nil {
		return f.getChainParameters(ctx)
	}
	return nil, nil
}

func (f *fakeTransport) BroadcastTransaction(ctx context.Context, tx *core.Transaction) (*api.Return, error) {
	if f.broadcastTransaction != nil {
		return f.broadcastTransaction(ctx, tx)
	}
	return nil, nil
}

func (f *fakeTransport) GetContract(ctx context.Context, address []byte) (*core.SmartContract, error) {
	if f.getContract != nil {
		return f.getContract(ctx, address)
	}
	return nil, nil
}

func (f *fakeTransport) Close() error {
	f.closeCalls++
	if f.closeFn != nil {
		return f.closeFn()
	}
	return nil
}

// --- remaining Transport methods: unused stubs ---

func (f *fakeTransport) GetNowBlock(ctx context.Context) (*api.BlockExtention, error) {
	if f.getNowBlock != nil {
		return f.getNowBlock(ctx)
	}
	return nil, nil
}
func (f *fakeTransport) GetBlockByNum(ctx context.Context, num int64) (*api.BlockExtention, error) {
	if f.getBlockByNum != nil {
		return f.getBlockByNum(ctx, num)
	}
	return nil, nil
}
func (f *fakeTransport) GetBlockById(context.Context, []byte) (*core.Block, error) { return nil, nil }
func (f *fakeTransport) GetBlockByLimitNext(context.Context, int64, int64) (*api.BlockListExtention, error) {
	return nil, nil
}
func (f *fakeTransport) GetBlockByLatestNum(context.Context, int64) (*api.BlockListExtention, error) {
	return nil, nil
}
func (f *fakeTransport) GetTransactionInfoByBlockNum(context.Context, int64) (*api.TransactionInfoList, error) {
	return nil, nil
}
func (f *fakeTransport) GetTransactionById(ctx context.Context, id []byte) (*core.Transaction, error) {
	if f.getTransactionById != nil {
		return f.getTransactionById(ctx, id)
	}
	return nil, nil
}
func (f *fakeTransport) GetTransactionInfoById(ctx context.Context, id []byte) (*core.TransactionInfo, error) {
	if f.getTransactionInfoById != nil {
		return f.getTransactionInfoById(ctx, id)
	}
	return nil, nil
}
func (f *fakeTransport) DeployContract(context.Context, *core.CreateSmartContract) (*api.TransactionExtention, error) {
	return nil, nil
}
func (f *fakeTransport) UpdateSetting(context.Context, *core.UpdateSettingContract) (*api.TransactionExtention, error) {
	return nil, nil
}
func (f *fakeTransport) UpdateEnergyLimit(context.Context, *core.UpdateEnergyLimitContract) (*api.TransactionExtention, error) {
	return nil, nil
}
func (f *fakeTransport) GetDelegatedResource(context.Context, *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
	return nil, nil
}
func (f *fakeTransport) GetDelegatedResourceV2(context.Context, *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
	return nil, nil
}
func (f *fakeTransport) GetDelegatedResourceAccountIndex(context.Context, []byte) (*core.DelegatedResourceAccountIndex, error) {
	return nil, nil
}
func (f *fakeTransport) GetDelegatedResourceAccountIndexV2(context.Context, []byte) (*core.DelegatedResourceAccountIndex, error) {
	return nil, nil
}
func (f *fakeTransport) GetCanDelegatedMaxSize(context.Context, *api.CanDelegatedMaxSizeRequestMessage) (*api.CanDelegatedMaxSizeResponseMessage, error) {
	return nil, nil
}
func (f *fakeTransport) DelegateResource(ctx context.Context, c *core.DelegateResourceContract) (*api.TransactionExtention, error) {
	if f.delegateResource != nil {
		return f.delegateResource(ctx, c)
	}
	return nil, nil
}
func (f *fakeTransport) UnDelegateResource(ctx context.Context, c *core.UnDelegateResourceContract) (*api.TransactionExtention, error) {
	if f.unDelegateResource != nil {
		return f.unDelegateResource(ctx, c)
	}
	return nil, nil
}
func (f *fakeTransport) GetAssetIssueById(context.Context, []byte) (*core.AssetIssueContract, error) {
	return nil, nil
}
func (f *fakeTransport) GetAssetIssueListByName(context.Context, []byte) (*api.AssetIssueList, error) {
	return nil, nil
}
func (f *fakeTransport) ListNodes(context.Context) (*api.NodeList, error)        { return nil, nil }
func (f *fakeTransport) GetNodeInfo(context.Context) (*core.NodeInfo, error)      { return nil, nil }
func (f *fakeTransport) GetNextMaintenanceTime(context.Context) (*api.NumberMessage, error) {
	return nil, nil
}
func (f *fakeTransport) TotalTransaction(context.Context) (*api.NumberMessage, error) { return nil, nil }

// newTestClient builds a Client backed by the fake transport (white-box).
func newTestClient(ft *fakeTransport) *Client {
	return &Client{transport: ft, config: Config{}}
}
