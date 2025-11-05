package client

import "errors"

var (
	ErrInvalidConfig  = errors.New("invalid client configuration")
	ErrNotConnected   = errors.New("client not connected")
	ErrInvalidAddress = errors.New("invalid address")
	ErrInvalidAmount  = errors.New("invalid amount")
	ErrEmptyAddress   = errors.New("address is empty")
)
