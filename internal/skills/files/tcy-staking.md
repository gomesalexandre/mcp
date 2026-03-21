---
name: TCY Staking
description: Stake and unstake TCY (THORChain Governance Token) to earn 10% of THORChain system income
tags: [tcy, staking, thorchain, defi, cosmos]
---

# TCY Staking

Stake and unstake TCY on THORChain to earn a proportional share of 10% of THORChain system income (swap fees + block emissions), distributed per block.

## Overview

TCY is a native THORChain token (fixed supply: 210M) that converts defaulted THORFi debt into equity. Staked TCY earns rewards when the TCY fund reaches ≥2,100 RUNE.

Docs: [THORChain TCY](https://dev.thorchain.org/concepts/tcy.html) | [Memos](https://dev.thorchain.org/concepts/memos.html#stake-tcy)

## Read Operations

### Check TCY Staking Position

Query the TCY staker endpoint to see current staking position:

```
thorchain_query(query_type: "tcy_staker", thor_address: "<thor_address>")
```

This returns the staked TCY amount and pending rewards.

### Check TCY Balance

Check TCY balance for a THORChain address:

```
cosmos_get_balance(chain: "THORChain", address: "<thor_address>", denom: "tcgy")
```

Or use the user's THORChain address from Addresses context.

## Token Details

| Property | Value |
|----------|-------|
| Symbol | TCY |
| Denom | tcgy (1e8 = 1 TCY) |
| Min Stake | 0.001 TCY (100,000 in 1e8) |
| Supply | 210,000,000 TCY (fixed) |

## Stake TCY Flow

Stake TCY to start earning rewards. Uses `build_custom_tx` with deposit.

### Step 1 — Check TCY Balance

Verify the user has enough TCY to stake:

```
cosmos_get_balance(chain: "THORChain", address: "<sender_address>", denom: "tcgy")
```

### Step 2 — Build Stake Transaction

Use `build_custom_tx` with:
- **chain**: "THORChain"
- **symbol**: "TCY"
- **amount**: human-readable amount (e.g., "10" for 10 TCY)
- **tx_type**: "deposit"
- **memo**: "TCY+"

Example: Stake 10 TCY
```
build_custom_tx(
  chain: "THORChain",
  symbol: "TCY",
  amount: "10",
  tx_type: "deposit",
  memo: "TCY+"
)
```

### Step 3 — Confirmation

Always confirm before executing. Template:
"[STAKE/UNSTAKE] [amount] TCY. Ready to execute?"

When user confirms → return `sign_tx` action with empty params.

## Unstake TCY Flow

Unstake TCY to withdraw staked tokens and accumulated rewards. Uses the same `build_custom_tx` with a different memo.

### Unstake Format

- **Memo**: `TCY-:<basis_points>`
- **basis_points**: 0–10000 (where 10000 = 100%)
- **amount**: Set to "0" for unstakes (amount goes in memo)

### Examples

| Basis Points | Percentage | Memo |
|--------------|------------|------|
| 1000 | 10% | TCY-:1000 |
| 5000 | 50% | TCY-:5000 |
| 10000 | 100% | TCY-:10000 |

### Build Unstake Transaction

Use `build_custom_tx` with:
- **chain**: "THORChain"
- **symbol**: "TCY"
- **amount**: "0"
- **tx_type**: "deposit"
- **memo**: "TCY-:<basis_points>"

Example: Unstake 50% of staked TCY
```
build_custom_tx(
  chain: "THORChain",
  symbol: "TCY",
  amount: "0",
  tx_type: "deposit",
  memo: "TCY-:5000"
)
```

## Rewards

When TCY is staked:
- Earns 10% of THORChain system income (swap fees + block emissions)
- Rewards are distributed proportionally per block
- Rewards accumulate when TCY fund ≥2,100 RUNE
- Rewards are automatically included when unstaking

## TCY Claim (Different from Stake/Unstake)

The `TCY:<address>` memo is for **claiming** TCY from the reserve, NOT for staking or unstaking. Do not confuse this with the `TCY+` (stake) and `TCY-:` (unstake) memos.

## DO NOTs

- **DO NOT** stake TCY without confirming the exact amount with the user
- **DO NOT** unstake more than the user's staked amount — query position first
- **DO NOT** confuse `TCY:<address>` (claim) with `TCY+` (stake) or `TCY-:` (unstake)
- **DO NOT** stake below the minimum of 0.001 TCY (100,000 in 1e8)
- **DO NOT** use EVM transaction tools for TCY — TCY is a Cosmos SDK coin on THORChain, use `build_custom_tx` with `tx_type: "deposit"`
