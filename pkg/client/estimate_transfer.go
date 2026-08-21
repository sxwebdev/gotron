package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/pkg/units"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

// estimateTransferFeeLimit is the fee limit used to build the throwaway TRC20
// transaction an estimate is measured on. It is never broadcast.
const estimateTransferFeeLimit = SUN(100 * units.SunPerTRX)

// ResourceUsage is what a transaction needs from the network, before any
// question of who pays for it.
type ResourceUsage struct {
	// Bandwidth is the whole transaction's bandwidth, matching the receipt's
	// net_usage + net_fee.
	Bandwidth decimal.Decimal `json:"bandwidth"`
	// Energy is the whole call's energy, matching the receipt's
	// energy_usage_total. It is not what the sender pays for - see
	// ContractEnergy.
	Energy decimal.Decimal `json:"energy"`
	// ContractEnergy is the part of Energy the contract's owner pays instead of
	// the sender, matching the receipt's origin_energy_usage. It is zero for a
	// TRX transfer, and for a contract whose consume_user_resource_percent is
	// 100 or whose owner has run out of energy.
	ContractEnergy decimal.Decimal `json:"contract_energy"`
}

// SenderEnergy is the energy the sender is actually accountable for: the call's
// total less whatever the contract's owner covers.
func (u ResourceUsage) SenderEnergy() decimal.Decimal {
	if !u.Energy.IsPositive() {
		return decimal.Zero
	}
	if u.ContractEnergy.GreaterThanOrEqual(u.Energy) {
		return decimal.Zero
	}
	return u.Energy.Sub(u.ContractEnergy)
}

// SenderResources is what the sending account can cover a transaction with.
//
// The two bandwidth pools are reported separately rather than summed because
// Tron charges a transaction to one pool or to neither and never splits it: an
// account holding 400 free and 400 staked pays in full for a 500-byte transfer.
// Their sum would therefore answer no question correctly.
type SenderResources struct {
	// FreeBandwidth is what is left of the daily allowance.
	FreeBandwidth decimal.Decimal `json:"free_bandwidth"`
	// StakedBandwidth is what is left of the bandwidth obtained by staking.
	StakedBandwidth decimal.Decimal `json:"staked_bandwidth"`
	// StakedEnergy is what is left of the energy obtained by staking.
	StakedEnergy decimal.Decimal `json:"staked_energy"`
}

// TransferCharges itemises every TRX charge a transfer incurs, so a caller can
// show what was paid for what. A zero field is a charge that does not apply, and
// the reason it does not apply is readable from the estimate's Usage and
// Available.
//
// One charge is deliberately absent: a transaction carrying more than one
// signature burns getMultiSignFee (1 TRX on mainnet). java-tron charges it on
// the signature count alone, not on the permission id, so one key signing under
// an active permission pays nothing while a 2-of-N under the owner permission
// pays it. The count depends on how the caller assembles signatures, which the
// estimate cannot see, so multi-signing callers must add it themselves.
type TransferCharges struct {
	// Bandwidth is burned when neither pool covers the transaction on its own.
	// It then pays for every byte of it, not for the shortfall. It is always
	// zero alongside AccountCreation: creating an account is settled by the flat
	// fee below instead, never by the byte.
	Bandwidth SUN `json:"bandwidth"`
	// Energy is burned for the energy the staked pool does not cover. Energy,
	// unlike bandwidth, is additive, so this really is a shortfall.
	Energy SUN `json:"energy"`
	// AccountCreation is the flat fee for a recipient that does not exist yet
	// (getCreateNewAccountFeeInSystemContract, 1 TRX on mainnet).
	AccountCreation SUN `json:"account_creation"`
	// UnstakedCreation is what the creation costs when the sender's staked
	// bandwidth cannot cover the transaction (getCreateAccountFee, 0.1 TRX on
	// mainnet). It replaces the per-byte charge rather than adding to it, and
	// the free daily allowance does not avert it - creation reads the staked
	// pool alone.
	UnstakedCreation SUN `json:"unstaked_creation"`
}

