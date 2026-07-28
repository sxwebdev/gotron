package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/sxwebdev/gotron/pkg/units"
)

// estimateTransferFeeLimit is the fee limit used to build the throwaway TRC20
// transaction an estimate is measured on. It is never broadcast.
const estimateTransferFeeLimit = SUN(100 * units.SunPerTRX)

// ResourceUsage is what a transaction needs from the network, before any
// question of who pays for it.
type ResourceUsage struct {
	Bandwidth decimal.Decimal `json:"bandwidth"`
	Energy    decimal.Decimal `json:"energy"`
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
		Energy: units.NewEnergy(
			billableEnergy(usage.Energy, available.StakedEnergy),
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
