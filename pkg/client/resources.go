package client

import (
	"github.com/shopspring/decimal"
	"github.com/sxwebdev/gotron/pkg/resources"
)

// DelegateResource delegates bandwidth or energy to another address
func (t *Client) DelegateResource(from, to string, balance decimal.Decimal, resourceType resources.ResourceType, privateKey string, lock bool, lockPeriod int64) (string, error) {
	params := resources.DelegateResourceParams{
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

// UndelegateResource undelegates bandwidth or energy from an address
func (t *Client) UndelegateResource(from, to string, balance decimal.Decimal, resourceType resources.ResourceType, privateKey string) (string, error) {
	params := resources.UndelegateResourceParams{
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
