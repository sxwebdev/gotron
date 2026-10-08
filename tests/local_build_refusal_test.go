package tests

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/client"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

// localChainParamCreateNewAccountFeeInSystemContract is the committee
// parameter of the fee CreateAccount burns from its owner; the private network
// starts it at zero, and no owner can then fall short of it.
const localChainParamCreateNewAccountFeeInSystemContract = 7

// The refusals a caller has to tell from a node that did not answer, asked of
// a real node over both transports: each matches its sentinel through
// errors.Is, keeps its type and code, and matches no other verdict. The
// sentinels' texts in pkg/client are this node's answers.
func TestLocal_RefusalsMatchTheirVerdicts(t *testing.T) {
	requireLocalNode(t)
	chain := localFixture(t)

	// An owner that exists but holds less than the creation fee: the fee is
	// raised to 1 SUN and the owner holds none.
	ready := enableChainParameter(t, localChainParamCreateNewAccountFeeInSystemContract, "getCreateNewAccountFeeInSystemContract")
	ready()
	poor := newLocalKey(t)
	g := newLocalGRPCClient(t)
	ctx := localContext(t, time.Minute)
	ext, err := g.CreateAccount(ctx, localWitnessAddress, poor.Address, core.AccountType_Normal)
	require.NoError(t, err)
	sendAndConfirm(t, g, ext, localWitness(t))

	transfer := `[{"address":"` + chain.Receiver.Address + `"},{"uint256":"1"}]`
	huge := `[{"address":"` + chain.Receiver.Address + `"},{"uint256":"99999999999999999999999999"}]`
	verdicts := []error{client.ErrAccountExists, client.ErrCreateAccountFeeShort, client.ErrContractNotExist}

	for name, c := range map[string]*client.Client{"grpc": g, "http": newLocalHTTPClient(t)} {
		t.Run(name, func(t *testing.T) {
			ctx := localContext(t, time.Minute)
			refused := func(t *testing.T, err error, want error) {
				t.Helper()
				_, ok := errors.AsType[*client.ContractValidateError](err)
				require.True(t, ok, "not a refusal: %v", err)
				for _, v := range verdicts {
					if v == want {
						require.ErrorIs(t, err, v)
					} else {
						require.NotErrorIs(t, err, v)
					}
				}
			}

			_, err := c.CreateAccount(ctx, poor.Address, newLocalKey(t).Address, core.AccountType_Normal)
			refused(t, err, client.ErrCreateAccountFeeShort)
			_, err = c.CreateAccount(ctx, localWitnessAddress, chain.Staker.Address, core.AccountType_Normal)
			refused(t, err, client.ErrAccountExists)
			// An owner the chain has never seen is a refusal of no named verdict.
			_, err = c.CreateAccount(ctx, newLocalKey(t).Address, newLocalKey(t).Address, core.AccountType_Normal)
			refused(t, err, nil)

			_, err = c.TriggerContract(ctx, chain.Staker.Address, chain.Receiver.Address, "transfer(address,uint256)", transfer, 0, 0, "", 0)
			refused(t, err, client.ErrContractNotExist)

			// A constant call and an estimate: to an account and to an address
			// the chain has never seen, no contract; a revert, a failed call.
			for _, to := range []string{chain.Receiver.Address, newLocalKey(t).Address} {
				_, err = c.TriggerConstantContractCustom(ctx, chain.Staker.Address, to, "transfer(address,uint256)", transfer)
				requireCallError(t, err, api.Return_CONTRACT_VALIDATE_ERROR, true)
				// An estimate the node will not run is a refusal of the request.
				_, err = c.EstimateEnergy(ctx, chain.Staker.Address, to, "transfer(address,uint256)", transfer, 0, "", 0)
				refused(t, err, client.ErrContractNotExist)
				require.NotErrorIs(t, err, client.ErrContractCallFailed)
			}
			_, err = c.TriggerConstantContractCustom(ctx, chain.Staker.Address, chain.TRC20, "transfer(address,uint256)", huge)
			requireCallError(t, err, api.Return_SUCCESS, false)
			_, err = c.EstimateEnergy(ctx, chain.Staker.Address, chain.TRC20, "transfer(address,uint256)", huge, 0, "", 0)
			requireCallError(t, err, api.Return_CONTRACT_EXE_ERROR, false)
		})
	}
}

func requireCallError(t *testing.T, err error, code api.ReturnResponseCode, notExist bool) {
	t.Helper()
	cce, ok := errors.AsType[*client.ContractCallError](err)
	require.True(t, ok, "not a call error: %v", err)
	require.Equal(t, code, cce.Code, "%v", err)
	require.ErrorIs(t, err, client.ErrContractCallFailed)
	require.Equal(t, notExist, errors.Is(err, client.ErrContractNotExist), "%v", err)
}
