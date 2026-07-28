package client

import (
	"context"
	"encoding/hex"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/client/abi"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
	"google.golang.org/protobuf/proto"
)

// toyDeployCode returns 10 bytes of runtime code, so a real deployment of it
// costs 200 x 10 energy for the code deposit plus a little execution.
const toyDeployCode = "600a600c600039600a6000f3602a60805260206080f3"

// deployProbe records what a deployment estimate asked the node.
type deployProbe struct {
	built    *core.CreateSmartContract
	simulate *core.TriggerSmartContract
	energy   int64
}

func (p *deployProbe) client(t *testing.T) *Client {
	t.Helper()

	return newTestClient(&fakeTransport{
		deployContract: func(_ context.Context, ct *core.CreateSmartContract) (*api.TransactionExtention, error) {
			p.built = ct
			// A node answers with a transaction carrying the contract it was
			// given, and the estimate reads it back out of there. A fake that
			// returned something unrelated would hide that.
			return deployTxFor(t, ct), nil
		},
		triggerConstantContract: func(_ context.Context, ct *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			p.simulate = ct
			return &api.TransactionExtention{
				Result:     &api.Return{Result: true},
				EnergyUsed: p.energy,
			}, nil
		},
		getAccountResource: func(context.Context, *core.Account) (*api.AccountResourceMessage, error) {
			return &api.AccountResourceMessage{FreeNetLimit: 600}, nil
		},
		getChainParameters: func(context.Context) (*core.ChainParameters, error) {
			return &core.ChainParameters{ChainParameter: []*core.ChainParameters_ChainParameter{
				{Key: "getEnergyFee", Value: 100},
				{Key: "getTransactionFee", Value: 1000},
				{Key: "getCreateAccountFee", Value: 100_000},
				{Key: "getCreateNewAccountFeeInSystemContract", Value: 1_000_000},
			}}, nil
		},
	})
}

func toyDeployRequest() DeployContractRequest {
	return DeployContractRequest{
		From:                       testAddr,
		Name:                       "Toy",
		Bytecode:                   toyDeployCode,
		ConsumeUserResourcePercent: 100,
		OriginEnergyLimit:          10_000_000,
	}
}

// The estimate must price the deployment that DeployContract would send, not a
// reconstruction of it - otherwise it can quote one contract and deploy another.
func TestEstimateDeployContractPricesWhatWouldBeDeployed(t *testing.T) {
	req := toyDeployRequest()
	req.ConstructorParams = `[{"uint256":"1000000"},{"address":"` + testAddr2 + `"}]`

	p := &deployProbe{energy: 2021}
	res, err := p.client(t).EstimateDeployContract(context.Background(), req)
	require.NoError(t, err)

	// The bytecode the node was asked to simulate is byte-for-byte the bytecode
	// the deployment carries, constructor arguments included.
	require.NotNil(t, p.built)
	require.NotNil(t, p.simulate)
	require.Equal(t, p.built.GetNewContract().GetBytecode(), p.simulate.GetData())
	require.Equal(t, p.built.GetOwnerAddress(), p.simulate.GetOwnerAddress())

	// An empty contract address is what makes the node run it as a deployment
	// rather than as a call into address zero.
	require.Empty(t, p.simulate.GetContractAddress())

	// And it is the same thing DeployContract builds from the same request.
	direct, err := req.build()
	require.NoError(t, err)
	require.True(t, proto.Equal(direct, p.built),
		"the estimate and the deployment built different contracts")

	// Constructor arguments really are in there: 22 bytes of code plus two
	// 32-byte words.
	code, err := hex.DecodeString(toyDeployCode)
	require.NoError(t, err)
	require.Len(t, p.simulate.GetData(), len(code)+64)

	params, err := abi.LoadFromJSON(req.ConstructorParams)
	require.NoError(t, err)
	encoded, err := abi.GetPaddedParam(params)
	require.NoError(t, err)
	require.Equal(t, encoded, p.simulate.GetData()[len(code):])

	require.True(t, res.Usage.Energy.Equal(decimal.NewFromInt(2021)))
}

// Bandwidth is measured on the transaction the node built, so the energy has to
// be measured on the same one. Rebuilding the contract from the request instead
// would price two different things whenever the node's answer differs from what
// was asked for - and it is the node's answer that gets broadcast.
func TestEstimateDeployContractPricesTheTransactionNotTheRequest(t *testing.T) {
	nodesCode := []byte{0xde, 0xad, 0xbe, 0xef}

	var simulated *core.TriggerSmartContract
	c := newTestClient(&fakeTransport{
		deployContract: func(_ context.Context, ct *core.CreateSmartContract) (*api.TransactionExtention, error) {
			echoed := proto.CloneOf(ct)
			echoed.NewContract.Bytecode = nodesCode
			return deployTxFor(t, echoed), nil
		},
		triggerConstantContract: func(_ context.Context, ct *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			simulated = ct
			return &api.TransactionExtention{Result: &api.Return{Result: true}, EnergyUsed: 7}, nil
		},
		getAccountResource: func(context.Context, *core.Account) (*api.AccountResourceMessage, error) {
			return &api.AccountResourceMessage{FreeNetLimit: 600}, nil
		},
		getChainParameters: func(context.Context) (*core.ChainParameters, error) {
			return &core.ChainParameters{ChainParameter: []*core.ChainParameters_ChainParameter{
				{Key: "getEnergyFee", Value: 100},
				{Key: "getTransactionFee", Value: 1000},
			}}, nil
		},
	})

	_, err := c.EstimateDeployContract(context.Background(), toyDeployRequest())
	require.NoError(t, err)

	require.NotNil(t, simulated)
	require.Equal(t, nodesCode, simulated.GetData())
}

