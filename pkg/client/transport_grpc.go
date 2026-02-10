package client

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// defaultMaxSizeOption is the default max message size for gRPC calls
var defaultMaxSizeOption = grpc.MaxCallRecvMsgSize(32 * 10e6)

// GRPCTransport implements Transport using gRPC protocol
type GRPCTransport struct {
	conn         *grpc.ClientConn
	walletClient api.WalletClient
}

// NewGRPCTransport creates a new gRPC transport
func NewGRPCTransport(cfg NodeConfig) (*GRPCTransport, error) {
	opts := append(
		cfg.DialOptions,
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*100)),
	)

	if cfg.UseTLS {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: false,
			MinVersion:         tls.VersionTLS13,
		})))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Build interceptor chain
	var interceptors []grpc.UnaryClientInterceptor

	// Add interceptor to inject headers as gRPC metadata
	if len(cfg.Headers) > 0 {
		interceptors = append(interceptors, headersInterceptor(cfg.Headers))
	}

	// Add interceptor to wrap errors with transport context
	interceptors = append(interceptors, transportErrorInterceptor(cfg.Address))

	opts = append(opts, grpc.WithChainUnaryInterceptor(interceptors...))

	conn, err := grpc.NewClient(cfg.Address, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial gRPC: %w", err)
	}

	return &GRPCTransport{
		conn:         conn,
		walletClient: api.NewWalletClient(conn),
	}, nil
}

// headersInterceptor returns a gRPC unary interceptor that adds headers as metadata
func headersInterceptor(headers map[string]string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		md := metadata.New(headers)
		ctx = metadata.NewOutgoingContext(ctx, md)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// transportErrorInterceptor returns a gRPC unary interceptor that wraps errors with TransportError
func transportErrorInterceptor(address string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		err := invoker(ctx, method, req, reply, cc, opts...)
		if err != nil {
			return &TransportError{
				Host:     address,
				Protocol: "grpc",
				Method:   method,
				Err:      err,
			}
		}
		return nil
	}
}

// WalletClient returns the underlying WalletClient for direct access
func (t *GRPCTransport) WalletClient() api.WalletClient {
	return t.walletClient
}

// Close closes the gRPC connection
func (t *GRPCTransport) Close() error {
	if t.conn != nil {
		return t.conn.Close()
	}
	return nil
}

// Account operations

func (t *GRPCTransport) GetAccount(ctx context.Context, account *core.Account) (*core.Account, error) {
	return t.walletClient.GetAccount(ctx, account)
}

func (t *GRPCTransport) GetAccountResource(ctx context.Context, account *core.Account) (*api.AccountResourceMessage, error) {
	return t.walletClient.GetAccountResource(ctx, account)
}

func (t *GRPCTransport) CreateAccount(ctx context.Context, contract *core.AccountCreateContract) (*api.TransactionExtention, error) {
	return t.walletClient.CreateAccount2(ctx, contract)
}

// Block operations

func (t *GRPCTransport) GetNowBlock(ctx context.Context) (*api.BlockExtention, error) {
	return t.walletClient.GetNowBlock2(ctx, new(api.EmptyMessage))
}

func (t *GRPCTransport) GetBlockByNum(ctx context.Context, num int64) (*api.BlockExtention, error) {
	req := &api.NumberMessage{Num: num}
	return t.walletClient.GetBlockByNum2(ctx, req, defaultMaxSizeOption)
}

func (t *GRPCTransport) GetBlockById(ctx context.Context, id []byte) (*core.Block, error) {
	req := &api.BytesMessage{Value: id}
	return t.walletClient.GetBlockById(ctx, req, defaultMaxSizeOption)
}

func (t *GRPCTransport) GetBlockByLimitNext(ctx context.Context, start, end int64) (*api.BlockListExtention, error) {
	req := &api.BlockLimit{StartNum: start, EndNum: end}
	return t.walletClient.GetBlockByLimitNext2(ctx, req, defaultMaxSizeOption)
}

func (t *GRPCTransport) GetBlockByLatestNum(ctx context.Context, num int64) (*api.BlockListExtention, error) {
	req := &api.NumberMessage{Num: num}
	return t.walletClient.GetBlockByLatestNum2(ctx, req, defaultMaxSizeOption)
}

func (t *GRPCTransport) GetTransactionInfoByBlockNum(ctx context.Context, num int64) (*api.TransactionInfoList, error) {
	req := &api.NumberMessage{Num: num}
	return t.walletClient.GetTransactionInfoByBlockNum(ctx, req, defaultMaxSizeOption)
}

// Transaction operations

func (t *GRPCTransport) GetTransactionById(ctx context.Context, id []byte) (*core.Transaction, error) {
	req := &api.BytesMessage{Value: id}
	return t.walletClient.GetTransactionById(ctx, req)
}

func (t *GRPCTransport) GetTransactionInfoById(ctx context.Context, id []byte) (*core.TransactionInfo, error) {
	req := &api.BytesMessage{Value: id}
	return t.walletClient.GetTransactionInfoById(ctx, req)
}

