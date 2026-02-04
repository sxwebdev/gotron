package client

import (
	"context"

	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

// Transport defines the interface for communication with Tron nodes.
// It can be implemented by different protocols (gRPC, HTTP).
type Transport interface {
	// Account operations
	GetAccount(ctx context.Context, account *core.Account) (*core.Account, error)
	GetAccountResource(ctx context.Context, account *core.Account) (*api.AccountResourceMessage, error)
	CreateAccount(ctx context.Context, contract *core.AccountCreateContract) (*api.TransactionExtention, error)

	// Block operations
	GetNowBlock(ctx context.Context) (*api.BlockExtention, error)
	GetBlockByNum(ctx context.Context, num int64) (*api.BlockExtention, error)
	GetBlockById(ctx context.Context, id []byte) (*core.Block, error)
	GetBlockByLimitNext(ctx context.Context, start, end int64) (*api.BlockListExtention, error)
	GetBlockByLatestNum(ctx context.Context, num int64) (*api.BlockListExtention, error)
	GetTransactionInfoByBlockNum(ctx context.Context, num int64) (*api.TransactionInfoList, error)

	// Transaction operations
	GetTransactionById(ctx context.Context, id []byte) (*core.Transaction, error)
	GetTransactionInfoById(ctx context.Context, id []byte) (*core.TransactionInfo, error)
	BroadcastTransaction(ctx context.Context, tx *core.Transaction) (*api.Return, error)
	CreateTransaction(ctx context.Context, contract *core.TransferContract) (*api.TransactionExtention, error)

	// Contract operations
	TriggerContract(ctx context.Context, contract *core.TriggerSmartContract) (*api.TransactionExtention, error)
	TriggerConstantContract(ctx context.Context, contract *core.TriggerSmartContract) (*api.TransactionExtention, error)
	EstimateEnergy(ctx context.Context, contract *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error)
	DeployContract(ctx context.Context, contract *core.CreateSmartContract) (*api.TransactionExtention, error)
	GetContract(ctx context.Context, address []byte) (*core.SmartContract, error)
	UpdateSetting(ctx context.Context, contract *core.UpdateSettingContract) (*api.TransactionExtention, error)
	UpdateEnergyLimit(ctx context.Context, contract *core.UpdateEnergyLimitContract) (*api.TransactionExtention, error)

	// Resource operations
	GetAccountResourceMessage(ctx context.Context, account *core.Account) (*api.AccountResourceMessage, error)
	GetDelegatedResource(ctx context.Context, msg *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error)
	GetDelegatedResourceV2(ctx context.Context, msg *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error)
	GetDelegatedResourceAccountIndex(ctx context.Context, address []byte) (*core.DelegatedResourceAccountIndex, error)
	GetDelegatedResourceAccountIndexV2(ctx context.Context, address []byte) (*core.DelegatedResourceAccountIndex, error)
	GetCanDelegatedMaxSize(ctx context.Context, msg *api.CanDelegatedMaxSizeRequestMessage) (*api.CanDelegatedMaxSizeResponseMessage, error)
	DelegateResource(ctx context.Context, contract *core.DelegateResourceContract) (*api.TransactionExtention, error)
	UnDelegateResource(ctx context.Context, contract *core.UnDelegateResourceContract) (*api.TransactionExtention, error)

	// Network operations
	ListNodes(ctx context.Context) (*api.NodeList, error)
	GetChainParameters(ctx context.Context) (*core.ChainParameters, error)
	GetNextMaintenanceTime(ctx context.Context) (*api.NumberMessage, error)
	TotalTransaction(ctx context.Context) (*api.NumberMessage, error)

	// Connection management
	Close() error
}
