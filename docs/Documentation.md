# Wallet Monitor — Documentation

## Overview

Wallet Monitor is a Go CLI tool that connects to a Solana Yellowstone gRPC (Geyser) endpoint and streams real-time updates for one or more wallet addresses. It subscribes to both **account data changes** and **transactions** involving the specified wallets, printing structured output to the terminal.

## Architecture

```
main.go                            Entry point, CLI orchestration
internal/
  config/config.go                 Flag parsing and validation
  grpc/client.go                   gRPC connection with TLS/keepalive/basic auth
  decoder/
    accounts.go                    Account role resolution from MessageHeader
    programs.go                    Well-known Solana program name registry
  subscriber/subscriber.go         Subscription lifecycle and output formatting
```

### Package Responsibilities

- **config** — Parses CLI flags (`--endpoint`, `--token`, `--username`, `--password`, `--wallet`, `--insecure`) and validates required fields.
- **grpc** — Creates a gRPC client connection with automatic TLS detection, system certificate pool, keepalive, and optional basic auth (per-RPC credentials).
- **decoder/accounts** — Resolves account roles (signer, writable, readonly, fee payer) from the Solana `MessageHeader` fields, including loaded addresses from versioned transactions.
- **decoder/programs** — Maps ~30 well-known Solana program addresses to human-readable names (Jupiter, Raydium, Orca, Meteora, Pump.fun, Token Program, System, etc.).
- **subscriber** — Builds the `SubscribeRequest`, sends it over a bidirectional stream, dispatches incoming updates, and renders detailed transaction breakdowns (instruction tree with CPI calls, balance changes, token movements, net wallet summary).

## Subscription Details

The tool subscribes with `CONFIRMED` commitment level to:

1. **Account filter** (`wallet_accounts`) — Watches for any data change on the wallet account(s) themselves (balance changes, data mutations).
2. **Transaction filter** (`wallet_txns`) — Watches for all non-vote transactions that include any of the monitored wallets in their account list.

## Output Format

### Account Updates

Printed when the wallet's on-chain account data changes:

- Public key, owner program, SOL balance, slot number
- Data size, write version, triggering transaction signature
- Matched subscription filters

### Transaction Updates

Printed when a transaction involving the wallet is confirmed:

- **Header** — Signature, slot, fee, status (SUCCESS/FAILED), compute units
- **Account table** — Every account in the transaction with index, address, and role flags (SIGNER, WRITABLE, READONLY, fee payer)
- **Instruction tree** — Each top-level instruction showing:
  - The program invoked (with human-readable name for ~30 well-known programs: Jupiter, Raydium, Orca, Meteora, Pump.fun, Token Program, System, etc.)
  - Accounts used by that instruction in order, with their roles
  - Inner CPI calls with program name, accounts, and stack depth
- **SOL balance changes** — Per-account delta with before/after lamport values
- **Token balance changes** — Grouped by mint, showing per-owner before/after amounts with delta
- **Net changes summary** — Wallet-centric view of net SOL and token movements (SENT/RECEIVED)
- **Program logs** — Full log output from all program invocations

## Graceful Shutdown

The process listens for `SIGINT` and `SIGTERM`. On signal, the context is cancelled, the gRPC stream closes, and the connection is cleaned up.

## Configuration

All configuration is provided via CLI flags. No environment variables or config files are required.

| Flag         | Type   | Default | Description                                      |
|--------------|--------|---------|--------------------------------------------------|
| `--endpoint` | string | —       | Yellowstone gRPC URL (required)                  |
| `--wallet`   | string | —       | Wallet address (required, repeatable)            |
| `--token`    | string | ""      | x-token authentication header                    |
| `--username` | string | ""      | Basic auth username                              |
| `--password` | string | ""      | Basic auth password                              |
| `--output`   | string | ""      | Write updates to file instead of terminal       |
| `--insecure` | bool   | false   | Skip TLS (auto-detected for `http://` endpoints) |

## Dependencies

- `google.golang.org/grpc` — gRPC client
- `github.com/mr-tron/base58` — Base58 encoding for Solana addresses/signatures
- `github.com/rpcpool/yellowstone-grpc/examples/golang/proto` — Geyser protobuf definitions (local reference via `replace` directive)
