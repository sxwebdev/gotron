package client

import (
	"context"

	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

// fakeTransport is an in-memory Transport for deterministic, network-free tests.
// Set the hook for the method under test; unset methods return zero values.
type fakeTransport struct {
	getAccount              func(ctx context.Context, account *core.Account) (*core.Account, error)
	getAccountResource      func(ctx context.Context, account *core.Account) (*api.AccountResourceMessage, error)
	getAccountResourceMsg   func(ctx context.Context, account *core.Account) (*api.AccountResourceMessage, error)
	createAccount           func(ctx context.Context, c *core.AccountCreateContract) (*api.TransactionExtention, error)
	accountPermissionUpdate func(ctx context.Context, c *core.AccountPermissionUpdateContract) (*api.TransactionExtention, error)
	createTransaction       func(ctx context.Context, c *core.TransferContract) (*api.TransactionExtention, error)
	triggerContract         func(ctx context.Context, c *core.TriggerSmartContract) (*api.TransactionExtention, error)
	triggerConstantContract func(ctx context.Context, c *core.TriggerSmartContract) (*api.TransactionExtention, error)
	estimateEnergy          func(ctx context.Context, c *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error)
	getChainParameters      func(ctx context.Context) (*core.ChainParameters, error)
	broadcastTransaction    func(ctx context.Context, tx *core.Transaction) (*api.Return, error)
	getContract             func(ctx context.Context, address []byte) (*core.SmartContract, error)
	deployContract          func(ctx context.Context, c *core.CreateSmartContract) (*api.TransactionExtention, error)

	getDelegatedResource               func(ctx context.Context, m *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error)
	getDelegatedResourceV2             func(ctx context.Context, m *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error)
	getDelegatedResourceAccountIndex   func(ctx context.Context, address []byte) (*core.DelegatedResourceAccountIndex, error)
	getDelegatedResourceAccountIndexV2 func(ctx context.Context, address []byte) (*core.DelegatedResourceAccountIndex, error)
	getCanDelegatedMaxSize             func(ctx context.Context, m *api.CanDelegatedMaxSizeRequestMessage) (*api.CanDelegatedMaxSizeResponseMessage, error)
	getNowBlock                        func(ctx context.Context) (*api.BlockExtention, error)
	getBlockByNum                      func(ctx context.Context, num int64) (*api.BlockExtention, error)
	getTransactionById                 func(ctx context.Context, id []byte) (*core.Transaction, error)
	getTransactionInfoById             func(ctx context.Context, id []byte) (*core.TransactionInfo, error)
	delegateResource                   func(ctx context.Context, c *core.DelegateResourceContract) (*api.TransactionExtention, error)
	unDelegateResource                 func(ctx context.Context, c *core.UnDelegateResourceContract) (*api.TransactionExtention, error)

	freezeBalanceV2              func(ctx context.Context, c *core.FreezeBalanceV2Contract) (*api.TransactionExtention, error)
	unfreezeBalanceV2            func(ctx context.Context, c *core.UnfreezeBalanceV2Contract) (*api.TransactionExtention, error)
	withdrawExpireUnfreeze       func(ctx context.Context, c *core.WithdrawExpireUnfreezeContract) (*api.TransactionExtention, error)
	cancelAllUnfreezeV2          func(ctx context.Context, c *core.CancelAllUnfreezeV2Contract) (*api.TransactionExtention, error)
	getAvailableUnfreezeCount    func(ctx context.Context, m *api.GetAvailableUnfreezeCountRequestMessage) (*api.GetAvailableUnfreezeCountResponseMessage, error)
	getCanWithdrawUnfreezeAmount func(ctx context.Context, m *api.CanWithdrawUnfreezeAmountRequestMessage) (*api.CanWithdrawUnfreezeAmountResponseMessage, error)

	voteWitnessAccount func(ctx context.Context, c *core.VoteWitnessContract) (*api.TransactionExtention, error)
	withdrawBalance    func(ctx context.Context, c *core.WithdrawBalanceContract) (*api.TransactionExtention, error)
	listWitnesses      func(ctx context.Context) (*api.WitnessList, error)
	getRewardInfo      func(ctx context.Context, address []byte) (*api.NumberMessage, error)
	getBrokerageInfo   func(ctx context.Context, address []byte) (*api.NumberMessage, error)

	closeFn func() error

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

func (f *fakeTransport) AccountPermissionUpdate(ctx context.Context, c *core.AccountPermissionUpdateContract) (*api.TransactionExtention, error) {
	if f.accountPermissionUpdate != nil {
		return f.accountPermissionUpdate(ctx, c)
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

func (f *fakeTransport) DeployContract(ctx context.Context, c *core.CreateSmartContract) (*api.TransactionExtention, error) {
	if f.deployContract != nil {
		return f.deployContract(ctx, c)
	}
	return nil, nil
}

func (f *fakeTransport) UpdateSetting(context.Context, *core.UpdateSettingContract) (*api.TransactionExtention, error) {
	return nil, nil
}

func (f *fakeTransport) UpdateEnergyLimit(context.Context, *core.UpdateEnergyLimitContract) (*api.TransactionExtention, error) {
	return nil, nil
}

func (f *fakeTransport) GetDelegatedResource(ctx context.Context, m *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
	if f.getDelegatedResource != nil {
		return f.getDelegatedResource(ctx, m)
	}
	return nil, nil
}

func (f *fakeTransport) GetDelegatedResourceV2(ctx context.Context, m *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
	if f.getDelegatedResourceV2 != nil {
		return f.getDelegatedResourceV2(ctx, m)
	}
	return nil, nil
}

func (f *fakeTransport) GetDelegatedResourceAccountIndex(ctx context.Context, address []byte) (*core.DelegatedResourceAccountIndex, error) {
	if f.getDelegatedResourceAccountIndex != nil {
		return f.getDelegatedResourceAccountIndex(ctx, address)
	}
	return nil, nil
}

func (f *fakeTransport) GetDelegatedResourceAccountIndexV2(ctx context.Context, address []byte) (*core.DelegatedResourceAccountIndex, error) {
	if f.getDelegatedResourceAccountIndexV2 != nil {
		return f.getDelegatedResourceAccountIndexV2(ctx, address)
	}
	return nil, nil
}

func (f *fakeTransport) GetCanDelegatedMaxSize(ctx context.Context, m *api.CanDelegatedMaxSizeRequestMessage) (*api.CanDelegatedMaxSizeResponseMessage, error) {
	if f.getCanDelegatedMaxSize != nil {
		return f.getCanDelegatedMaxSize(ctx, m)
	}
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

func (f *fakeTransport) FreezeBalanceV2(ctx context.Context, c *core.FreezeBalanceV2Contract) (*api.TransactionExtention, error) {
	if f.freezeBalanceV2 != nil {
		return f.freezeBalanceV2(ctx, c)
	}
	return nil, nil
}

func (f *fakeTransport) UnfreezeBalanceV2(ctx context.Context, c *core.UnfreezeBalanceV2Contract) (*api.TransactionExtention, error) {
	if f.unfreezeBalanceV2 != nil {
		return f.unfreezeBalanceV2(ctx, c)
	}
	return nil, nil
}

func (f *fakeTransport) WithdrawExpireUnfreeze(ctx context.Context, c *core.WithdrawExpireUnfreezeContract) (*api.TransactionExtention, error) {
	if f.withdrawExpireUnfreeze != nil {
		return f.withdrawExpireUnfreeze(ctx, c)
	}
	return nil, nil
}

func (f *fakeTransport) CancelAllUnfreezeV2(ctx context.Context, c *core.CancelAllUnfreezeV2Contract) (*api.TransactionExtention, error) {
	if f.cancelAllUnfreezeV2 != nil {
		return f.cancelAllUnfreezeV2(ctx, c)
	}
	return nil, nil
}

func (f *fakeTransport) GetAvailableUnfreezeCount(ctx context.Context, m *api.GetAvailableUnfreezeCountRequestMessage) (*api.GetAvailableUnfreezeCountResponseMessage, error) {
	if f.getAvailableUnfreezeCount != nil {
		return f.getAvailableUnfreezeCount(ctx, m)
	}
	return nil, nil
}

func (f *fakeTransport) GetCanWithdrawUnfreezeAmount(ctx context.Context, m *api.CanWithdrawUnfreezeAmountRequestMessage) (*api.CanWithdrawUnfreezeAmountResponseMessage, error) {
	if f.getCanWithdrawUnfreezeAmount != nil {
		return f.getCanWithdrawUnfreezeAmount(ctx, m)
	}
	return nil, nil
}

func (f *fakeTransport) VoteWitnessAccount(ctx context.Context, c *core.VoteWitnessContract) (*api.TransactionExtention, error) {
	if f.voteWitnessAccount != nil {
		return f.voteWitnessAccount(ctx, c)
	}
	return nil, nil
}

func (f *fakeTransport) WithdrawBalance(ctx context.Context, c *core.WithdrawBalanceContract) (*api.TransactionExtention, error) {
	if f.withdrawBalance != nil {
		return f.withdrawBalance(ctx, c)
	}
	return nil, nil
}

func (f *fakeTransport) ListWitnesses(ctx context.Context) (*api.WitnessList, error) {
	if f.listWitnesses != nil {
		return f.listWitnesses(ctx)
	}
	return nil, nil
}

func (f *fakeTransport) GetRewardInfo(ctx context.Context, address []byte) (*api.NumberMessage, error) {
	if f.getRewardInfo != nil {
		return f.getRewardInfo(ctx, address)
	}
	return nil, nil
}

func (f *fakeTransport) GetBrokerageInfo(ctx context.Context, address []byte) (*api.NumberMessage, error) {
	if f.getBrokerageInfo != nil {
		return f.getBrokerageInfo(ctx, address)
	}
	return nil, nil
}

func (f *fakeTransport) GetAssetIssueById(context.Context, []byte) (*core.AssetIssueContract, error) {
	return nil, nil
}

func (f *fakeTransport) GetAssetIssueListByName(context.Context, []byte) (*api.AssetIssueList, error) {
	return nil, nil
}
func (f *fakeTransport) ListNodes(context.Context) (*api.NodeList, error)    { return nil, nil }
func (f *fakeTransport) GetNodeInfo(context.Context) (*core.NodeInfo, error) { return nil, nil }

func (f *fakeTransport) GetNextMaintenanceTime(context.Context) (*api.NumberMessage, error) {
	return nil, nil
}

func (f *fakeTransport) TotalTransaction(context.Context) (*api.NumberMessage, error) {
	return nil, nil
}

// newTestClient builds a Client backed by the fake transport (white-box).
func newTestClient(ft *fakeTransport) *Client {
	return &Client{transport: ft, config: Config{}}
}

// okTx is a non-empty TransactionExtention, as a healthy node would return.
// An empty one is rejected by checkTransaction, so fakes must not use it for
// success paths.
func okTx() *api.TransactionExtention {
	return &api.TransactionExtention{
		Txid:        []byte{0x01},
		Transaction: &core.Transaction{RawData: &core.TransactionRaw{Timestamp: 1}},
	}
}