func (t *GRPCTransport) BroadcastTransaction(ctx context.Context, tx *core.Transaction) (*api.Return, error) {
	return t.walletClient.BroadcastTransaction(ctx, tx)
}

func (t *GRPCTransport) CreateTransaction(ctx context.Context, contract *core.TransferContract) (*api.TransactionExtention, error) {
	return t.walletClient.CreateTransaction2(ctx, contract)
}

// Contract operations

func (t *GRPCTransport) TriggerContract(ctx context.Context, contract *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	return t.walletClient.TriggerContract(ctx, contract)
}

func (t *GRPCTransport) TriggerConstantContract(ctx context.Context, contract *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	return t.walletClient.TriggerConstantContract(ctx, contract)
}

func (t *GRPCTransport) EstimateEnergy(ctx context.Context, contract *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
	return t.walletClient.EstimateEnergy(ctx, contract)
}

func (t *GRPCTransport) DeployContract(ctx context.Context, contract *core.CreateSmartContract) (*api.TransactionExtention, error) {
	return t.walletClient.DeployContract(ctx, contract)
}

func (t *GRPCTransport) GetContract(ctx context.Context, address []byte) (*core.SmartContract, error) {
	req := &api.BytesMessage{Value: address}
	return t.walletClient.GetContract(ctx, req)
}

func (t *GRPCTransport) UpdateSetting(ctx context.Context, contract *core.UpdateSettingContract) (*api.TransactionExtention, error) {
	return t.walletClient.UpdateSetting(ctx, contract)
}

func (t *GRPCTransport) UpdateEnergyLimit(ctx context.Context, contract *core.UpdateEnergyLimitContract) (*api.TransactionExtention, error) {
	return t.walletClient.UpdateEnergyLimit(ctx, contract)
}

// Resource operations

func (t *GRPCTransport) GetAccountResourceMessage(ctx context.Context, account *core.Account) (*api.AccountResourceMessage, error) {
	return t.walletClient.GetAccountResource(ctx, account)
}

func (t *GRPCTransport) GetDelegatedResource(ctx context.Context, msg *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
	return t.walletClient.GetDelegatedResource(ctx, msg)
}

func (t *GRPCTransport) GetDelegatedResourceV2(ctx context.Context, msg *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
	return t.walletClient.GetDelegatedResourceV2(ctx, msg)
}

func (t *GRPCTransport) GetDelegatedResourceAccountIndex(ctx context.Context, address []byte) (*core.DelegatedResourceAccountIndex, error) {
	req := &api.BytesMessage{Value: address}
	return t.walletClient.GetDelegatedResourceAccountIndex(ctx, req)
}

func (t *GRPCTransport) GetDelegatedResourceAccountIndexV2(ctx context.Context, address []byte) (*core.DelegatedResourceAccountIndex, error) {
	req := &api.BytesMessage{Value: address}
	return t.walletClient.GetDelegatedResourceAccountIndexV2(ctx, req)
}

func (t *GRPCTransport) GetCanDelegatedMaxSize(ctx context.Context, msg *api.CanDelegatedMaxSizeRequestMessage) (*api.CanDelegatedMaxSizeResponseMessage, error) {
	return t.walletClient.GetCanDelegatedMaxSize(ctx, msg)
}

func (t *GRPCTransport) DelegateResource(ctx context.Context, contract *core.DelegateResourceContract) (*api.TransactionExtention, error) {
	return t.walletClient.DelegateResource(ctx, contract)
}

func (t *GRPCTransport) UnDelegateResource(ctx context.Context, contract *core.UnDelegateResourceContract) (*api.TransactionExtention, error) {
	return t.walletClient.UnDelegateResource(ctx, contract)
}

// Asset operations

func (t *GRPCTransport) GetAssetIssueById(ctx context.Context, id []byte) (*core.AssetIssueContract, error) {
	req := &api.BytesMessage{Value: id}
	return t.walletClient.GetAssetIssueById(ctx, req)
}

func (t *GRPCTransport) GetAssetIssueListByName(ctx context.Context, name []byte) (*api.AssetIssueList, error) {
	req := &api.BytesMessage{Value: name}
	return t.walletClient.GetAssetIssueListByName(ctx, req)
}

// Network operations

func (t *GRPCTransport) ListNodes(ctx context.Context) (*api.NodeList, error) {
	return t.walletClient.ListNodes(ctx, new(api.EmptyMessage))
}

func (t *GRPCTransport) GetChainParameters(ctx context.Context) (*core.ChainParameters, error) {
	return t.walletClient.GetChainParameters(ctx, new(api.EmptyMessage))
}

func (t *GRPCTransport) GetNextMaintenanceTime(ctx context.Context) (*api.NumberMessage, error) {
	return t.walletClient.GetNextMaintenanceTime(ctx, new(api.EmptyMessage))
}

func (t *GRPCTransport) TotalTransaction(ctx context.Context) (*api.NumberMessage, error) {
	return t.walletClient.TotalTransaction(ctx, new(api.EmptyMessage))
}
