// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package trc20

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// TRC20MetaData contains all meta data concerning the TRC20 contract.
var TRC20MetaData = &bind.MetaData{
	ABI: "[{\"constant\":true,\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_upgradedAddress\",\"type\":\"address\"}],\"name\":\"deprecate\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_spender\",\"type\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"deprecated\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_evilUser\",\"type\":\"address\"}],\"name\":\"addBlackList\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_from\",\"type\":\"address\"},{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"upgradedAddress\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"balances\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"name\":\"\",\"type\":\"uint8\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"maximumFee\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"unpause\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_maker\",\"type\":\"address\"}],\"name\":\"getBlackListStatus\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"},{\"name\":\"\",\"type\":\"address\"}],\"name\":\"allowed\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"paused\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"who\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"pause\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"getOwner\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"newBasisPoints\",\"type\":\"uint256\"},{\"name\":\"newMaxFee\",\"type\":\"uint256\"}],\"name\":\"setParams\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"issue\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"redeem\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\"},{\"name\":\"_spender\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"name\":\"remaining\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"basisPointsRate\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"isBlackListed\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_clearedUser\",\"type\":\"address\"}],\"name\":\"removeBlackList\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"MAX_UINT\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_blackListedUser\",\"type\":\"address\"}],\"name\":\"destroyBlackFunds\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"_initialSupply\",\"type\":\"uint256\"},{\"name\":\"_name\",\"type\":\"string\"},{\"name\":\"_symbol\",\"type\":\"string\"},{\"name\":\"_decimals\",\"type\":\"uint8\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"Issue\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"Redeem\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"newAddress\",\"type\":\"address\"}],\"name\":\"Deprecate\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"feeBasisPoints\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"maxFee\",\"type\":\"uint256\"}],\"name\":\"Params\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"_blackListedUser\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_balance\",\"type\":\"uint256\"}],\"name\":\"DestroyedBlackFunds\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"_user\",\"type\":\"address\"}],\"name\":\"AddedBlackList\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"_user\",\"type\":\"address\"}],\"name\":\"RemovedBlackList\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"spender\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[],\"name\":\"Pause\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[],\"name\":\"Unpause\",\"type\":\"event\"}]",
}

// TRC20ABI is the input ABI used to generate the binding from.
// Deprecated: Use TRC20MetaData.ABI instead.
var TRC20ABI = TRC20MetaData.ABI

// TRC20 is an auto generated Go binding around an Ethereum contract.
type TRC20 struct {
	TRC20Caller     // Read-only binding to the contract
	TRC20Transactor // Write-only binding to the contract
	TRC20Filterer   // Log filterer for contract events
}

// TRC20Caller is an auto generated read-only Go binding around an Ethereum contract.
type TRC20Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TRC20Transactor is an auto generated write-only Go binding around an Ethereum contract.
type TRC20Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TRC20Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type TRC20Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TRC20Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type TRC20Session struct {
	Contract     *TRC20            // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// TRC20CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type TRC20CallerSession struct {
	Contract *TRC20Caller  // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// TRC20TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type TRC20TransactorSession struct {
	Contract     *TRC20Transactor  // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// TRC20Raw is an auto generated low-level Go binding around an Ethereum contract.
type TRC20Raw struct {
	Contract *TRC20 // Generic contract binding to access the raw methods on
}

// TRC20CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type TRC20CallerRaw struct {
	Contract *TRC20Caller // Generic read-only contract binding to access the raw methods on
}

// TRC20TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type TRC20TransactorRaw struct {
	Contract *TRC20Transactor // Generic write-only contract binding to access the raw methods on
}