// Total returns the sum of every charge.
func (c TransferCharges) Total() SUN {
	return c.Bandwidth + c.Energy + c.AccountCreation + c.UnstakedCreation
}

// EstimateTransferResult is what a transfer costs, in three separable parts:
// what the transaction needs, what the sender already has to meet that need, and
// the TRX charges left over.
//
// Every number here is a fact about what will happen. There is deliberately no
// aggregate "if the account had no resources" figure: it is one multiplication
// away from Usage, and reporting it alongside the real cost invited reading the
// hypothetical as the answer.
type EstimateTransferResult struct {
	// Usage is what the transfer transaction consumes.
	Usage ResourceUsage `json:"usage"`
	// Available is what the sender can cover that with.
	Available SenderResources `json:"available"`
	// Charges is every TRX charge, itemised.
	Charges TransferCharges `json:"charges"`
	// Fee is the sum of Charges: what actually leaves the account.
	Fee SUN `json:"fee"`
}

// billableBandwidth returns the bandwidth a transaction is actually charged for.
//
// Tron never splits one transaction's bandwidth across pools. It tries the
// staked pool for the whole transaction, then the free daily allowance for the
// whole transaction, and if neither covers it alone it burns TRX for every byte
// - the pools do not add up, and there is no such thing as paying only for the
// shortfall. Billing "needed minus available" would therefore under-report every
// transaction that overflows both pools, and adding the pools together would
// make an account with 400 free and 400 staked look able to cover 800.
func billableBandwidth(needed, staked, free decimal.Decimal) decimal.Decimal {
	if !needed.IsPositive() {
		return decimal.Zero
	}
	if needed.LessThanOrEqual(staked) || needed.LessThanOrEqual(free) {
		return decimal.Zero
	}
	return needed
}

// createAccountNeedsFee reports whether a transaction that creates an account is
// charged the flat getCreateAccountFee for it.
//
// Account creation does not follow the ordinary bandwidth rules at all.
// java-tron routes such a transaction through consumeForCreateNewAccount, which
// tries the staked pool alone and, if that falls short, charges a flat fee -
// the free daily allowance is never consulted and no per-byte burn ever
// happens. Verified on mainnet across 12 account-creating transfers: every
// sender with NetLimit 0 paid exactly 0.1 TRX of net_fee while sitting on
// hundreds of unused free bandwidth, every sender with staked bandwidth paid
// nothing beyond the 1 TRX, and none paid a per-byte amount.
func createAccountNeedsFee(needed, staked decimal.Decimal) bool {
	return needed.GreaterThan(staked)
}

// contractEnergyShare returns how much of a call's energy the contract's owner
// pays instead of the caller.
//
// A Tron contract carries consume_user_resource_percent, the share of every
// call the *caller* pays; the owner covers the rest out of the energy it staked,
// capped per call by origin_energy_limit. java-tron settles it in
// ReceiptCapsule.payEnergyBill: the owner's share is
// energy_usage_total * (100 - percent) / 100, then capped by the smaller of its
// remaining staked energy and origin_energy_limit, and whatever is left over
// falls back to the caller. Assuming the caller pays everything is how an
// estimate reports a fee for a transfer the chain settles for nothing: verified
// on Nile, tx e3e1633c… moved USDT with energy_usage_total 14650,
// origin_energy_usage 14650, energy_fee 0 and fee 0, where this SDK had
// quoted 1.465 TRX.
//
// The percentage is applied with integer truncation, as java-tron does.
func contractEnergyShare(total decimal.Decimal, consumeUserResourcePercent, originEnergyLimit int64, originAvailable decimal.Decimal) decimal.Decimal {
	if !total.IsPositive() {
		return decimal.Zero
	}

	ownerPercent := 100 - consumeUserResourcePercent
	if ownerPercent <= 0 {
		return decimal.Zero
	}
	if ownerPercent > 100 {
		ownerPercent = 100
	}

	share := total.Mul(decimal.NewFromInt(ownerPercent)).Div(decimal.NewFromInt(100)).Floor()

	limit := decimal.NewFromInt(originEnergyLimit)
	if originAvailable.LessThan(limit) {
		limit = originAvailable
	}
	if !limit.IsPositive() {
		return decimal.Zero
	}
	if share.GreaterThan(limit) {
		return limit
	}

	return share
}

