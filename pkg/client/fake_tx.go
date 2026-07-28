package client

import (
	"crypto/sha256"
	"time"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/core"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// CreateFakeResourceTransaction creates a fake resource transaction.
func CreateFakeResourceTransaction(fromAddress, toAddress string, amount int64, resourceType core.ResourceCode, reclaim bool) (*core.Transaction, error) {
	addrFromBytes, err := tronutils.DecodeCheck(fromAddress)
	if err != nil {
		return nil, err
	}

	addrToBytes, err := tronutils.DecodeCheck(toAddress)
	if err != nil {
		return nil, err
	}

	var contract proto.Message
	var transactionContractType core.Transaction_Contract_ContractType

	if !reclaim {
		contract = &core.DelegateResourceContract{
			OwnerAddress:    addrFromBytes,
			ReceiverAddress: addrToBytes,
			Balance:         amount,
			Resource:        resourceType,
			Lock:            false,
			LockPeriod:      0,
		}

		transactionContractType = core.Transaction_Contract_DelegateResourceContract
	} else {
		contract = &core.UnDelegateResourceContract{
			OwnerAddress:    addrFromBytes,
			ReceiverAddress: addrToBytes,
			Balance:         amount,
			Resource:        resourceType,
		}

		transactionContractType = core.Transaction_Contract_UnDelegateResourceContract
	}

	contractAnyType, err := anypb.New(contract)
	if err != nil {
		return nil, err
	}

	refBlockBytes := []byte{0x01, 0x01}
	hash := []byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01}
	now := time.Now().UnixNano() / int64(time.Millisecond)

	transaction := &core.Transaction{
		RawData: &core.TransactionRaw{
			RefBlockBytes: refBlockBytes,
			RefBlockHash:  hash,
			Expiration:    now,
			Timestamp:     now,
			Contract: []*core.Transaction_Contract{
				{
					Type:      transactionContractType,
					Parameter: contractAnyType,
				},
			},
		},
	}

	return transaction, nil
}

func CreateFakeCreateAccountTransaction(fromAddress, toAddress string) (*core.Transaction, error) {
	addrFromBytes, err := tronutils.DecodeCheck(fromAddress)
	if err != nil {
		return nil, err
	}

	addrToBytes, err := tronutils.DecodeCheck(toAddress)
	if err != nil {
		return nil, err
	}

	refBlockBytes := []byte{0x01, 0x01}
	hash := []byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01}
	now := time.Now().UnixNano() / int64(time.Millisecond)

	contract := &core.AccountCreateContract{
		OwnerAddress:   addrFromBytes,
		AccountAddress: addrToBytes,
		Type:           core.AccountType_Normal,
	}

	contractAnyType, err := anypb.New(contract)
	if err != nil {
		return nil, err
	}

	tx := &core.Transaction{
		RawData: &core.TransactionRaw{
			RefBlockBytes: refBlockBytes,
			RefBlockHash:  hash,
			Expiration:    now,
			Timestamp:     now,
			Contract: []*core.Transaction_Contract{
				{
					Type:      core.Transaction_Contract_AccountCreateContract,
					Parameter: contractAnyType,
				},
			},
		},
	}

	return tx, nil
}

// CreateFakeStakeTransaction creates a fake FreezeBalanceV2 (unstake=false) or
// UnfreezeBalanceV2 (unstake=true) transaction for bandwidth estimation.
func CreateFakeStakeTransaction(ownerAddress string, amount int64, resourceType core.ResourceCode, unstake bool) (*core.Transaction, error) {
	addrOwnerBytes, err := tronutils.DecodeCheck(ownerAddress)
	if err != nil {
		return nil, err
	}

	var contract proto.Message
	var transactionContractType core.Transaction_Contract_ContractType

	if !unstake {
		contract = &core.FreezeBalanceV2Contract{
			OwnerAddress:  addrOwnerBytes,
			FrozenBalance: amount,
			Resource:      resourceType,
		}

		transactionContractType = core.Transaction_Contract_FreezeBalanceV2Contract
	} else {
		contract = &core.UnfreezeBalanceV2Contract{
			OwnerAddress:    addrOwnerBytes,
			UnfreezeBalance: amount,
			Resource:        resourceType,
		}

		transactionContractType = core.Transaction_Contract_UnfreezeBalanceV2Contract
	}

	contractAnyType, err := anypb.New(contract)
	if err != nil {
		return nil, err
	}

	refBlockBytes := []byte{0x01, 0x01}
	hash := []byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01}
	now := time.Now().UnixNano() / int64(time.Millisecond)

	transaction := &core.Transaction{
		RawData: &core.TransactionRaw{
			RefBlockBytes: refBlockBytes,
			RefBlockHash:  hash,
			Expiration:    now,
			Timestamp:     now,
			Contract: []*core.Transaction_Contract{
				{
					Type:      transactionContractType,
					Parameter: contractAnyType,
				},
			},
		},
	}

	return transaction, nil
}

// CreateFakeWithdrawUnstakedTransaction creates a fake WithdrawExpireUnfreeze
// transaction for bandwidth estimation. CancelAllUnfreezeV2 carries the same
// owner-only contract, so its estimate matches this one.
func CreateFakeWithdrawUnstakedTransaction(ownerAddress string) (*core.Transaction, error) {
	addrOwnerBytes, err := tronutils.DecodeCheck(ownerAddress)
	if err != nil {
		return nil, err
	}

	contract := &core.WithdrawExpireUnfreezeContract{OwnerAddress: addrOwnerBytes}

	contractAnyType, err := anypb.New(contract)
	if err != nil {
		return nil, err
	}

	refBlockBytes := []byte{0x01, 0x01}
	hash := []byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01}
	now := time.Now().UnixNano() / int64(time.Millisecond)

	tx := &core.Transaction{
		RawData: &core.TransactionRaw{
			RefBlockBytes: refBlockBytes,
			RefBlockHash:  hash,
			Expiration:    now,
			Timestamp:     now,
			Contract: []*core.Transaction_Contract{
				{
					Type:      core.Transaction_Contract_WithdrawExpireUnfreezeContract,
					Parameter: contractAnyType,
				},
			},
		},
	}

	return tx, nil
}

// fillFakeTX fills the transaction with fake data.
func fillFakeTX(tx *core.Transaction) error {
	tx.Ret = nil

	rawData, err := proto.Marshal(tx.GetRawData())
	if err != nil {
		return err
	}

	h256h := sha256.New()
	_, err = h256h.Write(rawData)
	if err != nil {
		return err
	}

	pk, err := secp256k1.GeneratePrivateKey()
	if err != nil {
		return err
	}

	signature, err := crypto.Sign(h256h.Sum(nil), pk.ToECDSA())
	if err != nil {
		return err
	}

	tx.Signature = append(tx.Signature, signature)

	return nil
}
