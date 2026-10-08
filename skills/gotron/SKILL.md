---
name: gotron
description: >
  How to integrate github.com/sxwebdev/gotron, the Go SDK for the Tron blockchain, into an
  application: connecting to nodes (gRPC or HTTP, TronGrid API keys, failover tiers, health checks,
  Prometheus metrics), generating and validating addresses, sending TRX and TRC20 tokens (USDT),
  signing and broadcasting, waiting for confirmation, estimating fees and fee limits, Stake 2.0,
  energy/bandwidth delegation, voting and rewards, multisig/active permissions, and deploying or
  calling smart contracts. Use this skill whenever Go code imports gotron
  (`github.com/sxwebdev/gotron`, `pkg/client`, `pkg/address`, `pkg/units`, `pkg/client/abi`) or the
  user wants Go code that talks to Tron — TRX, SUN, TRC20, USDT on Tron, TronGrid, energy,
  bandwidth, staking, a Tron wallet or payment-processing backend — even if gotron is not named.
user-invocable: true
---

# Integrating gotron

gotron is a Go client for Tron nodes. It speaks gRPC or HTTP to one or many nodes, routes around
unhealthy ones, and exposes a typed API: base58 address strings, amounts whose unit is part of the
type, and unsigned transactions that you sign and broadcast yourself.

This skill is for **using** the library. If you are changing gotron itself, read `docs/` in the
repository instead (`docs/architecture.md`, `docs/transport.md`, `docs/testing.md`).

Read the reference files when you need detail beyond the recipes here:

- `references/api-surface.md` — every public type and method, with the chain behaviour behind
  fees, delegation and contract calls. Look a signature up there before guessing it.
- `references/constants.md` — sentinel errors, error types, enums, permission ids, metric names.

## Install

```bash
go get github.com/sxwebdev/gotron@latest
```

Packages you will import:

| Package                                                          | For                                                                            |
| ---------------------------------------------------------------- | ------------------------------------------------------------------------------ |
| `github.com/sxwebdev/gotron/pkg/client`                          | the client: every chain operation, `Config`, errors, `SUN`, `TokenAmount`      |
| `github.com/sxwebdev/gotron/pkg/address`                         | key generation, mnemonics, address validation                                  |
| `github.com/sxwebdev/gotron/pkg/client/abi`                      | ABI loading and encoding for contract calls and deployments                    |
| `github.com/sxwebdev/gotron/schema/pb/core`, `.../schema/pb/api` | protobuf types in results (`core.TransactionInfo`, `api.TransactionExtention`) |
| `github.com/sxwebdev/gotron`                                     | optional thin wrapper: `gotron.New`, aliases `gotron.SUN`, `gotron.Energy`, …  |

## Connect

```go
c, err := client.New(client.Config{
    Nodes: []client.NodeConfig{
        {Protocol: client.ProtocolGRPC, Address: "grpc.trongrid.io:50051", UseTLS: true,
            Headers: map[string]string{"TRON-PRO-API-KEY": apiKey}},
    },
})
if err != nil { return err }
defer c.Close()
```

