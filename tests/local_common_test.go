package tests

// Tests against a local single-witness java-tron private network.
//
// Public nodes cannot exercise the write paths - deploying, staking,
// delegating, a refused broadcast - and give no way to put a known transaction
// on chain and read it back through both transports. These tests do, and run
// only when GOTRON_LOCAL_NODE=1. The network is defined in tests/localnet:
//
//	make localnet-up    # docker compose, waits for the first blocks
//	make test-local     # GOTRON_LOCAL_NODE=1 go test -race ./tests/ -run Local
//	make localnet-down  # removes the chain; the next up starts from genesis
//
// GOTRON_LOCAL_GRPC and GOTRON_LOCAL_HTTP override the endpoints. The network's
// only witness holds every TRX and signs with the private key 0x…01.

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/client"
	"github.com/sxwebdev/gotron/pkg/client/abi"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

const (
	defaultLocalGRPC = "127.0.0.1:50051"
	defaultLocalHTTP = "http://127.0.0.1:18190"

	// localWitnessKey is the genesis witness of the private network; it holds
	// the whole TRX supply.
	localWitnessKey     = "0000000000000000000000000000000000000000000000000000000000000001"
	localWitnessAddress = "TMVQGm1qAQYVdetCeGRRkTWYYrLXuHK2HC"

	// localSelfPayingContract is hand-assembled bytecode whose runtime sends 1
	// SUN back to whoever calls it, so every call carries an internal
	// transaction in its receipt. Init: CODECOPY the 15-byte runtime and
	// RETURN it. Runtime: CALL(gas, caller, 1, 0, 0, 0, 0); POP; STOP.
	localSelfPayingContract = "600f80600b6000396000f3" + "60006000600060006001335af15000"

	localReceiptTimeout = 45 * time.Second
)

func localGRPCAddress() string {
	if v := os.Getenv("GOTRON_LOCAL_GRPC"); v != "" {
		return v
	}
	return defaultLocalGRPC
}

func localHTTPAddress() string {
	if v := os.Getenv("GOTRON_LOCAL_HTTP"); v != "" {
		return v
	}
	return defaultLocalHTTP
}

func requireLocalNode(t *testing.T) {
	t.Helper()
	if os.Getenv("GOTRON_LOCAL_NODE") != "1" {
		t.Skip("set GOTRON_LOCAL_NODE=1 to run tests against a local private network")
	}
}

func newLocalClient(t *testing.T, protocol client.Protocol) *client.Client {
	t.Helper()

	address := localGRPCAddress()
	if protocol == client.ProtocolHTTP {
		address = localHTTPAddress()
	}

	c, err := client.New(client.Config{
		Nodes: []client.NodeConfig{{Protocol: protocol, Address: address}},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })

	return c
}

func newLocalGRPCClient(t *testing.T) *client.Client {
	return newLocalClient(t, client.ProtocolGRPC)
}

func newLocalHTTPClient(t *testing.T) *client.Client {
	return newLocalClient(t, client.ProtocolHTTP)
}

// localTransports returns the two bare transports, which the parity tests
// call directly so that nothing in the Client layer can paper over a
// difference between them.
func localTransports(t *testing.T) (*client.GRPCTransport, *client.HTTPTransport) {
	t.Helper()

	g, err := client.NewGRPCTransport(client.NodeConfig{Protocol: client.ProtocolGRPC, Address: localGRPCAddress()})
	require.NoError(t, err)
	t.Cleanup(func() { _ = g.Close() })

	h, err := client.NewHTTPTransport(client.NodeConfig{Protocol: client.ProtocolHTTP, Address: localHTTPAddress()})
	require.NoError(t, err)

	return g, h
}

func localContext(t *testing.T, d time.Duration) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), d)
	t.Cleanup(cancel)
	return ctx
}

// localAccount is a key the tests can sign with.
type localAccount struct {
	Address string
	Key     *ecdsa.PrivateKey
}

func localWitness(t *testing.T) localAccount {
	t.Helper()
	key, err := crypto.HexToECDSA(localWitnessKey)
	require.NoError(t, err)
	return localAccount{Address: localWitnessAddress, Key: key}
}

