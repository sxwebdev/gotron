# Gotron API Surface Reference

## Table of Contents

- [Package gotron (top-level)](#package-gotron)
- [Package pkg/client](#package-pkgclient)
  - [Client construction](#client-construction)
  - [Account operations](#account-operations)
  - [Activation operations](#activation-operations)
  - [Block operations](#block-operations)
  - [Transaction operations](#transaction-operations)
  - [TRC20 token operations](#trc20-token-operations)
  - [Resource operations](#resource-operations)
  - [Estimate operations](#estimate-operations)
  - [Contract operations](#contract-operations)
  - [Network operations](#network-operations)
  - [Chain parameters](#chain-parameters)
  - [Common types](#common-types)
- [Package pkg/address](#package-pkgaddress)
- [Package pkg/tronutils](#package-pkgtronutils)

---

## Package gotron

**File:** `tron.go`

```go
type Tron struct { *client.Client }
type Config = client.Config

func New(cfg Config) (*Tron, error)
```

**Constants (re-exported from client):**

```go
Mainnet = client.NetworkMainnet
Shasta  = client.NetworkShasta
Nile    = client.NetworkNile
Bandwidth = client.ResourceTypeBandwidth
Energy    = client.ResourceTypeEnergy
TrxDecimals = client.TrxDecimals  // 6
Trc20TransferEventSignature = client.Trc20TransferEventSignature
```

---

## Package pkg/client

### Client construction

**File:** `client.go`, `config.go`

```go
type Client struct { /* unexported: transport, config */ }
func New(cfg Config) (*Client, error)
func (c *Client) Close() error
func (c *Client) GetNetwork() Network
```

```go
type Config struct {
    Nodes      []NodeConfig
    Network    Network         // informational only
    Blockchain string          // metrics label, default "tron"
    Metrics    MetricsCollector // nil = no metrics
}
func (c Config) Validate() error
```

```go
type NodeConfig struct {
    Protocol    Protocol              // "grpc" (default) or "http"
    Address     string                // "grpc.trongrid.io:50051" or "https://api.trongrid.io"
    UseTLS      bool                  // gRPC only
    DialOptions []grpc.DialOption     // gRPC only
    HTTPClient  *http.Client          // HTTP only
    Headers     map[string]string     // API keys, custom metadata
}
func (n NodeConfig) Validate() error
func (n NodeConfig) GetProtocol() Protocol
```

### Account operations

**File:** `account.go`

```go
func (c *Client) GetAccount(ctx context.Context, addr string) (*core.Account, error)
func (c *Client) GetAccountBalance(ctx context.Context, address string) (decimal.Decimal, error)
func (c *Client) IsAccountActivated(ctx context.Context, address string) (bool, error)
func (c *Client) CreateAccount(ctx context.Context, from, addr string, accountType core.AccountType) (*api.TransactionExtention, error)
func (c *Client) EstimateActivateAccount(ctx context.Context, fromAddress, toAddress string) (*EstimateActivateAccountResult, error)
```

```go
// Legacy result type, kept for backwards compatibility with EstimateActivateAccount.
// New code should use EstimateResult (see Common types).
type EstimateActivateAccountResult struct {
    Energy    decimal.Decimal `json:"energy"`
    Bandwidth decimal.Decimal `json:"bandwidth"`
    Trx       decimal.Decimal `json:"trx"`
}
```

### Activation operations

**File:** `activate.go`

Two estimators for the cost of activating a Tron address. Both return `*EstimateResult` and return zeros if the recipient is already activated.

```go
// EstimateActivationFee builds a local fake CreateAccount tx to size the bandwidth.
// fromAddress is assumed to be activated (typical processing-wallet case).
func (c *Client) EstimateActivationFee(ctx context.Context, fromAddress, toAddress string) (*EstimateResult, error)

// EstimateSystemContractActivation builds a real CreateAccount tx via the node
// (more accurate, one extra RPC). Returns zero result for already-activated receivers
// — including the race where the address gets activated between the IsAccountActivated
// check and CreateAccount ("Account has existed" is treated as zero, not error).
func (c *Client) EstimateSystemContractActivation(ctx context.Context, caller, receiver string) (*EstimateResult, error)
```

The activation fee in TRX is computed from chain params:

- Constant fee: `chainParams.CreateNewAccountFeeInSystemContract` (typically 1 TRX) — always added.
- If caller has enough **own staked** bandwidth: that bandwidth is consumed (`Bandwidth` field set).
- Otherwise: extra `chainParams.CreateAccountFee` (typically 0.1 TRX) is burned. Free daily quota and bandwidth received via delegation do **not** count.

### Block operations

**File:** `block.go`

```go
func (c *Client) GetLastBlock(ctx context.Context) (*api.BlockExtention, error)
func (c *Client) GetLastBlockHeight(ctx context.Context) (uint64, error)
func (c *Client) GetBlockByHeight(ctx context.Context, height uint64) (*api.BlockExtention, error)
func (c *Client) GetBlockByHash(ctx context.Context, hash []byte) (*core.Block, error)
func (c *Client) GetTransactionInfoByBlockNum(ctx context.Context, number uint64) (*api.TransactionInfoList, error)
func (c *Client) GetBlockByLimitNext2(ctx context.Context, start, end uint64) (*api.BlockListExtention, error)
func (c *Client) GetBlockByLatestNum2(ctx context.Context, height uint64) (*api.BlockListExtention, error)
```

### Transaction operations

**File:** `transactions.go`

```go
func (c *Client) GetTransactionByHash(ctx context.Context, hash string) (*core.Transaction, error)
func (c *Client) GetTransactionInfoByHash(ctx context.Context, hash string) (*core.TransactionInfo, error)
func (c *Client) GetTransactionExtensionByHash(ctx context.Context, hash string) (*api.TransactionExtention, *core.TransactionInfo, error)
func (c *Client) BroadcastTransaction(ctx context.Context, tx *core.Transaction) (*api.Return, error)
func (c *Client) SignTransaction(tx *core.Transaction, privateKey *ecdsa.PrivateKey) error
```

**File:** `transfer.go`

```go
// amount is in TRX (auto-converted to SUN internally)
func (c *Client) CreateTransferTransaction(ctx context.Context, from, to string, amount decimal.Decimal) (*api.TransactionExtention, error)
```

### TRC20 token operations

**File:** `trc20.go`

```go
func (c *Client) TRC20Call(ctx context.Context, from, contractAddress, data string, constant bool, feeLimit int64) (*api.TransactionExtention, error)
func (c *Client) TRC20GetName(ctx context.Context, contractAddress string) (string, error)
func (c *Client) TRC20GetSymbol(ctx context.Context, contractAddress string) (string, error)
func (c *Client) TRC20GetDecimals(ctx context.Context, contractAddress string) (*big.Int, error)
func (c *Client) TRC20ContractBalance(ctx context.Context, addr, contractAddress string) (*big.Int, error)
func (c *Client) TRC20Send(ctx context.Context, from, to, contract string, amount decimal.Decimal, feeLimit int64) (*api.TransactionExtention, error)
func (c *Client) TRC20Approve(ctx context.Context, from, to, contract string, amount decimal.Decimal, feeLimit int64) (*api.TransactionExtention, error)
func (c *Client) TRC20TransferFrom(ctx context.Context, owner, from, to, contract string, amount *big.Int, feeLimit int64) (*api.TransactionExtention, error)
func (c *Client) ParseTRC20NumericProperty(data string) (*big.Int, error)
func (c *Client) ParseTRC20StringProperty(data string) (string, error)
```

### Resource operations

**File:** `resources.go`

```go
func (c *Client) GetAccountResource(ctx context.Context, addr string) (*api.AccountResourceMessage, error)
func (c *Client) GetDelegatedResources(ctx context.Context, address string) ([]*api.DelegatedResourceList, error)
func (c *Client) GetDelegatedResourcesV2(ctx context.Context, address string) ([]*api.DelegatedResourceList, error)
func (c *Client) GetCanDelegatedMaxSize(ctx context.Context, address string, resource int32) (*api.CanDelegatedMaxSizeResponseMessage, error)
func (c *Client) DelegateResource(ctx context.Context, owner, receiver string, resource ResourceType, delegateBalance int64, lock bool, lockPeriod int64) (*api.TransactionExtention, error)
func (c *Client) ReclaimResource(ctx context.Context, owner, receiver string, resource ResourceType, delegateBalance int64) (*api.TransactionExtention, error)
func (c *Client) AvailableForDelegateResources(ctx context.Context, addr string) (*AvailableResources, error)
func (c *Client) TotalAvailableResources(ctx context.Context, addr string) (*AvailableResources, error)
func (c *Client) AvailableEnergy(res *api.AccountResourceMessage) decimal.Decimal
func (c *Client) AvailableBandwidth(res *api.AccountResourceMessage) decimal.Decimal
func (c *Client) AvailableBandwidthWithoutFree(res *api.AccountResourceMessage) decimal.Decimal
func (c *Client) TotalEnergyLimit(res *api.AccountResourceMessage) decimal.Decimal
func (c *Client) TotalBandwidthLimit(res *api.AccountResourceMessage) decimal.Decimal
```

```go
// Renamed from "Resources" — represents an account's currently usable resources
// plus its hard limits. Used as return type by AvailableForDelegateResources
// and TotalAvailableResources.
type AvailableResources struct {
    Energy         decimal.Decimal `json:"energy"`
    Bandwidth      decimal.Decimal `json:"bandwidth"`
    TotalEnergy    decimal.Decimal `json:"total_energy"`
    TotalBandwidth decimal.Decimal `json:"total_bandwidth"`
}
```

### Estimate operations

Cost estimators for transactions and transfers. Use these to compute fees before broadcasting.

**File:** `estimate_resources.go`

```go
// EstimateBandwidth fills required signature bytes into a fake transaction
// and returns proto.Size(tx) + 64 (Tron protocol overhead) as bandwidth points.
func (c *Client) EstimateBandwidth(tx *core.Transaction) (decimal.Decimal, error)

// EstimateEnergy queries the node's /wallet/estimateenergy or gRPC EstimateEnergy
// for a contract call. Used internally by EstimateTransfer for TRC20 paths.
func (c *Client) EstimateEnergy(
    ctx context.Context,
    from, contractAddress, method, jsonString string,
    tAmount int64, tTokenID string, tTokenAmount int64,
) (*api.EstimateEnergyMessage, error)
```

**File:** `estimate_transfer.go`

```go
// EstimateTransfer estimates the full cost of a TRX or TRC20 transfer,
// broken down into:
//   - Transfer:   the cost of the transfer transaction itself
//   - Activation: the cost of activating toAddress (zero if already activated)
//   - Total:      Transfer + Activation per resource (conservative upper bound)
//
// For TRX transfers pass contractAddress = TrxAssetIdentifier and decimals = TrxDecimals.
// For TRC20 pass the token contract address and the token's decimals.
//
// Note: when sending to an unactivated address Tron consumes the activation fee
// inside the transfer tx itself — Total slightly overestimates. Choose your own
// merge policy if you need a single number (e.g. max(Transfer.Trx, Activation.Trx)).
func (c *Client) EstimateTransfer(
    ctx context.Context,
    fromAddress, toAddress, contractAddress string,
    amount decimal.Decimal,
    decimals int64,
) (*EstimateTransferResult, error)
```

```go
type EstimateTransferResult struct {
    Total      EstimateResult `json:"total"`
    Transfer   EstimateResult `json:"transfer"`
    Activation EstimateResult `json:"activation"`
}
```

### Contract operations

**File:** `contract.go`

```go
func (c *Client) TriggerConstantContract(ctx context.Context, ct *core.TriggerSmartContract) (*api.TransactionExtention, error)
func (c *Client) TriggerConstantContractCustom(ctx context.Context, from, contractAddress, method, jsonString string) (*api.TransactionExtention, error)
func (c *Client) DeployContract(ctx context.Context, ct *core.CreateSmartContract) (*api.TransactionExtention, error)
func (c *Client) GetContract(ctx context.Context, address string) (*core.SmartContract, error)
```

### Network operations

**File:** `network.go`

```go
func (c *Client) ListNodes(ctx context.Context) (*api.NodeList, error)
func (c *Client) GetNextMaintenanceTime(ctx context.Context) (*api.NumberMessage, error)
func (c *Client) TotalTransaction(ctx context.Context) (*api.NumberMessage, error)
```

### Chain parameters

**File:** `chain_params.go`

```go
func (c *Client) ChainParams(ctx context.Context) (*ChainParams, error)
func (c *Client) ChainParam(ctx context.Context, paramKey string) (*core.ChainParameters_ChainParameter, error)
```

### Common types

**File:** `types.go`

Generic resource cost type used across activation and transfer estimators.

```go
// EstimateResult is the canonical result shape for any "how much does this cost"
// query — Energy and Bandwidth in raw points, Trx in TRX units (not SUN).
// Reused by EstimateActivationFee, EstimateSystemContractActivation, and as the
// value type for fields of EstimateTransferResult (Total/Transfer/Activation).
type EstimateResult struct {
    Energy    decimal.Decimal `json:"energy"`
    Bandwidth decimal.Decimal `json:"bandwidth"`
    Trx       decimal.Decimal `json:"trx"`
}
```

---

## Package pkg/address

**Files:** `address.go`, `generator.go`

```go
type Address struct {
    PrivateKeyECDSA *ecdsa.PrivateKey
    PublicKeyECDSA  *ecdsa.PublicKey
    PrivateKey      string   // hex-encoded
    PublicKey       string   // hex-encoded (compressed)
    Address         string   // base58 Tron address
    Mnemonic        string   // BIP39 mnemonic (if generated from mnemonic)
}

func Generate() (*Address, error)
func GenerateMnemonic(strength int) (string, error)
func FromMnemonic(mnemonic, passphrase string, index uint32) (*Address, error)
func FromPrivateKey(privateKeyHex string) (*Address, error)
func Validate(address string) error
```

**Generator (advanced BIP44 derivation):**

```go
type Generator struct { /* ... */ }
func NewGenerator(mnemonic, passphrase string) *Generator
func (g *Generator) SetBipPurpose(purpose uint32) *Generator
func (g *Generator) SetCoinType(coinType uint32) *Generator
func (g *Generator) SetAccount(account uint32) *Generator
func (g *Generator) SetNetwork(net *chaincfg.Params) *Generator
func (g *Generator) Generate(index uint32) (*Address, error)
```

Default BIP44 path: `m/44'/195'/0'/0/{index}`

---

## Package pkg/tronutils

**Files:** `address.go`, `hex.go`, `number.go`, `encoding.go`

**Address utilities:**

```go
func DecodeCheck(addr string) ([]byte, error)   // base58 -> bytes (with checksum verify)
func EncodeCheck(data []byte) string             // bytes -> base58 (with checksum)
func Base58ToAddress(s string) (Address, error)
func Base64ToAddress(s string) (Address, error)
func HexToAddress(s string) Address
func BigToAddress(b *big.Int) Address
```

**Hex utilities:**

```go
func BytesToHexString(bytes []byte) string  // "0x..." prefixed
func FromHex(s string) ([]byte, error)      // supports "0x" prefix
func Has0xPrefix(str string) bool
func IsHex(str string) bool
func Bytes2Hex(d []byte) string             // no prefix
func Hex2Bytes(str string) ([]byte, error)  // no prefix
func LeftPadBytes(slice []byte, l int) []byte
func RightPadBytes(slice []byte, l int) []byte
```

**Number utilities:**

```go
func FormatPrecisionNumber(value, decimals int) string
func DoubleSHA256(data []byte) []byte
```
