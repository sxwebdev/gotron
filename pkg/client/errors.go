package client

import "errors"

var (
	// Common errors
	ErrInvalidConfig = errors.New("invalid client configuration")
	ErrNotConnected  = errors.New("client not connected")
	ErrInvalidParams = errors.New("invalid parameters")

	// Address errors
	ErrInvalidAddress = errors.New("invalid address")
	ErrEmptyAddress   = errors.New("address is empty")

	// Transaction errors
	ErrInvalidAmount      = errors.New("invalid amount")
	ErrInvalidTransaction = errors.New("invalid transaction")
	ErrInvalidPrivateKey  = errors.New("invalid private key")

	// Resources errors
	ErrInvalidResourceType = errors.New("invalid resource type")
)
