---
name: Buy Credits
description: Purchase agent credits by sending USDC to the treasury wallet
tags: [credits, purchase, usdc, treasury, evm, token-transfer]
---

# Buy Credits

Purchase Vultisig agent credits by sending USDC to the treasury wallet. Credits are used for AI conversations with premium models. 1 USDC = $1.00 in credits.

## Addresses

### Treasury Wallet

The treasury address is the same on all supported chains:

`0x5864a6dfD50C4D3668FEe65f23890E46cbc54ec6`

### USDC Contracts

| Chain | USDC Address | Decimals |
|-------|-------------|----------|
| Ethereum | `0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48` | 6 |
| Arbitrum | `0xaf88d065e77c8cC2239327C5EDb3A432268e5831` | 6 |
| Base | `0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913` | 6 |

All USDC contracts use 6 decimals. 1 USDC = 1,000,000 base units.

## Pre-checks

STOP — before building a transaction, you MUST:

1. The user MUST specify how much USDC they want to spend (e.g., "$10 of credits", "buy 5 USDC of credits"). If they haven't, ask: "How much USDC would you like to spend on credits?"
2. Check their USDC balance on the chain they want to send from (see Read Operations below)
3. If they don't specify a chain, check balances on all three chains and suggest the one with the highest USDC balance
4. If insufficient USDC on all chains, tell the user and DO NOT build any transaction

## Read Operations

All read operations use `evm_call` or `evm_get_token_balance`. No transaction needed.

### Check USDC balance on a specific chain

```
evm_get_token_balance(
  chain: "<chain>",
  contract_address: "<usdc_address>",
  address: "<user_address>"
)
```

Example — check USDC on Base:

```
evm_get_token_balance(
  chain: "Base",
  contract_address: "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
  address: "<user_address>"
)
```

### Check USDC balance via evm_call (alternative)

```
evm_call(
  chain: "<chain>",
  to: "<usdc_address>",
  data: <abi_encode("balanceOf(address)", "<user_address>")>,
  output_types: "uint256"
)
```

The result is in base units (6 decimals). Divide by 1,000,000 to get USDC amount.

### Check balances on all chains

To find the best chain, check all three:

```
evm_get_token_balance(chain: "Ethereum", contract_address: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", address: "<user>")
evm_get_token_balance(chain: "Arbitrum", contract_address: "0xaf88d065e77c8cC2239327C5EDb3A432268e5831", address: "<user>")
evm_get_token_balance(chain: "Base", contract_address: "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913", address: "<user>")
```

Recommend the chain with the highest balance. If balances are similar, prefer Base or Arbitrum (lower gas fees).

## Purchase Flow

Purchasing credits is a simple USDC transfer to the treasury. No approval step needed (this is a direct ERC-20 transfer, not a contract interaction).

### Step 1 — Check USDC balance

Use the read operations above to verify the user has sufficient USDC.

### Step 2 — Build the USDC transfer

Encode the ERC-20 `transfer` calldata:

```
abi_encode(
  signature: "transfer(address,uint256)",
  args: "0x5864a6dfD50C4D3668FEe65f23890E46cbc54ec6,<amount_in_base_units>"
)
```

The amount must be in base units (6 decimals). For example:
- 1 USDC = `1000000`
- 5 USDC = `5000000`
- 10 USDC = `10000000`
- 100 USDC = `100000000`

Get transaction parameters:

```
evm_tx_info(
  address: "<sender>",
  to: "<usdc_address>",
  data: "<transfer_calldata>",
  value: "0"
)
```

Build the transaction:

```
build_evm_tx(
  to: "<usdc_address>",
  value: "0",
  data: "<transfer_calldata>",
  nonce: "<nonce>",
  gas_limit: "<estimated_gas>",
  max_fee_per_gas: "<suggested_max_fee>",
  max_priority_fee_per_gas: "<tip>"
)
```

### Step 3 — Confirm with user (respond_to_user required)

Call respond_to_user with BOTH text AND actions:

- text: "Send [amount] USDC to the Vultisig treasury on [chain] to purchase $[amount] in credits. Ready to send?"
- actions: the build_evm_tx action from Step 2

RULES:
- MUST call respond_to_user (not plain text — plain text breaks the app)
- Text MUST end with "Ready to send?"
- Zero extra words before or after the template

### Step 4 — Post-confirmation

After the user confirms:
- User confirms (yes, confirm, go) → the transaction is signed and broadcast
- User cancels → acknowledge, no further action

After the transaction is broadcast, tell the user:

"Your USDC transfer has been submitted on [chain]. Credits will be added to your account automatically within 1-2 minutes once the transaction confirms. You can check your balance with 'check my credits'."

## Alternative: Simple send flow

If the evm_tx building flow is complex, you can also use the simpler send-transfer approach:

Call respond_to_user with:
- text: "Send [amount] USDC to the Vultisig treasury on [chain] to purchase credits. Ready to send?"
- actions: [{type: "build_send_tx", title: "Buy Credits", params: {chain: "[chain]", symbol: "USDC", address: "0x5864a6dfD50C4D3668FEe65f23890E46cbc54ec6", amount: "[amount]"}}]

This uses the standard send-transfer flow which handles encoding automatically.

## Credit Delivery

- Credits are added **automatically** by the on-chain deposit scanner
- Typical confirmation time: 1-2 minutes after the transaction confirms
- The user can verify their updated balance using the `check_credits` tool
- 1 USDC = $1.00 in credits (no conversion fee)

## DO NOTs

- **DO NOT** fabricate or modify the treasury address. It MUST be exactly `0x5864a6dfD50C4D3668FEe65f23890E46cbc54ec6`
- **DO NOT** suggest sending tokens other than USDC. Only native USDC is accepted.
- **DO NOT** suggest sending on chains other than Ethereum, Arbitrum, or Base
- **DO NOT** set `value` to anything other than `"0"`. This is an ERC-20 transfer, not a native ETH send.
- **DO NOT** promise instant credit delivery. It takes 1-2 minutes for on-chain confirmation + scanner processing.
- **DO NOT** assume an amount. Always ask the user how much they want to buy.
- **DO NOT** say "Credits added" unless verified via `check_credits`
- **DO NOT** use `search_token` to look up USDC. Use the addresses in this skill directly.
- **DO NOT** confuse the USDC contract address (the `to` field) with the treasury address (the `transfer` recipient argument)
