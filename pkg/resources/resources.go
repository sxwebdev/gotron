// Package resources provides functionality for Tron resource management (delegation/undelegation).
package resources

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
)

var (
	// ErrInvalidParams is returned when parameters are invalid
	ErrInvalidParams = errors.New("invalid parameters")
	// ErrInvalidAddress is returned when an address is invalid
	ErrInvalidAddress = errors.New("invalid address")
	// ErrInvalidAmount is returned when an amount is invalid
	ErrInvalidAmount = errors.New("invalid amount")
	// ErrInvalidResourceType is returned when the resource type is invalid
	ErrInvalidResourceType = errors.New("invalid resource type")
)

// ResourceType represents the type of resource to delegate
type ResourceType int32

const (
	// Bandwidth represents bandwidth resource
	Bandwidth ResourceType = 0
	// Energy represents energy resource
	Energy ResourceType = 1
)

// String returns the string representation of the resource type
func (r ResourceType) String() string {
	switch r {
	case Bandwidth:
		return "BANDWIDTH"
	case Energy:
		return "ENERGY"
	default:
		return "UNKNOWN"
	}
}

// DelegateResourceParams represents parameters for resource delegation
type DelegateResourceParams struct {
	From         string
	To           string
	Balance      decimal.Decimal
	ResourceType ResourceType
	PrivateKey   string
	Lock         bool
	LockPeriod   int64 // Lock period in seconds (0 for no lock)
}

// Validate validates the delegation parameters
func (p *DelegateResourceParams) Validate() error {
	if p.From == "" {
		return fmt.Errorf("%w: from address is required", ErrInvalidAddress)
	}

	if p.To == "" {
		return fmt.Errorf("%w: to address is required", ErrInvalidAddress)
	}

	if p.Balance.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("%w: balance must be greater than zero", ErrInvalidAmount)
	}

	if p.ResourceType != Bandwidth && p.ResourceType != Energy {
		return fmt.Errorf("%w: must be Bandwidth or Energy", ErrInvalidResourceType)
	}

	if p.PrivateKey == "" {
		return fmt.Errorf("%w: private key is required", ErrInvalidParams)
	}

	if p.Lock && p.LockPeriod <= 0 {
		return fmt.Errorf("%w: lock period must be greater than zero when lock is enabled", ErrInvalidParams)
	}

	return nil
}

// UndelegateResourceParams represents parameters for resource undelegation
type UndelegateResourceParams struct {
	From         string
	To           string
	Balance      decimal.Decimal
	ResourceType ResourceType
	PrivateKey   string
}

// Validate validates the undelegation parameters
func (p *UndelegateResourceParams) Validate() error {
	if p.From == "" {
		return fmt.Errorf("%w: from address is required", ErrInvalidAddress)
	}

	if p.To == "" {
		return fmt.Errorf("%w: to address is required", ErrInvalidAddress)
	}

	if p.Balance.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("%w: balance must be greater than zero", ErrInvalidAmount)
	}

	if p.ResourceType != Bandwidth && p.ResourceType != Energy {
		return fmt.Errorf("%w: must be Bandwidth or Energy", ErrInvalidResourceType)
	}

	if p.PrivateKey == "" {
		return fmt.Errorf("%w: private key is required", ErrInvalidParams)
	}

	return nil
}

// AccountResources represents account resource information
type AccountResources struct {
	EnergyLimit    int64
	EnergyUsed     int64
	NetLimit       int64
	NetUsed        int64
	FreeNetLimit   int64
	FreeNetUsed    int64
	TotalNetLimit  int64
	TotalNetWeight int64
}