// The transaction has to be a deployment for any of this to mean anything.
func TestEstimateDeployContractRejectsANonDeployment(t *testing.T) {
	c := newTestClient(&fakeTransport{
		deployContract: func(context.Context, *core.CreateSmartContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Txid:        []byte{0x01},
				Transaction: &core.Transaction{RawData: &core.TransactionRaw{Timestamp: 1}},
			}, nil
		},
	})

	_, err := c.EstimateDeployContract(context.Background(), toyDeployRequest())
	require.ErrorIs(t, err, ErrInvalidTransaction)
}

func TestEstimateDeployContractCharges(t *testing.T) {
	p := &deployProbe{energy: 2021}
	res, err := p.client(t).EstimateDeployContract(context.Background(), toyDeployRequest())
	require.NoError(t, err)

	// The deployer pays the deployment's energy in full. consume_user_resource_percent
	// governs the contract's later calls, not this transaction, so nothing is
	// subsidised however the request sets it.
	require.True(t, res.Usage.ContractEnergy.IsZero())
	require.True(t, res.Usage.SenderEnergy().Equal(res.Usage.Energy))
	require.Equal(t, SUN(2021*100), res.Charges.Energy)

	// A deployment does create an account for the contract, but the chain bills
	// that inside the energy - adding the 1 TRX activation fee would be
	// double-counting.
	require.Zero(t, res.Charges.AccountCreation)
	require.Zero(t, res.Charges.UnstakedCreation)

	// Bandwidth: 600 free covers a transaction this small.
	require.True(t, res.Usage.Bandwidth.IsPositive())
	require.Zero(t, res.Charges.Bandwidth)

	require.Equal(t, res.Charges.Total(), res.Fee)
}

func TestEstimateDeployContractSubsidyIsNotApplied(t *testing.T) {
	// Even a contract that will pay for all of its future calls pays nothing
	// towards its own deployment.
	req := toyDeployRequest()
	req.ConsumeUserResourcePercent = 0

	p := &deployProbe{energy: 2021}
	res, err := p.client(t).EstimateDeployContract(context.Background(), req)
	require.NoError(t, err)

	require.True(t, res.Usage.ContractEnergy.IsZero())
	require.Equal(t, SUN(2021*100), res.Charges.Energy)
}

func TestEstimateDeployContractValidation(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*DeployContractRequest)
		wantErr error
	}{
		{"invalid from", func(r *DeployContractRequest) { r.From = "bad!" }, ErrInvalidAddress},
		{"empty bytecode", func(r *DeployContractRequest) { r.Bytecode = "" }, ErrInvalidParams},
		{"bad constructor params", func(r *DeployContractRequest) { r.ConstructorParams = `[{"uint256":1}]` }, ErrInvalidParams},
		{"zero origin energy limit", func(r *DeployContractRequest) { r.OriginEnergyLimit = 0 }, ErrInvalidParams},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := &deployProbe{energy: 2021}
			req := toyDeployRequest()
			tc.mutate(&req)

			_, err := p.client(t).EstimateDeployContract(context.Background(), req)
			require.ErrorIs(t, err, tc.wantErr)
			// A request the SDK can reject must not reach the node.
			require.Nil(t, p.built)
		})
	}
}

// A constructor that reverts is worth learning about before paying to find out.
func TestEstimateDeployContractSurfacesAFailingConstructor(t *testing.T) {
	c := newTestClient(&fakeTransport{
		deployContract: func(context.Context, *core.CreateSmartContract) (*api.TransactionExtention, error) {
			return deployTx(t, testAddr), nil
		},
		triggerConstantContract: func(context.Context, *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Result:     &api.Return{Result: true, Message: []byte("REVERT opcode executed")},
				EnergyUsed: 17,
			}, nil
		},
	})

	_, err := c.EstimateDeployContract(context.Background(), toyDeployRequest())
	require.ErrorIs(t, err, ErrContractCallFailed)
	require.ErrorContains(t, err, "REVERT")
}

// Regression: sizing a transaction used to strip its Ret and append a throwaway
// signature, corrupting the very transaction the caller was about to sign.
func TestEstimateBandwidthLeavesTheTransactionAlone(t *testing.T) {
	tx := deployTx(t, testAddr).GetTransaction()
	tx.Ret = []*core.Transaction_Result{{Fee: 7}}

	before, err := proto.Marshal(tx)
	require.NoError(t, err)

	c := newTestClient(&fakeTransport{})

	first, err := c.EstimateBandwidth(tx)
	require.NoError(t, err)
	require.True(t, first.IsPositive())

	after, err := proto.Marshal(tx)
	require.NoError(t, err)
	require.Equal(t, before, after, "EstimateBandwidth mutated its argument")

	// The specific damage, named so a regression says what broke.
	require.Len(t, tx.GetRet(), 1)
	require.Empty(t, tx.GetSignature())

	// And it is idempotent, which it cannot be while it appends signatures.
	second, err := c.EstimateBandwidth(tx)
	require.NoError(t, err)
	require.True(t, first.Equal(second), "first %s, second %s", first, second)
}

func TestEstimateBandwidthRejectsNil(t *testing.T) {
	c := newTestClient(&fakeTransport{})
	_, err := c.EstimateBandwidth(nil)
	require.ErrorIs(t, err, ErrInvalidTransaction)
}
