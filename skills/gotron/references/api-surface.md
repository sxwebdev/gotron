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
  - [Resource pricing helpers](#resource-pricing-helpers)
  - [Staking operations](#staking-operations)
  - [Witness and reward operations](#witness-and-reward-operations)
  - [Estimate operations](#estimate-operations)
  - [Contract operations](#contract-operations)
  - [Asset operations (TRC10)](#asset-operations-trc10)
  - [Network operations](#network-operations)
  - [Chain parameters](#chain-parameters)
  - [Transport stack (low-level)](#transport-stack-low-level)
  - [Common types](#common-types)
  - [Sentinel errors](#sentinel-errors)
- [Package pkg/client/abi](#package-pkgclientabi)
- [Package pkg/address](#package-pkgaddress)
- [Package pkg/units](#package-pkgunits)
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
SunPerTRX   = units.SunPerTRX     // 1_000_000
Trc20TransferEventSignature = client.Trc20TransferEventSignature
```

**Amount types and constructors (re-exported from client/units):**

```go
type SUN = client.SUN
type TokenAmount = client.TokenAmount

var FromTRX          = client.FromTRX
var MustFromTRX      = client.MustFromTRX
var FromTokenUnits   = client.FromTokenUnits
var FromTokenDecimal = client.FromTokenDecimal
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
    Network    Network          // informational only
    Blockchain string           // metrics label, default "tron"
    Metrics    MetricsCollector // nil = no metrics
    Health     HealthConfig     // zero value = sane defaults; .Disabled=true → legacy round-robin
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
    Tier        int                   // 0 = primary, 1 = fallback, 2+ = next; default 0
}
func (n NodeConfig) Validate() error
func (n NodeConfig) GetProtocol() Protocol
```

```go
type HealthConfig struct {
    Disabled             bool          // true → legacy plain RoundRobinTransport
    FailureThreshold     int           // default 2
    SuccessThreshold     int           // default 2
    HealthyInterval      time.Duration // default 30s — healthy nodes in active tier
    UnhealthyInterval    time.Duration // default 5s  — any unhealthy node
    InactiveTierInterval time.Duration // default 5m  — healthy nodes in inactive tier (fallbacks)
    ProbeTimeout         time.Duration // default 5s
    Probe                func(ctx context.Context, t Transport) error // default = GetNowBlock
    ClassifyErr          func(err error) bool                         // default = isNetworkError
    Logger               Logger                                       // nil = no-op (silent)
}

// Logger is the minimal interface used by HealthAwareTransport. Implement
// Infof to bridge to slog, log, zap, etc.; the zero value of HealthConfig
// uses a no-op logger.
type Logger interface {
    Infof(format string, args ...any)
}
```

### Account operations

**File:** `account.go`

```go
func (c *Client) GetAccount(ctx context.Context, addr string) (*core.Account, error)
func (c *Client) GetAccountBalance(ctx context.Context, address string) (SUN, error)
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
    Fee       SUN             `json:"fee"`
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

**These two are for standalone activation only — do not add their result to a transfer estimate.**
A TRX transfer to a new address creates the account inside the transfer transaction itself, so
`EstimateTRXTransfer` already prices the creation; adding an activation estimate on top counts a
`CreateAccount` transaction that never reaches the chain. Their surcharge decision is also less
precise than the transfer estimator's: they compare against `AvailableForDelegateResources`, whose
`Bandwidth` is `min(staked limit from FrozenV2, staked remaining + free remaining)`, so free
bandwidth can make it report staked bandwidth the account does not have.

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
func (c *Client) CreateTransferTransaction(ctx context.Context, from, to string, amount SUN) (*api.TransactionExtention, error)
```

### TRC20 token operations

**File:** `trc20.go`

```go
// Every amount is a TokenAmount (the token's own minimal units) and every fee
// limit is a SUN. Build a TokenAmount with FromTokenDecimal (using the decimals
// TRC20GetDecimals reports) or FromTokenUnits.
func (c *Client) TRC20Call(ctx context.Context, from, contractAddress, data string, constant bool, feeLimit SUN) (*api.TransactionExtention, error)
func (c *Client) TRC20GetName(ctx context.Context, contractAddress string) (string, error)
func (c *Client) TRC20GetSymbol(ctx context.Context, contractAddress string) (string, error)
func (c *Client) TRC20GetDecimals(ctx context.Context, contractAddress string) (*big.Int, error)
func (c *Client) TRC20ContractBalance(ctx context.Context, addr, contractAddress string) (TokenAmount, error)
func (c *Client) TRC20Send(ctx context.Context, from, to, contract string, amount TokenAmount, feeLimit SUN) (*api.TransactionExtention, error)
func (c *Client) TRC20Approve(ctx context.Context, from, to, contract string, amount TokenAmount, feeLimit SUN) (*api.TransactionExtention, error)
func (c *Client) TRC20TransferFrom(ctx context.Context, owner, from, to, contract string, amount TokenAmount, feeLimit SUN) (*api.TransactionExtention, error)
func (c *Client) ParseTRC20NumericProperty(data string) (*big.Int, error)
func (c *Client) ParseTRC20StringProperty(data string) (string, error)
```

### Resource operations

**File:** `resources.go`

```go
func (c *Client) GetAccountResource(ctx context.Context, addr string) (*api.AccountResourceMessage, error)
func (c *Client) GetDelegatedResources(ctx context.Context, address string) ([]Delegation, error)   // Stake 1.0 index
func (c *Client) GetDelegatedResourcesV2(ctx context.Context, address string) ([]Delegation, error) // Stake 2.0
func (c *Client) GetCanDelegatedMaxSize(ctx context.Context, addr string, resource ResourceType) (SUN, error)
func (c *Client) DelegateResource(ctx context.Context, owner, receiver string, resource ResourceType, delegateBalance SUN, lock bool, lockPeriod int64) (*api.TransactionExtention, error)
func (c *Client) ReclaimResource(ctx context.Context, owner, receiver string, resource ResourceType, delegateBalance SUN) (*api.TransactionExtention, error)
func (c *Client) AvailableForDelegateResources(ctx context.Context, addr string) (*AvailableResources, error)
func (c *Client) TotalAvailableResources(ctx context.Context, addr string) (*AvailableResources, error)
func (c *Client) AvailableEnergy(res *api.AccountResourceMessage) decimal.Decimal
// AvailableBandwidth sums the staked and free pools. That sum answers "how much
// bandwidth does this account have" but never "will this transaction be free" —
// see the Billable notes under Estimate operations.
func (c *Client) AvailableBandwidth(res *api.AccountResourceMessage) decimal.Decimal
func (c *Client) AvailableBandwidthWithoutFree(res *api.AccountResourceMessage) decimal.Decimal
func (c *Client) AvailableFreeBandwidth(res *api.AccountResourceMessage) decimal.Decimal
func (c *Client) TotalEnergyLimit(res *api.AccountResourceMessage) decimal.Decimal
func (c *Client) TotalBandwidthLimit(res *api.AccountResourceMessage) decimal.Decimal
```

```go
// Delegation is one active resource delegation, flattened out of the chain's
// nested list-of-lists and converted the way GetStakeInfo converts a stake:
// base58 addresses, SUN amounts, and time.Time expiries out of Unix
// milliseconds. An unlocked delegation has the zero time, not 1970. Records the
// node returns with nothing delegated are dropped.
type Delegation struct {
    From               string    `json:"from"`
    To                 string    `json:"to"`
    Bandwidth          SUN       `json:"bandwidth"`
    BandwidthExpiresAt time.Time `json:"bandwidth_expires_at"`
    Energy             SUN       `json:"energy"`
    EnergyExpiresAt    time.Time `json:"energy_expires_at"`
}
```

`GetCanDelegatedMaxSize` answers in **staked TRX**, not in resource units:
delegation moves the stake, and what that yields depends on the network weights
at the time. Convert with `ConvertStakedToBandwidth` / `ConvertStakedToEnergy`.

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

`AvailableForDelegateResources` caps each field by what the account staked itself (from
`core.Account.FrozenV2`), which is what delegation is allowed to draw on. Note this is a cap on a
*limit*, not on a remainder: `Bandwidth` is `min(staked limit, staked remaining + free remaining)`,
so an account with free bandwidth left can report more staked bandwidth than it actually has. For
"will this specific transaction be free" use `AvailableBandwidthWithoutFree` and
`AvailableFreeBandwidth`, or let `EstimateTRXTransfer` answer it.

### Resource pricing helpers

**File:** `converter.go`

Convert between staked TRX and the resources that stake yields, at the network's current weights
(`GetAccountResource` supplies `TotalNetWeight` / `TotalNetLimit` / `TotalEnergyWeight`, and
`ChainParams` supplies `TotalEnergyCurrentLimit`). The rates move with total network stake, so
these are point-in-time answers, not constants.

```go
func (c *Client) ConvertStakedToEnergy(totalEnergyCurrentLimit, totalEnergyWeight int64, staked SUN) decimal.Decimal
func (c *Client) ConvertEnergyToStaked(totalEnergyCurrentLimit, totalEnergyWeight int64, energy decimal.Decimal) SUN
func (c *Client) ConvertStakedToBandwidth(totalNetWeight, totalNetLimit int64, staked SUN) decimal.Decimal
func (c *Client) ConvertBandwidthToStaked(totalNetWeight, totalNetLimit int64, bandwidth decimal.Decimal) SUN
```

The two `…ToStaked` directions round up via `units.CeilToSUN`, so they answer "how much must I
stake", never "how much might be enough". All four return zero rather than an error when the
divisor the network supplies is zero, so check the weights yourself if a silent zero would be
mistaken for a real answer.

### Staking operations

Tron Stake 2.0. All amounts are in SUN. Legacy Stake 1.0 (FreezeBalance/UnfreezeBalance) is
deliberately not implemented.

**File:** `staking.go`

```go
func (c *Client) Stake(ctx context.Context, owner string, resource ResourceType, amount SUN) (*api.TransactionExtention, error)
func (c *Client) Unstake(ctx context.Context, owner string, resource ResourceType, amount SUN) (*api.TransactionExtention, error)
func (c *Client) WithdrawUnstaked(ctx context.Context, owner string) (*api.TransactionExtention, error)
func (c *Client) CancelAllUnstakes(ctx context.Context, owner string) (*api.TransactionExtention, error)
func (c *Client) GetAvailableUnstakeCount(ctx context.Context, owner string) (int64, error)
func (c *Client) GetWithdrawableUnstaked(ctx context.Context, owner string) (SUN, error)
func (c *Client) GetStakeInfo(ctx context.Context, addr string) (*StakeInfo, error)
```

`GetStakeInfo` reads `core.Account.FrozenV2` / `UnfrozenV2` in a single `GetAccount` call.
`TRON_POWER` entries in `FrozenV2` are always present and are skipped — summing them would double
the total. `UnfreezeExpireTime` is in Unix **milliseconds**.

`GetWithdrawableUnstaked` is the node's answer (`GetCanWithdrawUnfreezeAmount` with the current
timestamp); `StakeInfo.WithdrawableNow` is the same figure computed locally without a second
round-trip.

```go
// PendingUnstake is one in-flight unstake entry.
type PendingUnstake struct {
    Resource   ResourceType `json:"resource"`
    Amount     SUN          `json:"amount"`
    ExpireTime time.Time    `json:"expire_time"`
}

// StakeInfo aggregates an account's Stake 2.0 position. All amounts in SUN.
type StakeInfo struct {
    StakedBandwidth SUN              `json:"staked_bandwidth"`
    StakedEnergy    SUN              `json:"staked_energy"`
    TotalStaked     SUN              `json:"total_staked"`
    UnstakingTotal  SUN              `json:"unstaking_total"`
    WithdrawableNow SUN              `json:"withdrawable_now"`
    PendingUnstakes []PendingUnstake `json:"pending_unstakes"`
}
```

### Witness and reward operations

**File:** `witness.go`

```go
func (c *Client) VoteWitnesses(ctx context.Context, owner string, votes []Vote) (*api.TransactionExtention, error)
func (c *Client) ClaimRewards(ctx context.Context, owner string) (*api.TransactionExtention, error)
func (c *Client) ListWitnesses(ctx context.Context) (*api.WitnessList, error)
func (c *Client) GetUnclaimedReward(ctx context.Context, addr string) (SUN, error)
func (c *Client) GetWitnessBrokerage(ctx context.Context, witness string) (int64, error)
```

`VoteWitnesses` replaces the account's **entire** vote set, so always pass the full desired list.
`[]Vote` is a slice rather than a map so the resulting transaction bytes are reproducible.

```go
// Vote counts are in TRON POWER (1 per staked TRX), not SUN.
type Vote struct {
    WitnessAddress string `json:"witness_address"`
    Count          int64  `json:"count"`
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
// for a contract call.
func (c *Client) EstimateEnergy(
    ctx context.Context,
    from, contractAddress, method, jsonString string,
    tAmount int64, tTokenID string, tTokenAmount int64,
) (*api.EstimateEnergyMessage, error)
```

**`EstimateEnergy` needs a node that opted in** (`vm.estimateEnergy = true`), and the public ones
have not: `tron-rpc.publicnode.com` answers `CONTRACT_VALIDATE_ERROR: this node does not support
estimate energy`, which the client surfaces as an error. The portable way to price a contract call
is `TriggerConstantContract` / `TriggerConstantContractCustom` and its `GetEnergyUsed()` — that is
what `EstimateTRC20Transfer` uses.

**File:** `estimate_transfer.go`

```go
// TRX and TRC20 estimates are separate entry points because their amounts sit
// on different scales — a single function would take one parameter meaning two
// different things depending on the asset. Both answer in three separable parts:
//   - Usage:     what the transaction consumes
//   - Available: what the sender already has to meet that
//   - Charges:   the TRX charges left over, itemised
//   - Fee:       the sum of Charges — what actually leaves the account
//
// Both entry points issue one GetAccountResource for the sender.
func (c *Client) EstimateTRXTransfer(
    ctx context.Context,
    fromAddress, toAddress string,
    amount SUN,
) (*EstimateTransferResult, error)

func (c *Client) EstimateTRC20Transfer(
    ctx context.Context,
    fromAddress, toAddress, contractAddress string,
    amount TokenAmount,
) (*EstimateTransferResult, error)
```

```go
type EstimateTransferResult struct {
    Usage     ResourceUsage   `json:"usage"`     // Bandwidth, Energy, ContractEnergy
    Available SenderResources `json:"available"` // FreeBandwidth, StakedBandwidth, StakedEnergy
    Charges   TransferCharges `json:"charges"`   // Bandwidth, Energy, AccountCreation, UnstakedCreation
    Fee       SUN             `json:"fee"`       // == Charges.Total()
}

func (c TransferCharges) Total() SUN

// Usage.Energy is the whole call (receipt.energy_usage_total); ContractEnergy is
// the part the contract's owner absorbs (receipt.origin_energy_usage). Only the
// difference is the sender's to pay.
func (u ResourceUsage) SenderEnergy() decimal.Decimal
```

A real answer (Nile, sender with 600 free bandwidth and no stake, recipient not created):

```text
Usage:     bandwidth=267 energy=0
Available: free=600 staked=0 energy=0
Charges:   bandwidth=0 energy=0 accountCreation=1 TRX unstakedCreation=0.1 TRX
Fee:       1.1 TRX
```

There is deliberately **no aggregate "if the account had no resources" figure.** The previous
shape carried one (`Total`) and it was both derived and wrong — it summed the transfer transaction
with a phantom second CreateAccount transaction that never reaches the chain, reporting 1.367 TRX
where 1.1 is charged. Callers who want a worst case multiply `Usage` by the chain fees themselves.

**Charges follow the chain's two different resource rules — do not simplify either to
`needed - available`.**

- **Bandwidth is all-or-nothing per pool.** Tron charges the staked pool for the whole
  transaction, else the free daily allowance for the whole transaction, else burns TRX for every
  byte. The pools never combine and there is no "pay the shortfall": an account with 400 free and
  400 staked pays in full for a 500-byte transfer. Verified over 575 mainnet transactions — not one
  had both `receipt.net_usage` and `receipt.net_fee` non-zero.
- **A transaction that creates an account is never billed by the byte at all.** java-tron routes it
  through `consumeForCreateNewAccount`, which tries the *staked* pool alone and otherwise charges a
  flat `getCreateAccountFee` — the free allowance is not consulted on that path. So the bandwidth
  charge and the creation charge are mutually exclusive, and `Charges.Bandwidth` is always zero
  when `Charges.AccountCreation` is set. Verified over 12 account-creating transfers: every sender
  with `NetLimit 0` paid exactly 100000 SUN of `net_fee` while holding hundreds of unused free
  bandwidth, every sender with staked bandwidth paid nothing beyond the 1 TRX, and none paid a
  per-byte amount. (Beware when sampling this yourself: multi-signature transfers also carry a
  1 TRX `getMultiSignFee` and look like account creations if you filter on `fee >= 1 TRX`.)
- **Energy is additive.** The staked pool covers what it can and the remainder is bought by
  burning TRX, so the shortfall is what is billed. In the same 575 transactions, `energy_usage` and
  `energy_fee` were non-zero together six times.
- **Account creation is charged whole**, never reduced by the allowance: Tron does not let the
  free daily quota pay for creating an account. It is reported as two fields because the two chain
  parameters behind it apply independently — `AccountCreation` whenever the recipient does not
  exist, `UnstakedCreation` only when the sender also lacks the staked bandwidth to cover it.
- **The creation fees belong to system contracts only.** There are two ways an account comes into
  existence and they are billed completely differently:

  | how the account is created | cost | billed as |
  | --- | --- | --- |
  | TRX transfer (`TransferContract`) to a new address | 1 TRX, +0.1 TRX without staked bandwidth | **TRX fee** |
  | a contract call that creates it | 25000 energy (`NEW_ACCT_CALL`) | **energy** |
  | TRC20 `transfer()` | nothing — no account is created | — |

  So `EstimateTRC20Transfer` adds **no** creation fee for any recipient state, and that is not a
  special case for TRC20: whatever a contract does about the account is already inside the energy
  the constant call measured, because it runs against the real recipient. Adding a fee on top would
  double-count it.

  Measured on mainnet — the activator contract `TQuCVz7ZXMwcuT2ERcBYCZzLeNAZofcTgY` reports 8227
  energy for an existing target and 33227 for a fresh one (the 25000 gap is the account), and its
  receipt for tx `c0c64a67…4535e0` shows `energy_usage 33227`, `net_usage 313` and no fee at all.
  A plain USDT transfer meanwhile costs 130285 energy to an address with no balance against 64285
  to one that holds some — that 66000 is the new storage slot, identical whether the recipient has
  no balance or no account, so none of it is account creation. Mainnet is full of addresses holding
  USDT for which `getaccount` answers `{}`.

`AvailableBandwidth` sums the two pools and is therefore the wrong input here; use
`AvailableBandwidthWithoutFree` and `AvailableFreeBandwidth` separately.

**A contract can pay for its own calls, and the estimate must credit that.**
`consume_user_resource_percent` is the share of every call the *caller* pays; the contract's owner
covers the rest from its own staked energy, capped per call by `origin_energy_limit`, and only what
the owner cannot cover falls back to the caller (java-tron, `ReceiptCapsule.payEnergyBill`). Calling
a contract you own is a self-call and is billed to you in full. `Usage.Energy` stays the whole call
(`receipt.energy_usage_total`), `Usage.ContractEnergy` is the owner's share
(`receipt.origin_energy_usage`), and `SenderEnergy()` is the difference — the only part that can
become a fee. Assuming the caller pays everything is how an estimate quotes 1.465 TRX for a transfer
the chain settles for nothing: Nile tx `e3e1633c…` has `energy_usage_total 14650`,
`origin_energy_usage 14650`, `energy_fee 0`, `fee 0`. Establishing the split costs
`EstimateTRC20Transfer` two extra RPCs (`GetContract` plus the owner's `GetAccountResource`), because
it depends on the contract's settings and the owner's balance right now.

**`EstimateTRC20Transfer` fails for a transfer that would revert** — insufficient token balance
being the usual reason — because it measures energy with a constant call, and a reverted call
reports only what the VM burned before giving up. The error wraps `ErrContractCallFailed`. This is
deliberate: the alternative is an estimate an order of magnitude too low (8624 against 64285 for
USDT), which a caller then sets as a fee limit on a transfer that runs out of energy.

### Contract operations

**File:** `contract.go`

Every write method returns an **unsigned** transaction. Sign it with `SignTransaction` and send it
with `BroadcastTransaction`; nothing here broadcasts on its own.

```go
// Deploying.
type DeployContractRequest struct {
    From                       string                  // deploying account, base58check
    Name                       string                  // stored on-chain; metadata only
    ABI                        *core.SmartContract_ABI // build with abi.LoadContractABI
    Bytecode                   string                  // compiled contract as hex, 0x optional
    ConstructorParams          string                  // abi jsonString; "" when there are none
    FeeLimit                   SUN                     // 0 = leave unset (the node's default is low)
    ConsumeUserResourcePercent int64                   // 0..100
    OriginEnergyLimit          int64                   // must be > 0
}

func (r DeployContractRequest) Validate() error
func (c *Client) DeployContract(ctx context.Context, req DeployContractRequest) (*api.TransactionExtention, error)

// The address the deployment will occupy, derived locally from the transaction.
func DeployedContractAddress(tx *core.Transaction) (string, error)

// Calling. method is the full signature ("transfer(address,uint256)"); jsonString
// is the argument list in the pkg/client/abi form (see below).
func (c *Client) TriggerContract(
    ctx context.Context,
    from, contractAddress, method, jsonString string,
    feeLimit SUN, tAmount int64, tTokenID string, tTokenAmount int64,
) (*api.TransactionExtention, error)

// Read-only calls. Nothing is broadcast; the answer is in GetConstantResult(),
// the metered cost in GetEnergyUsed(). A call the VM refused returns an error
// wrapping ErrContractCallFailed, with the extention alongside it.
func (c *Client) TriggerConstantContract(ctx context.Context, ct *core.TriggerSmartContract) (*api.TransactionExtention, error)
func (c *Client) TriggerConstantContractCustom(ctx context.Context, from, contractAddress, method, jsonString string) (*api.TransactionExtention, error)

// Introspection.
func (c *Client) GetContract(ctx context.Context, contractAddress string) (*core.SmartContract, error)
func (c *Client) GetContractABI(ctx context.Context, contractAddress string) (*core.SmartContract_ABI, error)

// Owner-only settings on an already-deployed contract.
func (c *Client) UpdateSettingContract(ctx context.Context, from, contractAddress string, value int64) (*api.TransactionExtention, error)     // consume_user_resource_percent
func (c *Client) UpdateEnergyLimitContract(ctx context.Context, from, contractAddress string, value int64) (*api.TransactionExtention, error) // origin_energy_limit

// Recompute txID after editing RawData locally. DeployContract and
// TriggerContract already call it when they set a fee limit.
func (c *Client) UpdateHash(tx *api.TransactionExtention) error
```

`tAmount` on `TriggerContract` is the TRX call value in SUN; `tTokenID` / `tTokenAmount` attach a
TRC10 token and are both required for either to take effect.

`TriggerConstantContractCustom` accepts an empty `from` and substitutes the zero address, which is
what makes it usable for reads from an account you do not control.

**Deploying, end to end:**

```go
contractABI, err := abi.LoadContractABI(solcABIJSON)   // solc's array or Tron's {"entrys":…}

tx, err := c.DeployContract(ctx, client.DeployContractRequest{
    From:                       wallet,
    Name:                       "Token",
    ABI:                        contractABI,
    Bytecode:                   solcBytecodeHex,
    ConstructorParams:          `[{"uint256":"1000000"},{"address":"T..."}]`,
    FeeLimit:                   client.MustFromTRX(decimal.NewFromInt(1000)),
    ConsumeUserResourcePercent: 100,
    OriginEnergyLimit:          10_000_000,
})

addr, err := client.DeployedContractAddress(tx.GetTransaction()) // known before broadcasting

err = c.SignTransaction(tx.GetTransaction(), priv)
_, err = c.BroadcastTransaction(ctx, tx.GetTransaction())
```

- **Constructor arguments belong in `ConstructorParams`, not appended to `Bytecode`.** Tron has no
  field for them: they are ABI-encoded and concatenated onto the bytecode, which `DeployContract`
  does. Appending them yourself as well encodes them twice.
- **`Validate` covers the whole request**, constructor arguments included, and `DeployContract`
  runs it before spending a round-trip. `Bytecode` must have an even number of hex digits: an odd
  count means the string is truncated, and `tronutils.FromHex` would left-pad it with a zero nibble,
  shifting every byte by half a byte into code that means something else.
- **`DeployedContractAddress` derives the address rather than reading it back.** Tron computes it as
  `keccak256(txID ‖ ownerAddress)` keeping the low 20 bytes, with the `0x41` prefix put back, so it
  is fixed the moment the transaction is built — no need to wait for the receipt. It is pinned in
  the tests against two live mainnet contracts (USDT and `TQuCVz7…`) and their creation transactions.
  **Read it after the last edit to `raw_data`**: the fee limit is inside `raw_data`, so setting it
  changes the txID and therefore the address. `DeployContract` sets the fee limit before returning,
  so the transaction it hands back is already final.
- **`FeeLimit` matters.** A deployment left at the node's default routinely runs out of energy
  mid-construction, which still burns what it used.
- **The receipt is still the authority after the fact:**
  `GetTransactionInfoByHash(…).GetContractAddress()` returns the raw bytes;
  `tronutils.EncodeCheck` turns them into the `T…` form.

**A reverted constant call is not reported by the result code.** The node answers
`result.result = true` with code `SUCCESS` and puts the failure only in `result.message`
(`"REVERT opcode executed"`), leaving `constant_result` empty and `energy_used` at whatever the VM
burned before giving up — 8624 against 64285 for a real USDT transfer. `TriggerConstantContract`
therefore keys on the message, not the code, and returns `ErrContractCallFailed` with the extention
still attached so the partial evidence is readable. Anything that trusted the code alone priced the
pre-revert energy as the cost of the whole call.

### Asset operations (TRC10)

**File:** `asset.go`

```go
func (c *Client) GetAssetIssueById(ctx context.Context, id string) (*core.AssetIssueContract, error)
func (c *Client) GetAssetIssueListByName(ctx context.Context, name string) (*api.AssetIssueList, error)
```

Read-only TRC10 lookups. TRC10 is Tron's native token standard and is unrelated to TRC20, which is
contract-based — see [TRC20 token operations](#trc20-token-operations) for those.

### Network operations

**File:** `network.go`

```go
func (c *Client) ListNodes(ctx context.Context) (*api.NodeList, error)
func (c *Client) GetNodeInfo(ctx context.Context) (*core.NodeInfo, error)
func (c *Client) GetNextMaintenanceTime(ctx context.Context) (*api.NumberMessage, error)
func (c *Client) TotalTransaction(ctx context.Context) (*api.NumberMessage, error)
```

`GetNodeInfo` reports the node that actually served the call, so under the health-aware transport
it names whichever node the tier logic picked — it is not pinned to a configured address.

### Chain parameters

**File:** `chain_params.go`

```go
func (c *Client) ChainParams(ctx context.Context) (*ChainParams, error)
func (c *Client) ChainParam(ctx context.Context, paramKey string) (*core.ChainParameters_ChainParameter, error)
```

### Transport stack (low-level)

The `Transport` interface (`pkg/client/transport.go`) is the abstraction
implemented by every per-node and wrapping transport. The wrappers callers
typically interact with from outside `pkg/client`:

```go
// Default in production: tier-based fallback + per-node health checking.
func NewHealthAwareTransport(
    nodes []NodeConfig,
    factory func(NodeConfig) (Transport, error),
    cfg HealthConfig,
    metrics MetricsCollector,
    blockchain string,
) (*HealthAwareTransport, error)

// Legacy: plain round-robin, no health. Used when cfg.Health.Disabled = true.
func NewRoundRobinTransport(transports []Transport) *RoundRobinTransport

// Wraps any Transport to record latency/status via MetricsCollector.
func NewMetricsTransport(transport Transport, metrics MetricsCollector, blockchain string) *MetricsTransport
```

`client.New(cfg)` builds the stack automatically — direct use of the
constructors is only needed for advanced cases (custom transport graphs in
tests, embedding gotron in a higher-level multi-chain harness, etc.).

### Common types

**File:** `types.go`

```go
// EstimateResult is the result shape of the two activation estimators only —
// Energy and Bandwidth in raw resource points, Fee in SUN. Transfer estimates do
// not use it: EstimateTransferResult keeps what a transaction needs apart from
// what it is charged, which one Energy/Bandwidth/Fee triple cannot express.
type EstimateResult struct {
    Energy    decimal.Decimal `json:"energy"`
    Bandwidth decimal.Decimal `json:"bandwidth"`
    Fee       SUN             `json:"fee"`
}
```

```go
// Network is informational — it labels the client, it does not select endpoints.
// Which chain you talk to is decided entirely by Config.Nodes.
type Network string

const (
    NetworkMainnet Network = "mainnet"
    NetworkShasta  Network = "shasta"
    NetworkNile    Network = "nile"
)

func (n Network) Validate() error
func (n Network) String() string
```

```go
// ResourceType is the caller-facing form of core.ResourceCode, used by the
// delegation and staking methods.
type ResourceType int32

const (
    ResourceTypeBandwidth ResourceType = 0
    ResourceTypeEnergy    ResourceType = 1
)

func (r ResourceType) Validate() error          // ErrInvalidResourceType for anything else
func (r ResourceType) String() string           // "BANDWIDTH" / "ENERGY" / "UNKNOWN"
func (r ResourceType) ToProto() core.ResourceCode // -1 for an invalid value, so validate first
```

**File:** `chain_params.go`

```go
// ChainParams is the subset of getchainparameters this SDK prices with. Fees are
// per unit in SUN; the two creation fees are flat amounts in SUN.
type ChainParams struct {
    EnergyFee                           int64 // SUN per energy unit
    TransactionFee                      int64 // SUN per bandwidth point
    TotalEnergyCurrentLimit             int64
    FreeNetLimit                        int64
    CreateNewAccountFeeInSystemContract int64 // 1 TRX on mainnet
    CreateAccountFee                    int64 // 0.1 TRX on mainnet
}
```

**File:** `constants.go`

```go
const (
    TrxDecimals        = 6
    TrxAssetIdentifier = "trx"
)
```

### Sentinel errors

**File:** `errors.go`

Match these with `errors.Is`; the package wraps them with `fmt.Errorf("%w: …")` rather than
returning bare strings.

```go
// Configuration and connectivity
ErrInvalidConfig, ErrNotConnected, ErrInvalidParams, ErrNilResponse

// Addresses
ErrInvalidAddress, ErrEmptyAddress, ErrAccountNotActivated

// Transactions and amounts
ErrInvalidAmount           // == units.ErrInvalidAmount, so one check covers both layers
ErrInvalidTransaction, ErrInvalidPrivateKey
ErrTransactionNotFound, ErrTransactionInfoNotFound

// Resources
ErrInvalidResourceType

// Contracts
ErrContractCallFailed      // a constant call the VM refused, most often a revert

// Transport
ErrNoHealthyNodes          // every node in every tier is currently unhealthy; retry with backoff
```

Three error types carry structured detail and unwrap to the cause:

```go
type TransportError struct { Host, Protocol, Method string; Err error }
// ContractValidateError is a node refusing to *build* a transaction because the
// request is wrong - not because the node is unwell, so it never counts toward
// node health and retrying elsewhere gives the same answer. Both transports
// produce it (gRPC through Result.Code, HTTP through the "Error" field or a
// nested result object), and it unwraps to ErrInvalidTransaction so the older
// sentinel check still matches.
type ContractValidateError struct { Code api.ReturnResponseCode; Message string }
type HTTPStatusError struct { Code int; Body string }
type BroadcastError  struct { Code api.ReturnResponseCode; Message string }
```

---

## Package pkg/client/abi

**File:** `abi/abi.go`

Argument encoding for contract calls. `TriggerContract`, `TriggerConstantContractCustom` and
`EstimateEnergy` use it internally; reach for it directly when you need the calldata itself.

```go
// LoadContractABI turns a contract's ABI into the protobuf a deployment needs.
// It takes both shapes that occur: solc's top-level array and Tron's
// {"entrys": [...]} envelope, which is what /wallet/getcontract returns.
// protojson handles neither - an array has no message to unmarshal into, and the
// enums arrive lowercase against capitalised proto names.
func LoadContractABI(jString string) (*core.SmartContract_ABI, error)

// Param is one argument as a single-key map from Solidity type to value, which
// is why the JSON form is an array of one-field objects rather than a plain list.
type Param map[string]any

func LoadFromJSON(jString string) ([]Param, error)
func Pack(method string, param []Param) ([]byte, error)      // selector + arguments
func GetPaddedParam(param []Param) ([]byte, error)           // arguments only, no selector
func Signature(method string) []byte                         // first 4 bytes = the selector
func GetParser(ABI *core.SmartContract_ABI, method string) (eABI.Arguments, error)       // outputs
func GetInputsParser(ABI *core.SmartContract_ABI, method string) (eABI.Arguments, error) // inputs
```

The `jsonString` every contract method takes is what `LoadFromJSON` parses:

```json
[{"address":"TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"},{"uint256":"1000000"}]
```

Addresses go in base58 (`T…`) — they are decoded with `tronutils.DecodeCheck`, so a hex or bare
address is an error. Integers go as **strings**: a JSON number is rejected outright with
`invalid integer`, never silently rounded, because a `uint256` does not survive a float. Types
wider than 64 bits also accept a `0x`-prefixed hex string. `method` is always the full signature
with types and no spaces: `"transfer(address,uint256)"`.

`GetPaddedParam` is what `DeployContract` uses to encode `ConstructorParams` before appending them
to the bytecode; call it directly only if you are building a `CreateSmartContract` by hand.

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
func (g *Generator) Generate(index uint32) (*Address, error)
```

Default BIP44 path: `m/44'/195'/0'/0/{index}`. The HD key versions are fixed to
mainnet — there is no network selector.

---

## Package pkg/units

**File:** `units.go`

Amount types. Tron has two unrelated amount scales, so each gets its own type and the unit is part
of every signature rather than of a doc comment.

```go
const SunPerTRX = 1_000_000

// ErrInvalidAmount marks every amount the constructors refuse to build.
// client.ErrInvalidAmount and gotron.ErrInvalidAmount are this same variable,
// so one errors.Is covers both layers.
var ErrInvalidAmount = errors.New("invalid amount")

// SUN is every TRX-denominated value: transfers, balances, stake, fee limits, rewards.
type SUN int64

func FromTRX(trx decimal.Decimal) (SUN, error)  // rejects sub-SUN precision and out-of-int64
func MustFromTRX(trx decimal.Decimal) SUN       // panics; constants and tests only
func (s SUN) TRX() decimal.Decimal
func (s SUN) Int64() int64
func (s SUN) String() string                    // e.g. "1.5 TRX"

// TokenAmount is every TRC20 amount, in the token's own minimal units.
// The zero value is a valid zero amount.
type TokenAmount struct{ /* ... */ }

func FromTokenUnits(units *big.Int) (TokenAmount, error)                       // rejects nil, negative, >32 bytes
func FromTokenDecimal(amount decimal.Decimal, decimals int32) (TokenAmount, error) // rejects finer-than-decimals, decimals outside 0..78
func (a TokenAmount) TokenUnits() *big.Int      // copy
func (a TokenAmount) Decimal(decimals int32) decimal.Decimal
func (a TokenAmount) IsZero() bool
func (a TokenAmount) IsPositive() bool
```

Both types are re-exported as `client.SUN` / `client.TokenAmount` (and `gotron.*`) along with their
constructors, so building an amount needs no extra import.

Resource pricing helpers. `getEnergyFee` and `getTransactionFee` are quoted in SUN per unit, so
both conversions multiply:

```go
func NewEnergy(value decimal.Decimal) Energy
func (e Energy) ToSUN(energyFee int64) SUN        // rounds up
func NewBandwidth(value decimal.Decimal) Bandwidth
func (b Bandwidth) ToSUN(transactionFee int64) SUN // rounds up

// CeilToSUN is what those helpers (and the client's Convert*ToStaked pair) use
// to land on int64: it rounds up and saturates at the int64 bounds. Never reach
// for decimal.IntPart here — it keeps only the low 64 bits, so an impossible
// amount comes back as a plausible, often negative, one.
func CeilToSUN(d decimal.Decimal) SUN
```

---

## Package pkg/tronutils

**Files:** `address.go`, `base58.go`, `hash.go`, `hexutils.go`

**Address utilities** (`address.go`, `base58.go`):

```go
type Address []byte

func DecodeCheck(input string) ([]byte, error)  // base58 -> bytes (with checksum verify)
func EncodeCheck(input []byte) string           // bytes -> base58 (with checksum)
func Decode(input string) ([]byte, error)       // base58, no checksum
func Encode(input []byte) string                // base58, no checksum
func Base58ToAddress(s string) (Address, error)
func Base64ToAddress(s string) (Address, error)
func HexToAddress(s string) Address
func BigToAddress(b *big.Int) Address
func PubkeyToAddress(p ecdsa.PublicKey) Address
```

**Hex utilities** (`hexutils.go`):

```go
func BytesToHexString(bytes []byte) string      // "0x..." prefixed
func HexStringToBytes(input string) ([]byte, error)
func ToHex(b []byte) string
func ToHexArray(b [][]byte) []string
func FromHex(s string) ([]byte, error)          // supports "0x" prefix
func Has0xPrefix(str string) bool
func IsHex(str string) bool
func Bytes2Hex(d []byte) string                 // no prefix
func Hex2Bytes(str string) ([]byte, error)      // no prefix
func Hex2BytesFixed(str string, flen int) []byte
func LeftPadBytes(slice []byte, l int) []byte
func RightPadBytes(slice []byte, l int) []byte
func TrimLeftZeroes(s []byte) []byte
func CopyBytes(b []byte) []byte
```

**Hash utilities** (`hash.go`):

```go
type Hash [32]byte

func Keccak256(msg []byte) []byte
func BytesToHash(b []byte) Hash
func BigToHash(b *big.Int) Hash
func HexToHash(s string) (Hash, error)
```
