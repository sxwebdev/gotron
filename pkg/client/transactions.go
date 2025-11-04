package client

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sxwebdev/gotron/pkg/utils"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
	"google.golang.org/protobuf/proto"
)

// GetTransactionByHash returns transaction details by hash
func (c *Client) GetTransactionByHash(ctx context.Context, hash string) (*core.Transaction, error) {
	transactionID := new(api.BytesMessage)
	var err error

	transactionID.Value, err = utils.FromHex(hash)
	if err != nil {
		return nil, fmt.Errorf("get transaction by hash error: %v", err)
	}

	tx, err := c.walletClient.GetTransactionById(ctx, transactionID)
	if err != nil {
		return nil, err
	}
	if size := proto.Size(tx); size > 0 {
		return tx, nil
	}
	return nil, ErrTransactionNotFound
}

// GetTransactionInfoByHash returns transaction receipt by hash
func (c *Client) GetTransactionInfoByHash(ctx context.Context, hash string) (*core.TransactionInfo, error) {
	transactionID := new(api.BytesMessage)
	var err error

	transactionID.Value, err = utils.FromHex(hash)
	if err != nil {
		return nil, fmt.Errorf("get transaction by hash error: %v", err)
	}

	txi, err := c.walletClient.GetTransactionInfoById(ctx, transactionID)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(txi.Id, transactionID.Value) {
		return txi, nil
	}
	return nil, ErrTransactionInfoNotFound
}

func (c *Client) GetTransactionExtensionByHash(ctx context.Context, hash string) (*api.TransactionExtention, *core.TransactionInfo, error) {
	// Get transaction info
	txi, err := c.GetTransactionInfoByHash(ctx, hash)
	if err != nil {
		return nil, nil, err
	}

	// get block by height
	block, err := c.GetBlockByHeight(ctx, uint64(txi.GetBlockNumber()))
	if err != nil {
		return nil, nil, err
	}

	// find transaction in block
	var tx *api.TransactionExtention
	for _, item := range block.GetTransactions() {
		if bytes.Equal(item.GetTxid(), txi.GetId()) {
			tx = item
			break
		}
	}

	if tx == nil {
		return nil, nil, fmt.Errorf("can not find tx %s in block %d", hash, txi.GetBlockNumber())
	}

	return tx, txi, nil
}

// BroadcastTransaction broadcasts a signed transaction to the network
func (c *Client) BroadcastTransaction(ctx context.Context, tx *core.Transaction) (*api.Return, error) {
	result, err := c.walletClient.BroadcastTransaction(ctx, tx)
	if err != nil {
		return nil, err
	}
	if !result.GetResult() {
		return result, fmt.Errorf("result error: %s", result.GetMessage())
	}
	if result.GetCode() != api.Return_SUCCESS {
		return result, fmt.Errorf("result error(%s): %s", result.GetCode(), result.GetMessage())
	}
	return result, nil
}

// SignTransaction signs a raw transaction with the given private key
func (c *Client) SignTransaction(tx *core.Transaction, privateKey *ecdsa.PrivateKey) error {
	if tx == nil {
		return fmt.Errorf("empty tron tx")
	}

	rawData, err := proto.Marshal(tx.GetRawData())
	if err != nil {
		return err
	}

	h256h := sha256.New()
	if _, err := h256h.Write(rawData); err != nil {
		return err
	}

	signature, err := crypto.Sign(h256h.Sum(nil), privateKey)
	if err != nil {
		return err
	}

	tx.Signature = append(tx.Signature, signature)

	return nil
}