// NewTRC20 creates a new instance of TRC20, bound to a specific deployed contract.
func NewTRC20(address common.Address, backend bind.ContractBackend) (*TRC20, error) {
	contract, err := bindTRC20(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &TRC20{TRC20Caller: TRC20Caller{contract: contract}, TRC20Transactor: TRC20Transactor{contract: contract}, TRC20Filterer: TRC20Filterer{contract: contract}}, nil
}

// NewTRC20Caller creates a new read-only instance of TRC20, bound to a specific deployed contract.
func NewTRC20Caller(address common.Address, caller bind.ContractCaller) (*TRC20Caller, error) {
	contract, err := bindTRC20(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &TRC20Caller{contract: contract}, nil
}

// NewTRC20Transactor creates a new write-only instance of TRC20, bound to a specific deployed contract.
func NewTRC20Transactor(address common.Address, transactor bind.ContractTransactor) (*TRC20Transactor, error) {
	contract, err := bindTRC20(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &TRC20Transactor{contract: contract}, nil
}

// NewTRC20Filterer creates a new log filterer instance of TRC20, bound to a specific deployed contract.
func NewTRC20Filterer(address common.Address, filterer bind.ContractFilterer) (*TRC20Filterer, error) {
	contract, err := bindTRC20(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &TRC20Filterer{contract: contract}, nil
}

// bindTRC20 binds a generic wrapper to an already deployed contract.
func bindTRC20(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := TRC20MetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TRC20 *TRC20Raw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TRC20.Contract.TRC20Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TRC20 *TRC20Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TRC20.Contract.TRC20Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TRC20 *TRC20Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TRC20.Contract.TRC20Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TRC20 *TRC20CallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TRC20.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TRC20 *TRC20TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TRC20.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TRC20 *TRC20TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TRC20.Contract.contract.Transact(opts, method, params...)
}

// MAXUINT is a free data retrieval call binding the contract method 0xe5b5019a.
//
// Solidity: function MAX_UINT() view returns(uint256)
func (_TRC20 *TRC20Caller) MAXUINT(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _TRC20.contract.Call(opts, &out, "MAX_UINT")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MAXUINT is a free data retrieval call binding the contract method 0xe5b5019a.
//
// Solidity: function MAX_UINT() view returns(uint256)
func (_TRC20 *TRC20Session) MAXUINT() (*big.Int, error) {
	return _TRC20.Contract.MAXUINT(&_TRC20.CallOpts)
}

// MAXUINT is a free data retrieval call binding the contract method 0xe5b5019a.
//
// Solidity: function MAX_UINT() view returns(uint256)
func (_TRC20 *TRC20CallerSession) MAXUINT() (*big.Int, error) {
	return _TRC20.Contract.MAXUINT(&_TRC20.CallOpts)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address _owner, address _spender) view returns(uint256 remaining)
func (_TRC20 *TRC20Caller) Allowance(opts *bind.CallOpts, _owner common.Address, _spender common.Address) (*big.Int, error) {
	var out []interface{}
	err := _TRC20.contract.Call(opts, &out, "allowance", _owner, _spender)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address _owner, address _spender) view returns(uint256 remaining)
func (_TRC20 *TRC20Session) Allowance(_owner common.Address, _spender common.Address) (*big.Int, error) {
	return _TRC20.Contract.Allowance(&_TRC20.CallOpts, _owner, _spender)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address _owner, address _spender) view returns(uint256 remaining)
func (_TRC20 *TRC20CallerSession) Allowance(_owner common.Address, _spender common.Address) (*big.Int, error) {
	return _TRC20.Contract.Allowance(&_TRC20.CallOpts, _owner, _spender)
}

// Allowed is a free data retrieval call binding the contract method 0x5c658165.
//
// Solidity: function allowed(address , address ) view returns(uint256)
func (_TRC20 *TRC20Caller) Allowed(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _TRC20.contract.Call(opts, &out, "allowed", arg0, arg1)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Allowed is a free data retrieval call binding the contract method 0x5c658165.
//
// Solidity: function allowed(address , address ) view returns(uint256)
func (_TRC20 *TRC20Session) Allowed(arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	return _TRC20.Contract.Allowed(&_TRC20.CallOpts, arg0, arg1)
}

// Allowed is a free data retrieval call binding the contract method 0x5c658165.
//
// Solidity: function allowed(address , address ) view returns(uint256)
func (_TRC20 *TRC20CallerSession) Allowed(arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	return _TRC20.Contract.Allowed(&_TRC20.CallOpts, arg0, arg1)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address who) view returns(uint256)
func (_TRC20 *TRC20Caller) BalanceOf(opts *bind.CallOpts, who common.Address) (*big.Int, error) {
	var out []interface{}
	err := _TRC20.contract.Call(opts, &out, "balanceOf", who)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address who) view returns(uint256)
func (_TRC20 *TRC20Session) BalanceOf(who common.Address) (*big.Int, error) {
	return _TRC20.Contract.BalanceOf(&_TRC20.CallOpts, who)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address who) view returns(uint256)
func (_TRC20 *TRC20CallerSession) BalanceOf(who common.Address) (*big.Int, error) {
	return _TRC20.Contract.BalanceOf(&_TRC20.CallOpts, who)
}

// Balances is a free data retrieval call binding the contract method 0x27e235e3.
//
// Solidity: function balances(address ) view returns(uint256)
func (_TRC20 *TRC20Caller) Balances(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _TRC20.contract.Call(opts, &out, "balances", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Balances is a free data retrieval call binding the contract method 0x27e235e3.
//
// Solidity: function balances(address ) view returns(uint256)
func (_TRC20 *TRC20Session) Balances(arg0 common.Address) (*big.Int, error) {
	return _TRC20.Contract.Balances(&_TRC20.CallOpts, arg0)
}

// Balances is a free data retrieval call binding the contract method 0x27e235e3.
//
// Solidity: function balances(address ) view returns(uint256)
func (_TRC20 *TRC20CallerSession) Balances(arg0 common.Address) (*big.Int, error) {
	return _TRC20.Contract.Balances(&_TRC20.CallOpts, arg0)
}

// BasisPointsRate is a free data retrieval call binding the contract method 0xdd644f72.
//
// Solidity: function basisPointsRate() view returns(uint256)
func (_TRC20 *TRC20Caller) BasisPointsRate(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _TRC20.contract.Call(opts, &out, "basisPointsRate")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BasisPointsRate is a free data retrieval call binding the contract method 0xdd644f72.
//
// Solidity: function basisPointsRate() view returns(uint256)
func (_TRC20 *TRC20Session) BasisPointsRate() (*big.Int, error) {
	return _TRC20.Contract.BasisPointsRate(&_TRC20.CallOpts)
}

// BasisPointsRate is a free data retrieval call binding the contract method 0xdd644f72.
//
// Solidity: function basisPointsRate() view returns(uint256)
func (_TRC20 *TRC20CallerSession) BasisPointsRate() (*big.Int, error) {
	return _TRC20.Contract.BasisPointsRate(&_TRC20.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_TRC20 *TRC20Caller) Decimals(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _TRC20.contract.Call(opts, &out, "decimals")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_TRC20 *TRC20Session) Decimals() (uint8, error) {
	return _TRC20.Contract.Decimals(&_TRC20.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_TRC20 *TRC20CallerSession) Decimals() (uint8, error) {
	return _TRC20.Contract.Decimals(&_TRC20.CallOpts)
}

// Deprecated is a free data retrieval call binding the contract method 0x0e136b19.
//
// Solidity: function deprecated() view returns(bool)
func (_TRC20 *TRC20Caller) Deprecated(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _TRC20.contract.Call(opts, &out, "deprecated")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Deprecated is a free data retrieval call binding the contract method 0x0e136b19.
//
// Solidity: function deprecated() view returns(bool)
func (_TRC20 *TRC20Session) Deprecated() (bool, error) {
	return _TRC20.Contract.Deprecated(&_TRC20.CallOpts)
}

// Deprecated is a free data retrieval call binding the contract method 0x0e136b19.
//
// Solidity: function deprecated() view returns(bool)
func (_TRC20 *TRC20CallerSession) Deprecated() (bool, error) {
	return _TRC20.Contract.Deprecated(&_TRC20.CallOpts)
}

// GetBlackListStatus is a free data retrieval call binding the contract method 0x59bf1abe.
//
// Solidity: function getBlackListStatus(address _maker) view returns(bool)
func (_TRC20 *TRC20Caller) GetBlackListStatus(opts *bind.CallOpts, _maker common.Address) (bool, error) {
	var out []interface{}
	err := _TRC20.contract.Call(opts, &out, "getBlackListStatus", _maker)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// GetBlackListStatus is a free data retrieval call binding the contract method 0x59bf1abe.
//
// Solidity: function getBlackListStatus(address _maker) view returns(bool)
func (_TRC20 *TRC20Session) GetBlackListStatus(_maker common.Address) (bool, error) {
	return _TRC20.Contract.GetBlackListStatus(&_TRC20.CallOpts, _maker)
}

// GetBlackListStatus is a free data retrieval call binding the contract method 0x59bf1abe.
//
// Solidity: function getBlackListStatus(address _maker) view returns(bool)
func (_TRC20 *TRC20CallerSession) GetBlackListStatus(_maker common.Address) (bool, error) {
	return _TRC20.Contract.GetBlackListStatus(&_TRC20.CallOpts, _maker)
}

// GetOwner is a free data retrieval call binding the contract method 0x893d20e8.
//
// Solidity: function getOwner() view returns(address)
func (_TRC20 *TRC20Caller) GetOwner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _TRC20.contract.Call(opts, &out, "getOwner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetOwner is a free data retrieval call binding the contract method 0x893d20e8.
//
// Solidity: function getOwner() view returns(address)
func (_TRC20 *TRC20Session) GetOwner() (common.Address, error) {
	return _TRC20.Contract.GetOwner(&_TRC20.CallOpts)
}

// GetOwner is a free data retrieval call binding the contract method 0x893d20e8.
//
// Solidity: function getOwner() view returns(address)
func (_TRC20 *TRC20CallerSession) GetOwner() (common.Address, error) {
	return _TRC20.Contract.GetOwner(&_TRC20.CallOpts)
}

// IsBlackListed is a free data retrieval call binding the contract method 0xe47d6060.
//
// Solidity: function isBlackListed(address ) view returns(bool)
func (_TRC20 *TRC20Caller) IsBlackListed(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var out []interface{}
	err := _TRC20.contract.Call(opts, &out, "isBlackListed", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsBlackListed is a free data retrieval call binding the contract method 0xe47d6060.
//
// Solidity: function isBlackListed(address ) view returns(bool)
func (_TRC20 *TRC20Session) IsBlackListed(arg0 common.Address) (bool, error) {
	return _TRC20.Contract.IsBlackListed(&_TRC20.CallOpts, arg0)
}

// IsBlackListed is a free data retrieval call binding the contract method 0xe47d6060.
//
// Solidity: function isBlackListed(address ) view returns(bool)
func (_TRC20 *TRC20CallerSession) IsBlackListed(arg0 common.Address) (bool, error) {
	return _TRC20.Contract.IsBlackListed(&_TRC20.CallOpts, arg0)
}

// MaximumFee is a free data retrieval call binding the contract method 0x35390714.
//
// Solidity: function maximumFee() view returns(uint256)
func (_TRC20 *TRC20Caller) MaximumFee(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _TRC20.contract.Call(opts, &out, "maximumFee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaximumFee is a free data retrieval call binding the contract method 0x35390714.
//
// Solidity: function maximumFee() view returns(uint256)
func (_TRC20 *TRC20Session) MaximumFee() (*big.Int, error) {
	return _TRC20.Contract.MaximumFee(&_TRC20.CallOpts)
}

// MaximumFee is a free data retrieval call binding the contract method 0x35390714.
//
// Solidity: function maximumFee() view returns(uint256)
func (_TRC20 *TRC20CallerSession) MaximumFee() (*big.Int, error) {
	return _TRC20.Contract.MaximumFee(&_TRC20.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_TRC20 *TRC20Caller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _TRC20.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_TRC20 *TRC20Session) Name() (string, error) {
	return _TRC20.Contract.Name(&_TRC20.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_TRC20 *TRC20CallerSession) Name() (string, error) {
	return _TRC20.Contract.Name(&_TRC20.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_TRC20 *TRC20Caller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _TRC20.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_TRC20 *TRC20Session) Owner() (common.Address, error) {
	return _TRC20.Contract.Owner(&_TRC20.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_TRC20 *TRC20CallerSession) Owner() (common.Address, error) {
	return _TRC20.Contract.Owner(&_TRC20.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_TRC20 *TRC20Caller) Paused(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _TRC20.contract.Call(opts, &out, "paused")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_TRC20 *TRC20Session) Paused() (bool, error) {
	return _TRC20.Contract.Paused(&_TRC20.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_TRC20 *TRC20CallerSession) Paused() (bool, error) {
	return _TRC20.Contract.Paused(&_TRC20.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_TRC20 *TRC20Caller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _TRC20.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_TRC20 *TRC20Session) Symbol() (string, error) {
	return _TRC20.Contract.Symbol(&_TRC20.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_TRC20 *TRC20CallerSession) Symbol() (string, error) {
	return _TRC20.Contract.Symbol(&_TRC20.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_TRC20 *TRC20Caller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _TRC20.contract.Call(opts, &out, "totalSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_TRC20 *TRC20Session) TotalSupply() (*big.Int, error) {
	return _TRC20.Contract.TotalSupply(&_TRC20.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_TRC20 *TRC20CallerSession) TotalSupply() (*big.Int, error) {
	return _TRC20.Contract.TotalSupply(&_TRC20.CallOpts)
}

// UpgradedAddress is a free data retrieval call binding the contract method 0x26976e3f.
//
// Solidity: function upgradedAddress() view returns(address)
func (_TRC20 *TRC20Caller) UpgradedAddress(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _TRC20.contract.Call(opts, &out, "upgradedAddress")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// UpgradedAddress is a free data retrieval call binding the contract method 0x26976e3f.
//
// Solidity: function upgradedAddress() view returns(address)
func (_TRC20 *TRC20Session) UpgradedAddress() (common.Address, error) {
	return _TRC20.Contract.UpgradedAddress(&_TRC20.CallOpts)
}

// UpgradedAddress is a free data retrieval call binding the contract method 0x26976e3f.
//
// Solidity: function upgradedAddress() view returns(address)
func (_TRC20 *TRC20CallerSession) UpgradedAddress() (common.Address, error) {
	return _TRC20.Contract.UpgradedAddress(&_TRC20.CallOpts)
}

// AddBlackList is a paid mutator transaction binding the contract method 0x0ecb93c0.
//
// Solidity: function addBlackList(address _evilUser) returns()
func (_TRC20 *TRC20Transactor) AddBlackList(opts *bind.TransactOpts, _evilUser common.Address) (*types.Transaction, error) {
	return _TRC20.contract.Transact(opts, "addBlackList", _evilUser)
}

// AddBlackList is a paid mutator transaction binding the contract method 0x0ecb93c0.
//
// Solidity: function addBlackList(address _evilUser) returns()
func (_TRC20 *TRC20Session) AddBlackList(_evilUser common.Address) (*types.Transaction, error) {
	return _TRC20.Contract.AddBlackList(&_TRC20.TransactOpts, _evilUser)
}

// AddBlackList is a paid mutator transaction binding the contract method 0x0ecb93c0.
//
// Solidity: function addBlackList(address _evilUser) returns()
func (_TRC20 *TRC20TransactorSession) AddBlackList(_evilUser common.Address) (*types.Transaction, error) {
	return _TRC20.Contract.AddBlackList(&_TRC20.TransactOpts, _evilUser)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address _spender, uint256 _value) returns()
func (_TRC20 *TRC20Transactor) Approve(opts *bind.TransactOpts, _spender common.Address, _value *big.Int) (*types.Transaction, error) {
	return _TRC20.contract.Transact(opts, "approve", _spender, _value)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address _spender, uint256 _value) returns()
func (_TRC20 *TRC20Session) Approve(_spender common.Address, _value *big.Int) (*types.Transaction, error) {
	return _TRC20.Contract.Approve(&_TRC20.TransactOpts, _spender, _value)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address _spender, uint256 _value) returns()
func (_TRC20 *TRC20TransactorSession) Approve(_spender common.Address, _value *big.Int) (*types.Transaction, error) {
	return _TRC20.Contract.Approve(&_TRC20.TransactOpts, _spender, _value)
}

// Deprecate is a paid mutator transaction binding the contract method 0x0753c30c.
//
// Solidity: function deprecate(address _upgradedAddress) returns()
func (_TRC20 *TRC20Transactor) Deprecate(opts *bind.TransactOpts, _upgradedAddress common.Address) (*types.Transaction, error) {
	return _TRC20.contract.Transact(opts, "deprecate", _upgradedAddress)
}

// Deprecate is a paid mutator transaction binding the contract method 0x0753c30c.
//
// Solidity: function deprecate(address _upgradedAddress) returns()
func (_TRC20 *TRC20Session) Deprecate(_upgradedAddress common.Address) (*types.Transaction, error) {
	return _TRC20.Contract.Deprecate(&_TRC20.TransactOpts, _upgradedAddress)
}

// Deprecate is a paid mutator transaction binding the contract method 0x0753c30c.
//
// Solidity: function deprecate(address _upgradedAddress) returns()
func (_TRC20 *TRC20TransactorSession) Deprecate(_upgradedAddress common.Address) (*types.Transaction, error) {
	return _TRC20.Contract.Deprecate(&_TRC20.TransactOpts, _upgradedAddress)
}

// DestroyBlackFunds is a paid mutator transaction binding the contract method 0xf3bdc228.
//
// Solidity: function destroyBlackFunds(address _blackListedUser) returns()
func (_TRC20 *TRC20Transactor) DestroyBlackFunds(opts *bind.TransactOpts, _blackListedUser common.Address) (*types.Transaction, error) {
	return _TRC20.contract.Transact(opts, "destroyBlackFunds", _blackListedUser)
}

// DestroyBlackFunds is a paid mutator transaction binding the contract method 0xf3bdc228.
//
// Solidity: function destroyBlackFunds(address _blackListedUser) returns()
func (_TRC20 *TRC20Session) DestroyBlackFunds(_blackListedUser common.Address) (*types.Transaction, error) {
	return _TRC20.Contract.DestroyBlackFunds(&_TRC20.TransactOpts, _blackListedUser)
}

// DestroyBlackFunds is a paid mutator transaction binding the contract method 0xf3bdc228.
//
// Solidity: function destroyBlackFunds(address _blackListedUser) returns()
func (_TRC20 *TRC20TransactorSession) DestroyBlackFunds(_blackListedUser common.Address) (*types.Transaction, error) {
	return _TRC20.Contract.DestroyBlackFunds(&_TRC20.TransactOpts, _blackListedUser)
}

// Issue is a paid mutator transaction binding the contract method 0xcc872b66.
//
// Solidity: function issue(uint256 amount) returns()
func (_TRC20 *TRC20Transactor) Issue(opts *bind.TransactOpts, amount *big.Int) (*types.Transaction, error) {
	return _TRC20.contract.Transact(opts, "issue", amount)
}

// Issue is a paid mutator transaction binding the contract method 0xcc872b66.
//
// Solidity: function issue(uint256 amount) returns()
func (_TRC20 *TRC20Session) Issue(amount *big.Int) (*types.Transaction, error) {
	return _TRC20.Contract.Issue(&_TRC20.TransactOpts, amount)
}

// Issue is a paid mutator transaction binding the contract method 0xcc872b66.
//
// Solidity: function issue(uint256 amount) returns()
func (_TRC20 *TRC20TransactorSession) Issue(amount *big.Int) (*types.Transaction, error) {
	return _TRC20.Contract.Issue(&_TRC20.TransactOpts, amount)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_TRC20 *TRC20Transactor) Pause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TRC20.contract.Transact(opts, "pause")
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_TRC20 *TRC20Session) Pause() (*types.Transaction, error) {
	return _TRC20.Contract.Pause(&_TRC20.TransactOpts)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_TRC20 *TRC20TransactorSession) Pause() (*types.Transaction, error) {
	return _TRC20.Contract.Pause(&_TRC20.TransactOpts)
}

// Redeem is a paid mutator transaction binding the contract method 0xdb006a75.
//
// Solidity: function redeem(uint256 amount) returns()
func (_TRC20 *TRC20Transactor) Redeem(opts *bind.TransactOpts, amount *big.Int) (*types.Transaction, error) {
	return _TRC20.contract.Transact(opts, "redeem", amount)
}

// Redeem is a paid mutator transaction binding the contract method 0xdb006a75.
//
// Solidity: function redeem(uint256 amount) returns()
func (_TRC20 *TRC20Session) Redeem(amount *big.Int) (*types.Transaction, error) {
	return _TRC20.Contract.Redeem(&_TRC20.TransactOpts, amount)
}

// Redeem is a paid mutator transaction binding the contract method 0xdb006a75.
//
// Solidity: function redeem(uint256 amount) returns()
func (_TRC20 *TRC20TransactorSession) Redeem(amount *big.Int) (*types.Transaction, error) {
	return _TRC20.Contract.Redeem(&_TRC20.TransactOpts, amount)
}

// RemoveBlackList is a paid mutator transaction binding the contract method 0xe4997dc5.
//
// Solidity: function removeBlackList(address _clearedUser) returns()
func (_TRC20 *TRC20Transactor) RemoveBlackList(opts *bind.TransactOpts, _clearedUser common.Address) (*types.Transaction, error) {
	return _TRC20.contract.Transact(opts, "removeBlackList", _clearedUser)
}

// RemoveBlackList is a paid mutator transaction binding the contract method 0xe4997dc5.
//
// Solidity: function removeBlackList(address _clearedUser) returns()
func (_TRC20 *TRC20Session) RemoveBlackList(_clearedUser common.Address) (*types.Transaction, error) {
	return _TRC20.Contract.RemoveBlackList(&_TRC20.TransactOpts, _clearedUser)
}

// RemoveBlackList is a paid mutator transaction binding the contract method 0xe4997dc5.
//
// Solidity: function removeBlackList(address _clearedUser) returns()
func (_TRC20 *TRC20TransactorSession) RemoveBlackList(_clearedUser common.Address) (*types.Transaction, error) {
	return _TRC20.Contract.RemoveBlackList(&_TRC20.TransactOpts, _clearedUser)
}

// SetParams is a paid mutator transaction binding the contract method 0xc0324c77.
//
// Solidity: function setParams(uint256 newBasisPoints, uint256 newMaxFee) returns()
func (_TRC20 *TRC20Transactor) SetParams(opts *bind.TransactOpts, newBasisPoints *big.Int, newMaxFee *big.Int) (*types.Transaction, error) {
	return _TRC20.contract.Transact(opts, "setParams", newBasisPoints, newMaxFee)
}

// SetParams is a paid mutator transaction binding the contract method 0xc0324c77.
//
// Solidity: function setParams(uint256 newBasisPoints, uint256 newMaxFee) returns()
func (_TRC20 *TRC20Session) SetParams(newBasisPoints *big.Int, newMaxFee *big.Int) (*types.Transaction, error) {
	return _TRC20.Contract.SetParams(&_TRC20.TransactOpts, newBasisPoints, newMaxFee)
}

// SetParams is a paid mutator transaction binding the contract method 0xc0324c77.
//
// Solidity: function setParams(uint256 newBasisPoints, uint256 newMaxFee) returns()
func (_TRC20 *TRC20TransactorSession) SetParams(newBasisPoints *big.Int, newMaxFee *big.Int) (*types.Transaction, error) {
	return _TRC20.Contract.SetParams(&_TRC20.TransactOpts, newBasisPoints, newMaxFee)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address _to, uint256 _value) returns()
func (_TRC20 *TRC20Transactor) Transfer(opts *bind.TransactOpts, _to common.Address, _value *big.Int) (*types.Transaction, error) {
	return _TRC20.contract.Transact(opts, "transfer", _to, _value)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address _to, uint256 _value) returns()
func (_TRC20 *TRC20Session) Transfer(_to common.Address, _value *big.Int) (*types.Transaction, error) {
	return _TRC20.Contract.Transfer(&_TRC20.TransactOpts, _to, _value)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address _to, uint256 _value) returns()
func (_TRC20 *TRC20TransactorSession) Transfer(_to common.Address, _value *big.Int) (*types.Transaction, error) {
	return _TRC20.Contract.Transfer(&_TRC20.TransactOpts, _to, _value)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address _from, address _to, uint256 _value) returns()
func (_TRC20 *TRC20Transactor) TransferFrom(opts *bind.TransactOpts, _from common.Address, _to common.Address, _value *big.Int) (*types.Transaction, error) {
	return _TRC20.contract.Transact(opts, "transferFrom", _from, _to, _value)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address _from, address _to, uint256 _value) returns()
func (_TRC20 *TRC20Session) TransferFrom(_from common.Address, _to common.Address, _value *big.Int) (*types.Transaction, error) {
	return _TRC20.Contract.TransferFrom(&_TRC20.TransactOpts, _from, _to, _value)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address _from, address _to, uint256 _value) returns()
func (_TRC20 *TRC20TransactorSession) TransferFrom(_from common.Address, _to common.Address, _value *big.Int) (*types.Transaction, error) {
	return _TRC20.Contract.TransferFrom(&_TRC20.TransactOpts, _from, _to, _value)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_TRC20 *TRC20Transactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _TRC20.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_TRC20 *TRC20Session) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _TRC20.Contract.TransferOwnership(&_TRC20.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_TRC20 *TRC20TransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _TRC20.Contract.TransferOwnership(&_TRC20.TransactOpts, newOwner)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_TRC20 *TRC20Transactor) Unpause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TRC20.contract.Transact(opts, "unpause")
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_TRC20 *TRC20Session) Unpause() (*types.Transaction, error) {
	return _TRC20.Contract.Unpause(&_TRC20.TransactOpts)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_TRC20 *TRC20TransactorSession) Unpause() (*types.Transaction, error) {
	return _TRC20.Contract.Unpause(&_TRC20.TransactOpts)
}

// TRC20AddedBlackListIterator is returned from FilterAddedBlackList and is used to iterate over the raw logs and unpacked data for AddedBlackList events raised by the TRC20 contract.
type TRC20AddedBlackListIterator struct {
	Event *TRC20AddedBlackList // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *TRC20AddedBlackListIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TRC20AddedBlackList)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(TRC20AddedBlackList)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *TRC20AddedBlackListIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TRC20AddedBlackListIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TRC20AddedBlackList represents a AddedBlackList event raised by the TRC20 contract.
type TRC20AddedBlackList struct {
	User common.Address
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterAddedBlackList is a free log retrieval operation binding the contract event 0x42e160154868087d6bfdc0ca23d96a1c1cfa32f1b72ba9ba27b69b98a0d819dc.
//
// Solidity: event AddedBlackList(address _user)
func (_TRC20 *TRC20Filterer) FilterAddedBlackList(opts *bind.FilterOpts) (*TRC20AddedBlackListIterator, error) {

	logs, sub, err := _TRC20.contract.FilterLogs(opts, "AddedBlackList")
	if err != nil {
		return nil, err
	}
	return &TRC20AddedBlackListIterator{contract: _TRC20.contract, event: "AddedBlackList", logs: logs, sub: sub}, nil
}

// WatchAddedBlackList is a free log subscription operation binding the contract event 0x42e160154868087d6bfdc0ca23d96a1c1cfa32f1b72ba9ba27b69b98a0d819dc.
//
// Solidity: event AddedBlackList(address _user)
func (_TRC20 *TRC20Filterer) WatchAddedBlackList(opts *bind.WatchOpts, sink chan<- *TRC20AddedBlackList) (event.Subscription, error) {

	logs, sub, err := _TRC20.contract.WatchLogs(opts, "AddedBlackList")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TRC20AddedBlackList)
				if err := _TRC20.contract.UnpackLog(event, "AddedBlackList", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseAddedBlackList is a log parse operation binding the contract event 0x42e160154868087d6bfdc0ca23d96a1c1cfa32f1b72ba9ba27b69b98a0d819dc.
//
// Solidity: event AddedBlackList(address _user)
func (_TRC20 *TRC20Filterer) ParseAddedBlackList(log types.Log) (*TRC20AddedBlackList, error) {
	event := new(TRC20AddedBlackList)
	if err := _TRC20.contract.UnpackLog(event, "AddedBlackList", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TRC20ApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the TRC20 contract.
type TRC20ApprovalIterator struct {
	Event *TRC20Approval // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *TRC20ApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TRC20Approval)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(TRC20Approval)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *TRC20ApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TRC20ApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TRC20Approval represents a Approval event raised by the TRC20 contract.
type TRC20Approval struct {
	Owner   common.Address
	Spender common.Address
	Value   *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_TRC20 *TRC20Filterer) FilterApproval(opts *bind.FilterOpts, owner []common.Address, spender []common.Address) (*TRC20ApprovalIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _TRC20.contract.FilterLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return &TRC20ApprovalIterator{contract: _TRC20.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_TRC20 *TRC20Filterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *TRC20Approval, owner []common.Address, spender []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _TRC20.contract.WatchLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TRC20Approval)
				if err := _TRC20.contract.UnpackLog(event, "Approval", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseApproval is a log parse operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_TRC20 *TRC20Filterer) ParseApproval(log types.Log) (*TRC20Approval, error) {
	event := new(TRC20Approval)
	if err := _TRC20.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TRC20DeprecateIterator is returned from FilterDeprecate and is used to iterate over the raw logs and unpacked data for Deprecate events raised by the TRC20 contract.
type TRC20DeprecateIterator struct {
	Event *TRC20Deprecate // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *TRC20DeprecateIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TRC20Deprecate)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(TRC20Deprecate)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *TRC20DeprecateIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TRC20DeprecateIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TRC20Deprecate represents a Deprecate event raised by the TRC20 contract.
type TRC20Deprecate struct {
	NewAddress common.Address
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterDeprecate is a free log retrieval operation binding the contract event 0xcc358699805e9a8b7f77b522628c7cb9abd07d9efb86b6fb616af1609036a99e.
//
// Solidity: event Deprecate(address newAddress)
func (_TRC20 *TRC20Filterer) FilterDeprecate(opts *bind.FilterOpts) (*TRC20DeprecateIterator, error) {

	logs, sub, err := _TRC20.contract.FilterLogs(opts, "Deprecate")
	if err != nil {
		return nil, err
	}
	return &TRC20DeprecateIterator{contract: _TRC20.contract, event: "Deprecate", logs: logs, sub: sub}, nil
}

// WatchDeprecate is a free log subscription operation binding the contract event 0xcc358699805e9a8b7f77b522628c7cb9abd07d9efb86b6fb616af1609036a99e.
//
// Solidity: event Deprecate(address newAddress)
func (_TRC20 *TRC20Filterer) WatchDeprecate(opts *bind.WatchOpts, sink chan<- *TRC20Deprecate) (event.Subscription, error) {

	logs, sub, err := _TRC20.contract.WatchLogs(opts, "Deprecate")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TRC20Deprecate)
				if err := _TRC20.contract.UnpackLog(event, "Deprecate", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseDeprecate is a log parse operation binding the contract event 0xcc358699805e9a8b7f77b522628c7cb9abd07d9efb86b6fb616af1609036a99e.
//
// Solidity: event Deprecate(address newAddress)
func (_TRC20 *TRC20Filterer) ParseDeprecate(log types.Log) (*TRC20Deprecate, error) {
	event := new(TRC20Deprecate)
	if err := _TRC20.contract.UnpackLog(event, "Deprecate", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TRC20DestroyedBlackFundsIterator is returned from FilterDestroyedBlackFunds and is used to iterate over the raw logs and unpacked data for DestroyedBlackFunds events raised by the TRC20 contract.
type TRC20DestroyedBlackFundsIterator struct {
	Event *TRC20DestroyedBlackFunds // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *TRC20DestroyedBlackFundsIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TRC20DestroyedBlackFunds)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(TRC20DestroyedBlackFunds)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *TRC20DestroyedBlackFundsIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TRC20DestroyedBlackFundsIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TRC20DestroyedBlackFunds represents a DestroyedBlackFunds event raised by the TRC20 contract.
type TRC20DestroyedBlackFunds struct {
	BlackListedUser common.Address
	Balance         *big.Int
	Raw             types.Log // Blockchain specific contextual infos
}

// FilterDestroyedBlackFunds is a free log retrieval operation binding the contract event 0x61e6e66b0d6339b2980aecc6ccc0039736791f0ccde9ed512e789a7fbdd698c6.
//
// Solidity: event DestroyedBlackFunds(address _blackListedUser, uint256 _balance)
func (_TRC20 *TRC20Filterer) FilterDestroyedBlackFunds(opts *bind.FilterOpts) (*TRC20DestroyedBlackFundsIterator, error) {

	logs, sub, err := _TRC20.contract.FilterLogs(opts, "DestroyedBlackFunds")
	if err != nil {
		return nil, err
	}
	return &TRC20DestroyedBlackFundsIterator{contract: _TRC20.contract, event: "DestroyedBlackFunds", logs: logs, sub: sub}, nil
}

// WatchDestroyedBlackFunds is a free log subscription operation binding the contract event 0x61e6e66b0d6339b2980aecc6ccc0039736791f0ccde9ed512e789a7fbdd698c6.
//
// Solidity: event DestroyedBlackFunds(address _blackListedUser, uint256 _balance)
func (_TRC20 *TRC20Filterer) WatchDestroyedBlackFunds(opts *bind.WatchOpts, sink chan<- *TRC20DestroyedBlackFunds) (event.Subscription, error) {

	logs, sub, err := _TRC20.contract.WatchLogs(opts, "DestroyedBlackFunds")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TRC20DestroyedBlackFunds)
				if err := _TRC20.contract.UnpackLog(event, "DestroyedBlackFunds", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseDestroyedBlackFunds is a log parse operation binding the contract event 0x61e6e66b0d6339b2980aecc6ccc0039736791f0ccde9ed512e789a7fbdd698c6.
//
// Solidity: event DestroyedBlackFunds(address _blackListedUser, uint256 _balance)
func (_TRC20 *TRC20Filterer) ParseDestroyedBlackFunds(log types.Log) (*TRC20DestroyedBlackFunds, error) {
	event := new(TRC20DestroyedBlackFunds)
	if err := _TRC20.contract.UnpackLog(event, "DestroyedBlackFunds", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TRC20IssueIterator is returned from FilterIssue and is used to iterate over the raw logs and unpacked data for Issue events raised by the TRC20 contract.
type TRC20IssueIterator struct {
	Event *TRC20Issue // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *TRC20IssueIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TRC20Issue)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(TRC20Issue)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *TRC20IssueIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TRC20IssueIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TRC20Issue represents a Issue event raised by the TRC20 contract.
type TRC20Issue struct {
	Amount *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterIssue is a free log retrieval operation binding the contract event 0xcb8241adb0c3fdb35b70c24ce35c5eb0c17af7431c99f827d44a445ca624176a.
//
// Solidity: event Issue(uint256 amount)
func (_TRC20 *TRC20Filterer) FilterIssue(opts *bind.FilterOpts) (*TRC20IssueIterator, error) {

	logs, sub, err := _TRC20.contract.FilterLogs(opts, "Issue")
	if err != nil {
		return nil, err
	}
	return &TRC20IssueIterator{contract: _TRC20.contract, event: "Issue", logs: logs, sub: sub}, nil
}

// WatchIssue is a free log subscription operation binding the contract event 0xcb8241adb0c3fdb35b70c24ce35c5eb0c17af7431c99f827d44a445ca624176a.
//
// Solidity: event Issue(uint256 amount)
func (_TRC20 *TRC20Filterer) WatchIssue(opts *bind.WatchOpts, sink chan<- *TRC20Issue) (event.Subscription, error) {

	logs, sub, err := _TRC20.contract.WatchLogs(opts, "Issue")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TRC20Issue)
				if err := _TRC20.contract.UnpackLog(event, "Issue", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseIssue is a log parse operation binding the contract event 0xcb8241adb0c3fdb35b70c24ce35c5eb0c17af7431c99f827d44a445ca624176a.
//
// Solidity: event Issue(uint256 amount)
func (_TRC20 *TRC20Filterer) ParseIssue(log types.Log) (*TRC20Issue, error) {
	event := new(TRC20Issue)
	if err := _TRC20.contract.UnpackLog(event, "Issue", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TRC20ParamsIterator is returned from FilterParams and is used to iterate over the raw logs and unpacked data for Params events raised by the TRC20 contract.
type TRC20ParamsIterator struct {
	Event *TRC20Params // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *TRC20ParamsIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TRC20Params)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(TRC20Params)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *TRC20ParamsIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TRC20ParamsIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TRC20Params represents a Params event raised by the TRC20 contract.
type TRC20Params struct {
	FeeBasisPoints *big.Int
	MaxFee         *big.Int
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterParams is a free log retrieval operation binding the contract event 0xb044a1e409eac5c48e5af22d4af52670dd1a99059537a78b31b48c6500a6354e.
//
// Solidity: event Params(uint256 feeBasisPoints, uint256 maxFee)
func (_TRC20 *TRC20Filterer) FilterParams(opts *bind.FilterOpts) (*TRC20ParamsIterator, error) {

	logs, sub, err := _TRC20.contract.FilterLogs(opts, "Params")
	if err != nil {
		return nil, err
	}
	return &TRC20ParamsIterator{contract: _TRC20.contract, event: "Params", logs: logs, sub: sub}, nil
}

// WatchParams is a free log subscription operation binding the contract event 0xb044a1e409eac5c48e5af22d4af52670dd1a99059537a78b31b48c6500a6354e.
//
// Solidity: event Params(uint256 feeBasisPoints, uint256 maxFee)
func (_TRC20 *TRC20Filterer) WatchParams(opts *bind.WatchOpts, sink chan<- *TRC20Params) (event.Subscription, error) {

	logs, sub, err := _TRC20.contract.WatchLogs(opts, "Params")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TRC20Params)
				if err := _TRC20.contract.UnpackLog(event, "Params", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseParams is a log parse operation binding the contract event 0xb044a1e409eac5c48e5af22d4af52670dd1a99059537a78b31b48c6500a6354e.
//
// Solidity: event Params(uint256 feeBasisPoints, uint256 maxFee)
func (_TRC20 *TRC20Filterer) ParseParams(log types.Log) (*TRC20Params, error) {
	event := new(TRC20Params)
	if err := _TRC20.contract.UnpackLog(event, "Params", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TRC20PauseIterator is returned from FilterPause and is used to iterate over the raw logs and unpacked data for Pause events raised by the TRC20 contract.
type TRC20PauseIterator struct {
	Event *TRC20Pause // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *TRC20PauseIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TRC20Pause)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(TRC20Pause)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *TRC20PauseIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TRC20PauseIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TRC20Pause represents a Pause event raised by the TRC20 contract.
type TRC20Pause struct {
	Raw types.Log // Blockchain specific contextual infos
}

// FilterPause is a free log retrieval operation binding the contract event 0x6985a02210a168e66602d3235cb6db0e70f92b3ba4d376a33c0f3d9434bff625.
//
// Solidity: event Pause()
func (_TRC20 *TRC20Filterer) FilterPause(opts *bind.FilterOpts) (*TRC20PauseIterator, error) {

	logs, sub, err := _TRC20.contract.FilterLogs(opts, "Pause")
	if err != nil {
		return nil, err
	}
	return &TRC20PauseIterator{contract: _TRC20.contract, event: "Pause", logs: logs, sub: sub}, nil
}

// WatchPause is a free log subscription operation binding the contract event 0x6985a02210a168e66602d3235cb6db0e70f92b3ba4d376a33c0f3d9434bff625.
//
// Solidity: event Pause()
func (_TRC20 *TRC20Filterer) WatchPause(opts *bind.WatchOpts, sink chan<- *TRC20Pause) (event.Subscription, error) {

	logs, sub, err := _TRC20.contract.WatchLogs(opts, "Pause")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TRC20Pause)
				if err := _TRC20.contract.UnpackLog(event, "Pause", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParsePause is a log parse operation binding the contract event 0x6985a02210a168e66602d3235cb6db0e70f92b3ba4d376a33c0f3d9434bff625.
//
// Solidity: event Pause()
func (_TRC20 *TRC20Filterer) ParsePause(log types.Log) (*TRC20Pause, error) {
	event := new(TRC20Pause)
	if err := _TRC20.contract.UnpackLog(event, "Pause", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TRC20RedeemIterator is returned from FilterRedeem and is used to iterate over the raw logs and unpacked data for Redeem events raised by the TRC20 contract.
type TRC20RedeemIterator struct {
	Event *TRC20Redeem // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *TRC20RedeemIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TRC20Redeem)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(TRC20Redeem)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *TRC20RedeemIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TRC20RedeemIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TRC20Redeem represents a Redeem event raised by the TRC20 contract.
type TRC20Redeem struct {
	Amount *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterRedeem is a free log retrieval operation binding the contract event 0x702d5967f45f6513a38ffc42d6ba9bf230bd40e8f53b16363c7eb4fd2deb9a44.
//
// Solidity: event Redeem(uint256 amount)
func (_TRC20 *TRC20Filterer) FilterRedeem(opts *bind.FilterOpts) (*TRC20RedeemIterator, error) {

	logs, sub, err := _TRC20.contract.FilterLogs(opts, "Redeem")
	if err != nil {
		return nil, err
	}
	return &TRC20RedeemIterator{contract: _TRC20.contract, event: "Redeem", logs: logs, sub: sub}, nil
}

// WatchRedeem is a free log subscription operation binding the contract event 0x702d5967f45f6513a38ffc42d6ba9bf230bd40e8f53b16363c7eb4fd2deb9a44.
//
// Solidity: event Redeem(uint256 amount)
func (_TRC20 *TRC20Filterer) WatchRedeem(opts *bind.WatchOpts, sink chan<- *TRC20Redeem) (event.Subscription, error) {

	logs, sub, err := _TRC20.contract.WatchLogs(opts, "Redeem")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TRC20Redeem)
				if err := _TRC20.contract.UnpackLog(event, "Redeem", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseRedeem is a log parse operation binding the contract event 0x702d5967f45f6513a38ffc42d6ba9bf230bd40e8f53b16363c7eb4fd2deb9a44.
//
// Solidity: event Redeem(uint256 amount)
func (_TRC20 *TRC20Filterer) ParseRedeem(log types.Log) (*TRC20Redeem, error) {
	event := new(TRC20Redeem)
	if err := _TRC20.contract.UnpackLog(event, "Redeem", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TRC20RemovedBlackListIterator is returned from FilterRemovedBlackList and is used to iterate over the raw logs and unpacked data for RemovedBlackList events raised by the TRC20 contract.
type TRC20RemovedBlackListIterator struct {
	Event *TRC20RemovedBlackList // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *TRC20RemovedBlackListIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TRC20RemovedBlackList)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(TRC20RemovedBlackList)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *TRC20RemovedBlackListIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TRC20RemovedBlackListIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TRC20RemovedBlackList represents a RemovedBlackList event raised by the TRC20 contract.
type TRC20RemovedBlackList struct {
	User common.Address
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterRemovedBlackList is a free log retrieval operation binding the contract event 0xd7e9ec6e6ecd65492dce6bf513cd6867560d49544421d0783ddf06e76c24470c.
//
// Solidity: event RemovedBlackList(address _user)
func (_TRC20 *TRC20Filterer) FilterRemovedBlackList(opts *bind.FilterOpts) (*TRC20RemovedBlackListIterator, error) {

	logs, sub, err := _TRC20.contract.FilterLogs(opts, "RemovedBlackList")
	if err != nil {
		return nil, err
	}
	return &TRC20RemovedBlackListIterator{contract: _TRC20.contract, event: "RemovedBlackList", logs: logs, sub: sub}, nil
}

// WatchRemovedBlackList is a free log subscription operation binding the contract event 0xd7e9ec6e6ecd65492dce6bf513cd6867560d49544421d0783ddf06e76c24470c.
//
// Solidity: event RemovedBlackList(address _user)
func (_TRC20 *TRC20Filterer) WatchRemovedBlackList(opts *bind.WatchOpts, sink chan<- *TRC20RemovedBlackList) (event.Subscription, error) {

	logs, sub, err := _TRC20.contract.WatchLogs(opts, "RemovedBlackList")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TRC20RemovedBlackList)
				if err := _TRC20.contract.UnpackLog(event, "RemovedBlackList", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseRemovedBlackList is a log parse operation binding the contract event 0xd7e9ec6e6ecd65492dce6bf513cd6867560d49544421d0783ddf06e76c24470c.
//
// Solidity: event RemovedBlackList(address _user)
func (_TRC20 *TRC20Filterer) ParseRemovedBlackList(log types.Log) (*TRC20RemovedBlackList, error) {
	event := new(TRC20RemovedBlackList)
	if err := _TRC20.contract.UnpackLog(event, "RemovedBlackList", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TRC20TransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the TRC20 contract.
type TRC20TransferIterator struct {
	Event *TRC20Transfer // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *TRC20TransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TRC20Transfer)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(TRC20Transfer)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *TRC20TransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TRC20TransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TRC20Transfer represents a Transfer event raised by the TRC20 contract.
type TRC20Transfer struct {
	From  common.Address
	To    common.Address
	Value *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_TRC20 *TRC20Filterer) FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*TRC20TransferIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _TRC20.contract.FilterLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &TRC20TransferIterator{contract: _TRC20.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_TRC20 *TRC20Filterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *TRC20Transfer, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _TRC20.contract.WatchLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TRC20Transfer)
				if err := _TRC20.contract.UnpackLog(event, "Transfer", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseTransfer is a log parse operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_TRC20 *TRC20Filterer) ParseTransfer(log types.Log) (*TRC20Transfer, error) {
	event := new(TRC20Transfer)
	if err := _TRC20.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TRC20UnpauseIterator is returned from FilterUnpause and is used to iterate over the raw logs and unpacked data for Unpause events raised by the TRC20 contract.
type TRC20UnpauseIterator struct {
	Event *TRC20Unpause // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *TRC20UnpauseIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TRC20Unpause)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(TRC20Unpause)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *TRC20UnpauseIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TRC20UnpauseIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TRC20Unpause represents a Unpause event raised by the TRC20 contract.
type TRC20Unpause struct {
	Raw types.Log // Blockchain specific contextual infos
}

// FilterUnpause is a free log retrieval operation binding the contract event 0x7805862f689e2f13df9f062ff482ad3ad112aca9e0847911ed832e158c525b33.
//
// Solidity: event Unpause()
func (_TRC20 *TRC20Filterer) FilterUnpause(opts *bind.FilterOpts) (*TRC20UnpauseIterator, error) {

	logs, sub, err := _TRC20.contract.FilterLogs(opts, "Unpause")
	if err != nil {
		return nil, err
	}
	return &TRC20UnpauseIterator{contract: _TRC20.contract, event: "Unpause", logs: logs, sub: sub}, nil
}

// WatchUnpause is a free log subscription operation binding the contract event 0x7805862f689e2f13df9f062ff482ad3ad112aca9e0847911ed832e158c525b33.
//
// Solidity: event Unpause()
func (_TRC20 *TRC20Filterer) WatchUnpause(opts *bind.WatchOpts, sink chan<- *TRC20Unpause) (event.Subscription, error) {

	logs, sub, err := _TRC20.contract.WatchLogs(opts, "Unpause")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TRC20Unpause)
				if err := _TRC20.contract.UnpackLog(event, "Unpause", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseUnpause is a log parse operation binding the contract event 0x7805862f689e2f13df9f062ff482ad3ad112aca9e0847911ed832e158c525b33.
//
// Solidity: event Unpause()
func (_TRC20 *TRC20Filterer) ParseUnpause(log types.Log) (*TRC20Unpause, error) {
	event := new(TRC20Unpause)
	if err := _TRC20.contract.UnpackLog(event, "Unpause", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