func newLocalKey(t *testing.T) localAccount {
	t.Helper()
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	address, err := client.AddressFromPrivateKeyRaw(crypto.FromECDSA(key))
	require.NoError(t, err)
	return localAccount{Address: address, Key: key}
}

func trx(n int64) client.SUN {
	return client.MustFromTRX(decimal.NewFromInt(n))
}

// signAndBroadcast signs ext with signer, broadcasts it through c and returns
// its txid in hex. It does not wait for the block.
func signAndBroadcast(t *testing.T, c *client.Client, ext *api.TransactionExtention, signer localAccount) string {
	t.Helper()
	require.NotNil(t, ext)
	require.NoError(t, c.SignTransaction(ext.GetTransaction(), signer.Key))

	ctx := localContext(t, 15*time.Second)
	_, err := c.BroadcastTransaction(ctx, ext.GetTransaction())
	require.NoError(t, err)

	return hex.EncodeToString(ext.GetTxid())
}

// waitReceipt polls c until the receipt of txid is in a block.
func waitReceipt(t *testing.T, c *client.Client, txid string) *core.TransactionInfo {
	t.Helper()

	ctx := localContext(t, localReceiptTimeout)
	for {
		info, err := c.GetTransactionInfoByHash(ctx, txid)
		if err == nil && info.GetBlockNumber() > 0 {
			return info
		}
		if err != nil && !errors.Is(err, client.ErrTransactionInfoNotFound) {
			require.NoError(t, err, "receipt of %s", txid)
		}

		select {
		case <-ctx.Done():
			t.Fatalf("no receipt for %s within %s (last error: %v)", txid, localReceiptTimeout, err)
		case <-time.After(500 * time.Millisecond):
		}
	}
}

// sendAndConfirm is signAndBroadcast followed by waitReceipt on the same
// client, and fails the test unless the transaction succeeded.
func sendAndConfirm(t *testing.T, c *client.Client, ext *api.TransactionExtention, signer localAccount) (string, *core.TransactionInfo) {
	t.Helper()
	txid := signAndBroadcast(t, c, ext, signer)
	info := waitReceipt(t, c, txid)
	require.Equal(t, core.TransactionInfo_SUCESS, info.GetResult(), "tx %s failed: %s", txid, info.GetResMessage())
	return txid, info
}

// fundAccount sends amount from the witness to to and waits for the block.
func fundAccount(t *testing.T, c *client.Client, to string, amount client.SUN) string {
	t.Helper()
	ctx := localContext(t, 10*time.Second)
	ext, err := c.CreateTransferTransaction(ctx, localWitnessAddress, to, amount)
	require.NoError(t, err)
	txid, _ := sendAndConfirm(t, c, ext, localWitness(t))
	return txid
}

func readTestdata(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	require.NoError(t, err)
	return strings.TrimSpace(string(b))
}

// tetherTokenRequest is mainnet USDT's creation transaction: its 46-entry ABI
// and its bytecode with the original constructor arguments appended (10 USDT
// supply, 6 decimals, minted to the deployer).
func tetherTokenRequest(t *testing.T, from string) client.DeployContractRequest {
	t.Helper()
	contractABI, err := abi.LoadContractABI(readTestdata(t, "tethertoken.abi.json"))
	require.NoError(t, err)
	require.Len(t, contractABI.GetEntrys(), 46)

	return client.DeployContractRequest{
		From:                       from,
		Name:                       "TetherToken",
		ABI:                        contractABI,
		Bytecode:                   readTestdata(t, "tethertoken.bin"),
		FeeLimit:                   trx(1000),
		ConsumeUserResourcePercent: 100,
		OriginEnergyLimit:          10_000_000,
	}
}

// requireProtoEqual fails with a field-level diff when the two messages differ.
func requireProtoEqual(t *testing.T, want, got proto.Message, msgAndArgs ...any) {
	t.Helper()
	if proto.Equal(want, got) {
		return
	}
	opts := prototext.MarshalOptions{Multiline: true, EmitUnknown: true}
	require.Equal(t, opts.Format(want), opts.Format(got), msgAndArgs...)
}
