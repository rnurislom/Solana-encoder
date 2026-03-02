# Wallet Monitor

A real-time Solana wallet monitoring tool built on [Yellowstone gRPC](https://github.com/rpcpool/yellowstone-grpc) (Geyser plugin). Subscribes to account updates and transactions for specified wallet addresses and prints them to the terminal.

## Prerequisites

- Go 1.24+
- A Yellowstone gRPC endpoint (e.g. from Chainstack, Triton, Helius, or self-hosted)

## Installation

```bash
go build -o wallet-monitor .
```

## Usage

```bash
# Authenticate with x-token
./wallet-monitor --endpoint https://your-grpc-endpoint.com --token YOUR_TOKEN --wallet WALLET_ADDRESS

# Authenticate with username & password
./wallet-monitor --endpoint https://your-grpc-endpoint.com \
  --username USER --password PASS --wallet WALLET_ADDRESS

# Both x-token and basic auth (if provider requires both)
./wallet-monitor --endpoint https://your-grpc-endpoint.com \
  --token YOUR_TOKEN --username USER --password PASS --wallet WALLET_ADDRESS

go build -o wallet-monitor . ./wallet-monitor --endpoint yellowstone-solana-mainnet.core.chainstack.com --token c061b329a471c3c6e40421c6788578a5 --username sharp-poincare --password PAblank-fame-slug-case-glade-eitherSS --wallet WALLET_ADDRESS

# Monitor multiple wallets
./wallet-monitor --endpoint https://your-grpc-endpoint.com --token YOUR_TOKEN \
  --wallet WALLET_ADDRESS_1 \
  --wallet WALLET_ADDRESS_2

# Save updates to file instead of terminal
./wallet-monitor --endpoint https://your-grpc-endpoint.com --token YOUR_TOKEN --wallet WALLET_ADDRESS --output updates.log

# Insecure (non-TLS) connection
./wallet-monitor --endpoint http://localhost:10000 --insecure --wallet WALLET_ADDRESS
```

## Flags

| Flag         | Required | Description                                   |
|--------------|----------|-----------------------------------------------|
| `--endpoint` | Yes      | Yellowstone gRPC endpoint URL                 |
| `--wallet`   | Yes      | Wallet address to monitor (repeatable)        |
| `--token`    | No       | Authentication token (x-token header)         |
| `--username` | No       | Basic auth username                           |
| `--password` | No       | Basic auth password                           |
| `--output`   | No       | Write updates to file instead of terminal    |
| `--insecure` | No       | Use insecure (non-TLS) connection             |

## Links

- [Detailed Documentation](Documentation.md)