- **HTTP** works the same: `{Protocol: client.ProtocolHTTP, Address: "https://api.trongrid.io"}`.
  Both transports return the same data for every method; pick whichever your provider offers.
  gRPC is cheaper for block-heavy work. The few remaining differences are under
  [Choosing a transport](#choosing-a-transport).
- **Several nodes** are load-balanced. `Tier` makes fallbacks: requests go to the lowest tier that
  has a healthy node, and return to it as soon as it recovers.

  ```go
  Nodes: []client.NodeConfig{
      {Protocol: client.ProtocolGRPC, Address: "my-node:50051", Tier: 0},                // primary
      {Protocol: client.ProtocolHTTP, Address: "https://api.trongrid.io", Tier: 1,
          Headers: map[string]string{"TRON-PRO-API-KEY": apiKey}},                     // fallback
  },
  ```

- **Health checking** is on by default (background probe per node). Tune it with
  `Config.Health` (`FailureThreshold`, `HealthyInterval`, `ProbeTimeout`, `Logger`, …); the zero
  value is sensible. Only network failures mark a node unhealthy — a refused request does not.
- **There is no automatic retry.** A failed call returns its error; the next call simply avoids the
  failing node. Retry in your code where it is safe (reads always; broadcasts — see
  [Errors](#errors)).
- **Metrics:** `Config.Metrics = client.NewMetrics(prometheus.DefaultRegisterer)` records request
  counts, latency and pool health (`gotron_rpc_*`).
- `Config.Network` is a label only. Which chain you are on is decided by the nodes you list.

## The three rules

**1. Addresses are base58check strings** (`T…`). Validate user input with
`address.Validate(addr)`. Methods return `client.ErrInvalidAddress` for anything malformed.

**2. Amounts are typed, and the two types are not interchangeable.**

- `client.SUN` is every TRX-denominated value: transfers, balances, stakes, fee limits, fees.
  1 TRX = 1,000,000 SUN. Build one with `client.FromTRX(decimal)` (errors on more than 6 decimal
  places) or `client.SUN(n)` for a known integer of SUN; read it back with `.TRX()`.
- `client.TokenAmount` is every TRC20 amount, in the token's own minimal units. Build it with
  `client.FromTokenDecimal(decimal, decimals)` using the decimals the contract reports, or
  `client.FromTokenUnits(*big.Int)`; render it with `.Decimal(decimals)`.
- One USDT is `TokenAmount` 1,000,000; one TRX is `SUN` 1,000,000. The compiler keeps them apart —
  do not convert one into the other through `int64`.
- Energy, bandwidth, percentages and vote counts are plain numbers, not money.

**3. Every write is build → sign → broadcast → confirm.** Write methods (`CreateTransferTransaction`,
`TRC20Send`, `Stake`, `DeployContract`, …) return an **unsigned** `*api.TransactionExtention` and
touch nothing on chain. You sign `ext.Transaction`, broadcast it, then wait for its receipt.
If you edit `ext.Transaction.RawData` after building (fee limit, expiration), call
`ext.UpdateHash()` before signing, or the signature covers a txid nobody will find.

## Send TRX

```go
amount, err := client.FromTRX(decimal.RequireFromString("1.5"))
if err != nil { return err }

ext, err := c.CreateTransferTransaction(ctx, from, to, amount)
if err != nil { return err }

signer, err := address.FromPrivateKey(privateKeyHex)
if err != nil { return err }
if err := c.SignTransaction(ext.Transaction, signer.PrivateKeyECDSA); err != nil { return err }

if _, err := c.BroadcastTransaction(ctx, ext.Transaction); err != nil { return err }

txid := hex.EncodeToString(ext.Txid)
info, err := waitReceipt(ctx, c, txid) // below
```

- Sending to an address that does not exist yet creates it; that costs a fee on top
  (1 TRX + 0.1 TRX without staked bandwidth on mainnet). Do not gate payments on
  `IsAccountActivated`.
- Price it first with `c.EstimateTRXTransfer(ctx, from, to, amount)`: `est.Fee` is what will
  actually leave the sender beyond `amount`, and `est.Charges` itemises it.
- When the key's lifetime in memory matters, keep it as 32 raw bytes and sign with
  `c.SignTransactionRaw(tx, key)`, then `clear(key)`.

## Wait for confirmation

A transaction is in a block once its receipt exists. `GetTransactionInfoByHash` returns
`client.ErrTransactionInfoNotFound` until then.

```go
func waitReceipt(ctx context.Context, c *client.Client, txid string) (*core.TransactionInfo, error) {
    tick := time.NewTicker(time.Second)
    defer tick.Stop()
    for {
        info, err := c.GetTransactionInfoByHash(ctx, txid)
        if err == nil {
            return info, nil
        }
        if !errors.Is(err, client.ErrTransactionInfoNotFound) {
            return nil, err
        }
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        case <-tick.C:
        }
    }
}
```

Then check the outcome — a mined transaction can still have failed and been charged:

- `info.GetResult() == core.TransactionInfo_FAILED` (the enum's success value is spelled
  `TransactionInfo_SUCESS`), with the reason in `string(info.GetResMessage())`.
- For contract calls also check `info.GetReceipt().GetResult()`: `REVERT`, `OUT_OF_ENERGY` and the
  like mean the call did nothing but the energy was paid.
- A block is irreversible about 19 blocks later. If a payment must not be rolled back, also wait
  until `c.GetLastBlockHeight(ctx)` is at least `info.GetBlockNumber() + 19`.
- `info.GetFee()` is the total TRX burned, in SUN.

## TRC20 tokens (USDT)

```go
const usdt = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"

decimals, err := c.TRC20GetDecimals(ctx, usdt)                 // *big.Int, 6 for USDT
amount, err := client.FromTokenDecimal(decimal.RequireFromString("25.5"), int32(decimals.Int64()))

balance, err := c.TRC20ContractBalance(ctx, from, usdt)        // TokenAmount
fmt.Println(balance.Decimal(int32(decimals.Int64())))

est, err := c.EstimateTRC20Transfer(ctx, from, to, usdt, amount)
if err != nil { return err } // wraps client.ErrContractCallFailed if the transfer would revert

params, err := c.ChainParams(ctx)
feeLimit := client.SUN(est.Usage.SenderEnergy().
    Mul(decimal.NewFromInt(params.EnergyFee)).
    Mul(decimal.NewFromFloat(1.2)).Ceil().IntPart()) // 20% headroom

ext, err := c.TRC20Send(ctx, from, to, usdt, amount, feeLimit)
// sign, broadcast, waitReceipt as for TRX
```

- **The fee limit caps the call's whole energy, priced in SUN — staked energy counts against it
  too.** Size it from `est.Usage`, not from `est.Charges`, which is only the part that will be
  burned. Too low, and the transaction is mined, fails with `OUT_OF_ENERGY` and is still charged.
- `est.Fee` is what the sender will actually pay given its current staked energy and bandwidth.
- A transfer to an address that never held the token costs about twice the energy (a new storage
  slot). The estimate already measures that against the real recipient; there is no account
  creation fee for TRC20.
- `TRC20Approve` and `TRC20TransferFrom` cover the allowance flow; `TRC20GetName`/`TRC20GetSymbol`
  read metadata.

## Staking, delegation, voting

All amounts are `SUN` of staked TRX; resources are `client.ResourceTypeEnergy` /
`client.ResourceTypeBandwidth`. Each call returns an unsigned transaction.

```go
ext, err := c.Stake(ctx, owner, client.ResourceTypeEnergy, client.MustFromTRX(decimal.NewFromInt(1000)))
ext, err  = c.DelegateResource(ctx, owner, receiver, client.ResourceTypeEnergy, amount, false, 0)
ext, err  = c.ReclaimResource(ctx, owner, receiver, client.ResourceTypeEnergy, amount)
ext, err  = c.Unstake(ctx, owner, client.ResourceTypeEnergy, amount)  // starts the unstake delay
ext, err  = c.WithdrawUnstaked(ctx, owner)                             // after the delay
ext, err  = c.CancelAllUnstakes(ctx, owner)                            // returns pending unstakes to stake
ext, err  = c.VoteWitnesses(ctx, owner, []client.Vote{{WitnessAddress: sr, Count: 1000}})
ext, err  = c.ClaimRewards(ctx, owner)
```

- `c.GetStakeInfo(ctx, addr)` summarises staked, unstaking and withdrawable amounts;
  `c.GetDelegatedResourcesV2(ctx, addr)` lists what an account lends out.
- `GetCanDelegatedMaxSize` answers in staked TRX, not energy. Convert stake to resource units with
  `ConvertStakedToEnergy` / `ConvertStakedToBandwidth` and the current network weights.
- `VoteWitnesses` **replaces** the whole vote set; pass every vote you want to keep. Votes are
  TRON POWER (1 per staked TRX), not SUN.
- A locked delegation (`lock=true`, `lockPeriod` in blocks) cannot be reclaimed until it expires.

## Smart contracts

```go
contractABI, err := abi.LoadContractABI(abiJSON) // solc's array or Tron's {"entrys": [...]}
req := client.DeployContractRequest{
    From:              owner,
    Name:              "MyToken",
    ABI:               contractABI,
    Bytecode:          bytecodeHex,
    ConstructorParams: `[{"uint256":"1000000"},{"string":"My Token"}]`, // never append them yourself
    FeeLimit:          client.MustFromTRX(decimal.NewFromInt(1000)),
    OriginEnergyLimit: 10_000_000,
}
est, err := c.EstimateDeployContract(ctx, req) // price it: a failed deployment is still charged
ext, err := c.DeployContract(ctx, req)
contractAddr, err := client.DeployedContractAddress(ext.Transaction) // known before broadcast
```

- **Calling:** `c.TriggerContract(ctx, from, contract, "transfer(address,uint256)",
`[{"address":"T…"},{"uint256":"5"}]`, feeLimit, callValue, tokenID, tokenAmount)` returns an
  unsigned transaction.
- **Reading:** `c.TriggerConstantContractCustom(ctx, from, contract, "balanceOf(address)", params)`
  runs the call without a transaction; `from` may be empty. Results are in
  `ext.GetConstantResult()`.
- **A revert is reported as an error** wrapping `client.ErrContractCallFailed`. The node itself
  answers "success" with the failure only in a message; the extention is returned next to the
  error so you can still read the partial result.
- `c.GetContract` / `c.GetContractABI` read a deployed contract and its ABI.

## Multisig and active permissions

- `c.GetAccountPermission(ctx, account, id)` and `c.ValidatePermissionSigner(ctx, account, signer,
id, contractTypes...)` check, before you accept a key, that it can actually sign what you need.
- `client.SetPermissionID(ext, 2)` marks an unsigned transaction to be authorised by active
  permission 2. It refreshes the txid itself and refuses a transaction that is already signed.
- Sign once per required key: `c.SignTransaction(ext.Transaction, keyN)`. More than one signature
  costs the multi-sign fee (1 TRX on mainnet), which no estimate includes.
- `c.UpdateAccountPermissions(ctx, req)` replaces the **complete** permission set (100 TRX on
  mainnet). Build permissions with `client.NewOwnerPermission`, `client.NewActivePermission` and
  `client.ContractOperations`; copy every permission you want to keep. An operations bitmap allows
  contract _types_ — it cannot restrict a key to one contract or method.

## Reading the chain

- Accounts: `GetAccount` (an address that was never activated returns `ErrAccountNotFound`),
  `GetAccountBalance` (`SUN`), `GetAccountResource`, `IsAccountActivated`.
- Blocks: `GetLastBlockHeight`, `GetLastBlock`, `GetBlockByHeight`, `GetBlockByHash`,
  `GetBlockByLimitNext2(start, end)` (end exclusive). Each block transaction carries `Txid` and
  `Transaction`; decode the contract with `tx.GetRawData().GetContract()[0].GetParameter().UnmarshalTo(&core.TransferContract{})`.
- Receipts for a whole block: `GetTransactionInfoByBlockNum` — one call instead of one per
  transaction; TRC20 transfers are in each receipt's `Log` (topic 0 is
  `client.Trc20TransferEventSignature`).
- Single transactions: `GetTransactionByHash`, `GetTransactionInfoByHash`,
  `GetTransactionExtensionByHash` (both at once).
- Network: `ChainParams` (current fees — never hard-code mainnet's), `ListWitnesses`,
  `GetNodeInfo`.

## Errors

Match with `errors.Is` / `errors.As`, never on message text: the node words the same refusal
differently over gRPC and HTTP.

| What you get                                                                          | Meaning                                                                                                  | What to do                                                |
| ------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- | --------------------------------------------------------- |
| `*client.ContractValidateError` (also `errors.Is(err, client.ErrInvalidTransaction)`) | the node refused to **build** the transaction: insufficient balance, unknown contract, locked delegation | fix the request; retrying elsewhere gives the same answer |
| `*client.BroadcastError` with `.Code`                                                 | the node refused to **accept** a signed transaction                                                      | see codes below                                           |
| `client.ErrDelegateStakeShort` (on a `*ContractValidateError`)                        | `DelegateResource` for more than the owner's stake of the resource now holds                             | delegate less, or stake more                              |
| `client.ErrDelegateBelowMinimum` (on a `*ContractValidateError`)                      | `DelegateResource` for under `client.MinDelegateBalance` (1 TRX); refused before any RPC                 | delegate at least the minimum                             |
| `client.ErrContractCallFailed`                                                        | a constant call or estimate reverted                                                                     | the real transaction would revert too                     |
| `client.ErrTransactionInfoNotFound`                                                   | not in a block yet                                                                                       | keep polling                                              |
| `client.ErrAccountNotFound`                                                           | address never activated                                                                                  | expected for new addresses                                |
| `client.ErrInvalidAddress`, `client.ErrInvalidAmount`                                 | bad input, caught before any RPC                                                                         | validate earlier                                          |
| `client.ErrNoHealthyNodes`                                                            | every node is down right now                                                                             | back off and retry                                        |
| `client.ErrNodeRefusedRequest`                                                        | an HTTP node rejected a malformed read                                                                   | fix the request                                           |
| `*client.TransportError`                                                              | network failure; `.Host`, `.Protocol`, `.Method` say where                                               | retry reads                                               |

Broadcast codes worth handling (`api.Return_*`):

- `DUP_TRANSACTION_ERROR` — the network already has this exact transaction. When you re-broadcast
  after a timeout, treat it as success and wait for the receipt.
- `TRANSACTION_EXPIRATION_ERROR`, `TAPOS_ERROR` — the transaction is stale. Build a new one; do not
  re-sign the old one.
- `SIGERROR` — wrong key, wrong permission id, or a txid that no longer matches `raw_data`.
- `BANDWITH_ERROR` (sic) — not enough TRX to pay for bandwidth.

## Choosing a transport

gRPC and HTTP return identical data for every method; the library tests each one against the
other on the same node. What still differs comes from the Tron node itself:

- A refusal to build a transaction: over gRPC `ContractValidateError.Code` is set and the message
  starts with `Contract validate error :`. Over HTTP `Code` is zero and the message starts with a
  Java class name. Match on the type, not on `Code` or the text.
- A deployment **without** an ABI gets an empty ABI field over HTTP, so its txid and contract
  address differ from what gRPC would build. Always take the address from
  `DeployedContractAddress` of the transaction you actually sign.
- `EstimateEnergy` only works on nodes started with `vm.estimateEnergy` (public mainnet nodes are
  not). `EstimateTRC20Transfer` and `EstimateDeployContract` measure with a constant call and work
  everywhere.

## Pitfalls

- Never pass a bare number where an amount is expected, and never convert `TokenAmount` to `SUN`.
- Never hard-code fees: read `c.ChainParams(ctx)`. Testnets and private networks differ from mainnet.
- Never edit `RawData` after signing; edit, `UpdateHash()`, then sign.
- Do not add `EstimateActivationFee` on top of `EstimateTRXTransfer`: the transfer estimate already
  prices creating the recipient.
- A broadcast that returns no error is not a confirmation. Wait for the receipt and check its result.
- `GetBlockByLimitNext2` excludes `end`. Public nodes keep no deep history: an old block may come
  back empty rather than as an error.