// billableEnergy returns the energy a transaction is actually charged for.
//
// Energy is the opposite of bandwidth: the staked pool covers what it can and
// the contract keeps running on energy bought by burning TRX, so only the
// shortfall is billed. Confirmed on chain - a single transaction routinely
// reports both energy_usage and energy_fee, which never happens for bandwidth.
func billableEnergy(needed, available decimal.Decimal) decimal.Decimal {
	if !needed.IsPositive() {
		return decimal.Zero
	}
	if available.IsNegative() {
		available = decimal.Zero
	}
	if needed.LessThanOrEqual(available) {
		return decimal.Zero
	}
	return needed.Sub(available)
}

// EstimateTRXTransfer estimates the cost of sending TRX.
//
// TRX and TRC20 estimates are separate calls because their amounts are measured
// on different scales: SUN is fixed at 1e6, while a token's scale comes from its
// own decimals. A single entry point would have to take one amount meaning two
// different things depending on the asset.
func (c *Client) EstimateTRXTransfer(ctx context.Context, fromAddress, toAddress string, amount SUN) (*EstimateTransferResult, error) {
	if fromAddress == "" {
		return nil, fmt.Errorf("%w: from address is required", ErrInvalidAddress)
	}

	if toAddress == "" {
		return nil, fmt.Errorf("%w: to address is required", ErrInvalidAddress)
	}

	if amount <= 0 {
		return nil, fmt.Errorf("%w: amount must be greater than zero", ErrInvalidAmount)
	}

	data, err := c.CreateTransferTransaction(ctx, fromAddress, toAddress, amount)
	if err != nil && !strings.Contains(err.Error(), "reset by peer") {
		return nil, fmt.Errorf("transfer: %w", err)
	}

	var usage ResourceUsage
	usage.Bandwidth, err = c.EstimateBandwidth(data.GetTransaction())
	if err != nil {
		return nil, err
	}

	// A TRX transfer to an address that has no account creates that account, and
	// the chain charges for it.
	activated, err := c.IsAccountActivated(ctx, toAddress)
	if err != nil {
		return nil, fmt.Errorf("check recipient activation: %w", err)
	}

	return c.priceTransfer(ctx, fromAddress, usage, !activated)
}

