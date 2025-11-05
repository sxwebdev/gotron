package client

import (
	"fmt"

	"github.com/shopspring/decimal"
)

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

	if err := p.ResourceType.Validate(); err != nil {
		return err
	}

	if p.PrivateKey == "" {
		return fmt.Errorf("%w: private key is required", ErrInvalidParams)
	}

	if p.Lock && p.LockPeriod <= 0 {
		return fmt.Errorf("%w: lock period must be greater than zero when lock is enabled", ErrInvalidParams)
	}

	return nil
}

// DelegateResource delegates bandwidth or energy to another address
func (c *Client) DelegateResource(from, to string, balance decimal.Decimal, resourceType ResourceType, privateKey string, lock bool, lockPeriod int64) (string, error) {
	params := DelegateResourceParams{
		From:         from,
		To:           to,
		Balance:      balance,
		ResourceType: resourceType,
		PrivateKey:   privateKey,
		Lock:         lock,
		LockPeriod:   lockPeriod,
	}

	if err := params.Validate(); err != nil {
		return "", err
	}

	// TODO: Implement actual resource delegation
	return "", nil
}

// ReclaimResourceParams represents parameters for resource undelegation
type ReclaimResourceParams struct {
	From         string
	To           string
	Balance      decimal.Decimal
	ResourceType ResourceType
	PrivateKey   string
}

// Validate validates the undelegation parameters
func (p *ReclaimResourceParams) Validate() error {
	if p.From == "" {
		return fmt.Errorf("%w: from address is required", ErrInvalidAddress)
	}

	if p.To == "" {
		return fmt.Errorf("%w: to address is required", ErrInvalidAddress)
	}

	if p.Balance.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("%w: balance must be greater than zero", ErrInvalidAmount)
	}

	if err := p.ResourceType.Validate(); err != nil {
		return err
	}

	if p.PrivateKey == "" {
		return fmt.Errorf("%w: private key is required", ErrInvalidParams)
	}

	return nil
}

// ReclaimResource undelegates bandwidth or energy from an address
func (c *Client) ReclaimResource(from, to string, balance decimal.Decimal, resourceType ResourceType, privateKey string) (string, error) {
	params := ReclaimResourceParams{
		From:         from,
		To:           to,
		Balance:      balance,
		ResourceType: resourceType,
		PrivateKey:   privateKey,
	}

	if err := params.Validate(); err != nil {
		return "", err
	}

	// TODO: Implement actual resource undelegation
	return "", nil
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
