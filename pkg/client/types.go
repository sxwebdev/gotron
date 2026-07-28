package client

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/sxwebdev/gotron/pkg/units"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

// Network represents the Tron network type
type Network string

const (
	// NetworkMainnet represents the Tron mainnet
	NetworkMainnet Network = "mainnet"
	// NetworkShasta represents the Tron testnet (Shasta)
	NetworkShasta Network = "shasta"
	// NetworkNile represents the Tron testnet (Nile)
	NetworkNile Network = "nile"
)

// Validate validates the network type
func (n Network) Validate() error {
	switch n {
	case NetworkMainnet, NetworkShasta, NetworkNile:
		return nil
	default:
		return fmt.Errorf("invalid network: %s", n)
	}
}

// String returns the string representation of the network type
func (n Network) String() string {
	return string(n)
}

// ResourceType represents the type of resource to delegate
type ResourceType int32

const (
	// ResourceTypeBandwidth represents bandwidth resource
	ResourceTypeBandwidth ResourceType = 0
	// ResourceTypeEnergy represents energy resource
	ResourceTypeEnergy ResourceType = 1
)

// Validate validates the resource type
func (r ResourceType) Validate() error {
	if r != ResourceTypeBandwidth && r != ResourceTypeEnergy {
		return fmt.Errorf("%w: must be Bandwidth or Energy", ErrInvalidResourceType)
	}
	return nil
}

// String returns the string representation of the resource type
func (r ResourceType) String() string {
	switch r {
	case ResourceTypeBandwidth:
		return "BANDWIDTH"
	case ResourceTypeEnergy:
		return "ENERGY"
	default:
		return "UNKNOWN"
	}
}

// ToProto converts the resource type to its protobuf representation
func (r ResourceType) ToProto() core.ResourceCode {
	switch r {
	case ResourceTypeBandwidth:
		return core.ResourceCode_BANDWIDTH
	case ResourceTypeEnergy:
		return core.ResourceCode_ENERGY
	default:
		return -1
	}
}

// SUN is an amount of TRX in the chain's native unit: 1 TRX = 1,000,000 SUN.
// Every TRX-denominated value in this package uses it, so the unit is part of
// the signature rather than of a doc comment. Build one with units.FromTRX or
// from a literal, e.g. client.SUN(1_500_000).
type SUN = units.SUN

// TokenAmount is an amount of a TRC20 token in the token's own minimal units.
// Its scale comes from the token's decimals, so it is deliberately a different
// type from SUN. Build one with FromTokenDecimal or FromTokenUnits.
type TokenAmount = units.TokenAmount

// Amount constructors, re-exported from pkg/units so that building an amount
// does not require a second import.
var (
	// FromTRX converts a TRX amount to SUN, rejecting values that cannot be
	// represented exactly: sub-SUN precision or outside int64.
	FromTRX = units.FromTRX
	// MustFromTRX is FromTRX for constants and tests; it panics on bad input.
	MustFromTRX = units.MustFromTRX
	// FromTokenUnits wraps a raw minimal-unit token amount.
	FromTokenUnits = units.FromTokenUnits
	// FromTokenDecimal converts a human-facing token amount using the token's decimals.
	FromTokenDecimal = units.FromTokenDecimal
)

// EstimateResult is the cost of an operation. Energy and Bandwidth are resource
// units; Fee is what those resources cost when they have to be burned.
type EstimateResult struct {
	Energy    decimal.Decimal `json:"energy"`
	Bandwidth decimal.Decimal `json:"bandwidth"`
	Fee       SUN             `json:"fee"`
}

// AvailableResources represents the resources of an account
type AvailableResources struct {
	Energy         decimal.Decimal `json:"energy"`
	Bandwidth      decimal.Decimal `json:"bandwidth"`
	TotalEnergy    decimal.Decimal `json:"total_energy"`
	TotalBandwidth decimal.Decimal `json:"total_bandwidth"`
}

// Vote is a single super representative vote. Count is in TRON POWER units
// (1 TRON POWER = 1 TRX staked), not SUN.
type Vote struct {
	WitnessAddress string `json:"witness_address"`
	Count          int64  `json:"count"`
}

// PendingUnstake is one in-flight unstake entry. Amount becomes withdrawable at
// ExpireTime.
type PendingUnstake struct {
	Resource   ResourceType `json:"resource"`
	Amount     SUN          `json:"amount"`
	ExpireTime time.Time    `json:"expire_time"`
}

// StakeInfo is an aggregated view of an account's Stake 2.0 position.
type StakeInfo struct {
	StakedBandwidth SUN              `json:"staked_bandwidth"`
	StakedEnergy    SUN              `json:"staked_energy"`
	TotalStaked     SUN              `json:"total_staked"`
	UnstakingTotal  SUN              `json:"unstaking_total"`
	WithdrawableNow SUN              `json:"withdrawable_now"`
	PendingUnstakes []PendingUnstake `json:"pending_unstakes"`
}