// EstimateTRC20Transfer estimates the cost of sending a TRC20 token.
//
// The amount is in the token's minimal units; build it with
// units.FromTokenDecimal and the decimals TRC20GetDecimals reports.
func (c *Client) EstimateTRC20Transfer(ctx context.Context, fromAddress, toAddress, contractAddress string, amount TokenAmount) (*EstimateTransferResult, error) {
	if fromAddress == "" {
		return nil, fmt.Errorf("%w: from address is required", ErrInvalidAddress)
	}

	if toAddress == "" {
		return nil, fmt.Errorf("%w: to address is required", ErrInvalidAddress)
	}

	if contractAddress == "" {
		return nil, fmt.Errorf("%w: contract address is required", ErrInvalidAddress)
	}

	if !amount.IsPositive() {
		return nil, fmt.Errorf("%w: amount must be greater than zero", ErrInvalidAmount)
	}

	data, err := c.TRC20Send(ctx, fromAddress, toAddress, contractAddress, amount, estimateTransferFeeLimit)
	if err != nil && !strings.Contains(err.Error(), "reset by peer") {
		return nil, fmt.Errorf("cannot make tron transaction: %w", err)
	}

	var usage ResourceUsage
	usage.Bandwidth, err = c.EstimateBandwidth(data.GetTransaction())
	if err != nil {
		return nil, err
	}

	jsonString := fmt.Sprintf(`[{"address":"%s"},{"uint256":"%s"}]`, toAddress, amount)

	data, err = c.TriggerConstantContractCustom(ctx, fromAddress, contractAddress, "transfer(address,uint256)", jsonString)
	if err != nil && !strings.Contains(err.Error(), "reset by peer") {
		return nil, fmt.Errorf("cannot trigger contract: %w", err)
	}

	usage.Energy = decimal.NewFromInt(data.GetEnergyUsed())

	usage.ContractEnergy, err = c.contractEnergySubsidy(ctx, fromAddress, contractAddress, usage.Energy)
	if err != nil {
		return nil, err
	}

	// No activation charge, whatever state the recipient is in. The
	// account-creation fees belong to Tron's system contracts; a contract call
	// that creates an account is billed 25000 energy for it instead (java-tron's
	// NEW_ACCT_CALL), and the constant call above already measured that, because
	// it ran against the real toAddress.
	//
	// So whichever way the recipient is handled, the cost is in usage.Energy
	// and adding a fee on top would double-count it. Measured on mainnet:
	//   - the activator contract at TQuCVz7ZXMwcuT2ERcBYCZzLeNAZofcTgY reports
	//     8227 energy for a target that exists and 33227 for a fresh one, the
	//     25000 difference being the account it creates, and the on-chain
	//     receipt burns no TRX at all;
	//   - a plain TRC20 transfer creates no account whatsoever - mainnet is full
	//     of addresses holding USDT for which getaccount answers {} - and its
	//     energy varies only with the balance slot: 130285 to an address with no
	//     balance against 64285 to one that already holds some, with no
	//     difference between a recipient that has no balance and one that has no
	//     account.
	return c.priceTransfer(ctx, fromAddress, usage, false)
}

// contractEnergySubsidy reads the contract and its owner to work out how much
// of a call's energy the owner absorbs.
//
// It costs two extra RPCs, which is the price of not inventing a fee: the split
// depends on the contract's settings and on the owner's balance right now, and
// neither can be guessed from the call itself.
func (c *Client) contractEnergySubsidy(ctx context.Context, fromAddress, contractAddress string, total decimal.Decimal) (decimal.Decimal, error) {
	if !total.IsPositive() {
		return decimal.Zero, nil
	}

	contract, err := c.GetContract(ctx, contractAddress)
	if err != nil {
		return decimal.Zero, fmt.Errorf("get contract: %w", err)
	}

	origin := contract.GetOriginAddress()
	if len(origin) == 0 {
		return decimal.Zero, nil
	}

	// Calling one's own contract is settled as a self-call: java-tron skips the
	// split entirely and bills the caller for everything.
	originAddress := tronutils.EncodeCheck(origin)
	if originAddress == fromAddress {
		return decimal.Zero, nil
	}

	res, err := c.GetAccountResource(ctx, originAddress)
	if err != nil {
		return decimal.Zero, fmt.Errorf("get contract owner resources: %w", err)
	}

	return contractEnergyShare(
		total,
		contract.GetConsumeUserResourcePercent(),
		contract.GetOriginEnergyLimit(),
		c.AvailableEnergy(res),
	), nil
}

// EstimateDeployContract prices a deployment without broadcasting it.
//
// It takes the same request DeployContract takes, and builds the transaction
// through DeployContract itself, so the estimate cannot end up describing
// different code than gets deployed - constructor arguments included.
//
// This matters more than for other operations: a deployment whose fee limit is
// too low is still accepted, mined and charged for, and only the receipt says
// OUT_OF_ENERGY. The fee is gone and no contract exists, so there is no second
// chance to find out what it costs.
//
// req.ConsumeUserResourcePercent does not enter into it. That field decides who
// pays for the contract's *later* calls - it is what contractEnergySubsidy
// applies to a TRC20 transfer - but the deployment itself is always billed to
// the deployer, so Usage.ContractEnergy is zero here.
//
// The creation charges are zero too: a deployment does create an account for
// the contract, but the chain bills that inside the energy rather than as the
// 1 TRX activation fee that a TRX transfer to a new address pays.
func (c *Client) EstimateDeployContract(ctx context.Context, req DeployContractRequest) (*EstimateTransferResult, error) {
	tx, err := c.DeployContract(ctx, req)
	if err != nil {
		return nil, err
	}

	var usage ResourceUsage
	usage.Bandwidth, err = c.EstimateBandwidth(tx.GetTransaction())
	if err != nil {
		return nil, err
	}

	// Read the deployment back out of the transaction rather than building it a
	// second time from req: the bandwidth above was measured on this exact
	// transaction, and the energy has to be measured on the same one or the two
	// halves of the estimate can describe different code.
	contract, err := deploymentContract(tx.GetTransaction())
	if err != nil {
		return nil, err
	}

	// A TriggerSmartContract with no contract address is how java-tron is asked
	// to run a deployment: it reads Data as creation bytecode and executes the
	// constructor, answering with the CreateSmartContract it built.
	//
	// The energy comes from a constant call rather than from EstimateEnergy
	// because EstimateEnergy needs a node started with vm.estimateEnergy, which
	// the public mainnet nodes are not - and because it is the more accurate of
	// the two: measured against two real Nile deployments, the constant call
	// reproduced receipt.energy_usage_total exactly (1372886 and 1365499) while
	// EstimateEnergy reported roughly 0.4% more.
	probe, err := c.TriggerConstantContract(ctx, &core.TriggerSmartContract{
		OwnerAddress: contract.GetOwnerAddress(),
		Data:         contract.GetNewContract().GetBytecode(),
	})
	if err != nil {
		return nil, fmt.Errorf("simulate deployment: %w", err)
	}

	usage.Energy = decimal.NewFromInt(probe.GetEnergyUsed())

	return c.priceTransfer(ctx, req.From, usage, false)
}

// priceTransfer turns what a transfer consumes into what it costs, by pricing
// the part the sender's own resources do not cover.
//
// createsAccount is supplied by the caller rather than derived from the
// recipient's state, because the recipient's state does not determine it: the
// same address with no account is created by a TRX transfer and left alone by a
// TRC20 one. Only the estimator knows which it is building.
func (c *Client) priceTransfer(ctx context.Context, fromAddress string, usage ResourceUsage, createsAccount bool) (*EstimateTransferResult, error) {
	chainParams, err := c.ChainParams(ctx)
	if err != nil {
		return nil, err
	}

	res, err := c.GetAccountResource(ctx, fromAddress)
	if err != nil {
		return nil, fmt.Errorf("get sender resources: %w", err)
	}

	available := SenderResources{
		FreeBandwidth:   c.AvailableFreeBandwidth(res),
		StakedBandwidth: c.AvailableBandwidthWithoutFree(res),
		StakedEnergy:    c.AvailableEnergy(res),
	}

	charges := TransferCharges{
		// SenderEnergy, not Energy: a contract that pays for its own calls
		// leaves the sender nothing to be billed for.
		Energy: units.NewEnergy(
			billableEnergy(usage.SenderEnergy(), available.StakedEnergy),
		).ToSUN(chainParams.EnergyFee),
	}

	// The two bandwidth rules are mutually exclusive, so exactly one of these
	// branches produces a charge: an account-creating transaction is settled by
	// the flat creation fee and is never billed by the byte, and an ordinary one
	// has no creation fee to pay.
	if createsAccount {
		charges.AccountCreation = SUN(chainParams.CreateNewAccountFeeInSystemContract)
		if createAccountNeedsFee(usage.Bandwidth, available.StakedBandwidth) {
			charges.UnstakedCreation = SUN(chainParams.CreateAccountFee)
		}
	} else {
		charges.Bandwidth = units.NewBandwidth(
			billableBandwidth(usage.Bandwidth, available.StakedBandwidth, available.FreeBandwidth),
		).ToSUN(chainParams.TransactionFee)
	}

	return &EstimateTransferResult{
		Usage:     usage,
		Available: available,
		Charges:   charges,
		Fee:       charges.Total(),
	}, nil
}
