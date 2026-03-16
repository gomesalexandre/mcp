---
name: Hyperliquid Trading
description: Trade perpetuals and spot on Hyperliquid DEX
tags: [hyperliquid, perpetuals, spot, trading, defi]
---

# Hyperliquid Trading

Trade perpetuals and spot assets on Hyperliquid, a high-performance decentralized exchange on its own L1 chain.

## Key Facts

- Hyperliquid uses EVM (Ethereum) addresses — same as the user's Ethereum address
- All prices are USD-quoted for perpetuals
- Actions are signed via EIP-712 typed data — build tools return unsigned payloads for the client to sign
- REST API: `POST /info` for reads (no auth), `POST /exchange` for signed actions
- Spot assets use `@{index}` format internally (e.g. `@1` for the first spot pair)
- Perpetual assets use simple coin names (BTC, ETH)

## Spot Symbol Resolution

Spot markets have canonical pair names (e.g. PURR/USDC) and internal `@{index}` addressing. The tools handle resolution automatically — pass the coin name (e.g. PURR, BTC) and set `is_spot: true` for spot markets. Some UI names differ from API names (e.g. BTC spot may appear as UBTC/USDC). When in doubt, check `hyperliquid_get_all_mids` for available symbols.

## Checking Account State

1. **Spot balances**: `hyperliquid_get_spot_balances` — shows all spot token holdings
2. **Perp state**: `hyperliquid_get_perp_state` — shows open positions (non-zero only), margin summary, account value, withdrawable balance
3. **Open orders**: `hyperliquid_get_open_orders` — all active resting orders with coin, side, size, price, type, oid, cloid
4. **Order status**: `hyperliquid_get_order_status` — check a specific order by oid or cloid. Returns status (filled, resting, cancelled, triggered, rejected, marginCanceled) and timestamp
5. **Historical orders**: `hyperliquid_get_historical_orders` — up to 2000 recent historical orders with status

## Checking Market Data

1. **Mid prices**: `hyperliquid_get_all_mids` — map of all coin mid prices (no params)
2. **Order book**: `hyperliquid_get_order_book` — L2 book with bids/asks. Params: `coin` (required), `n_sig_figs` (optional, 2-5, default 5)
3. **Candles**: `hyperliquid_get_candles` — OHLCV data. Intervals: 1m, 3m, 5m, 15m, 30m, 1h, 2h, 4h, 8h, 12h, 1d, 3d, 1w, 1M. Max 5000 candles per call.

## Trade History & Rate Limits

1. **User fills**: `hyperliquid_get_user_fills` — trade fills with coin, side, price, size, fee, closedPnl. Max 2000 per call. Supports `start_time`/`end_time` for time-range filtering — always prefer narrow ranges.
2. **Historical orders**: `hyperliquid_get_historical_orders` — order lifecycle history. Max 2000 per call.
3. **Rate limit status**: `hyperliquid_get_user_rate_limit` — check current API usage (cumVlm, nRequestsUsed, nRequestsCap)

## Rate-Limit Awareness

The following endpoints have **extra API rate-limit weight proportional to the number of returned items**: `hyperliquid_get_historical_orders`, `hyperliquid_get_user_fills`, and `hyperliquid_get_candles`.

In agentic loops:
- Always use narrow time ranges for fills and candles
- Avoid polling these endpoints frequently
- Check `hyperliquid_get_user_rate_limit` before making heavy queries
- Prefer `hyperliquid_get_open_orders` and `hyperliquid_get_order_status` for order monitoring (lighter weight)

## Placing an Order

1. **Check state**: `hyperliquid_get_perp_state` or `hyperliquid_get_spot_balances` — verify margin/balance
2. **Check price**: `hyperliquid_get_all_mids` or `hyperliquid_get_order_book` — get current market price
3. **Build order**: `hyperliquid_build_order` with coin, side, size, price
   - `order_type`: "limit" (default) or "market" (encoded as aggressive IOC at worst-case price)
   - `tif`: Gtc (default), Ioc, Alo
   - `is_spot`: true for spot markets
   - `cloid`: optional client order ID for tracking
   - `expires_after`: unix ms timestamp for payload expiry safety
4. **Confirm** with user — show coin, side, size, price, order type
5. Client signs the EIP-712 typed data and submits to the exchange URL

Size is automatically truncated to `szDecimals` and price is normalized to comply with Hyperliquid precision rules (max 5 significant figures).

## Modifying an Order

Use `hyperliquid_build_modify_order` to change price and/or size of a resting order:
- Identify by `target_oid` (order ID) or `target_cloid` (client order ID) — exactly one required
- Provide new `coin`, `side`, `size`, `price` (all required)
- Optionally set a new `cloid` for the replacement order

## Cancelling an Order

Use `hyperliquid_build_cancel`:
- Identify by `oid` or `cloid` — exactly one required
- Must specify `coin` and `is_spot` (if spot market)

## Dead-Man Switch

Use `hyperliquid_build_schedule_cancel` for agentic safety:
- Set `time` to a future unix timestamp (>= 5 seconds ahead) — when triggered, cancels ALL open orders
- Omit `time` to clear a previously scheduled cancel
- Max 10 triggers per day (resets 00:00 UTC)
- Useful as a safety net in automated trading loops

## Withdrawing to Arbitrum

1. **Check balance**: `hyperliquid_get_perp_state` AND `hyperliquid_get_spot_balances` — verify the user has enough USDC (withdrawable amount from perp state). If USDC is in spot, they may need to class-transfer to perp first.
2. **Build withdraw**: `hyperliquid_build_withdraw` with `amount` and optionally `destination`
   - If `destination` is omitted, defaults to user's own address
3. Confirm with user — show amount, destination, and note ~$1 fee
4. Client signs and submits; withdrawal takes ~5 minutes to finalize on Arbitrum

## Sending USDC on Hyperliquid

1. **Check balance**: `hyperliquid_get_perp_state` AND `hyperliquid_get_spot_balances` — verify sufficient USDC for the transfer
2. **Build transfer**: `hyperliquid_build_usd_transfer` with `destination` and `amount`
3. Confirm with user — this is an internal L1 transfer, NOT a withdrawal to Arbitrum

## Sending Tokens

1. **Check balance**: `hyperliquid_get_spot_balances` — verify token holdings for the asset being sent
2. **Build transfer**: `hyperliquid_build_send_asset` with `destination`, `token` (format "tokenName:tokenId"), and `amount`
   - `source_dex`: "" for USDC perp, "spot" for spot
   - `destination_dex`: target DEX name
   - `from_sub_account`: subaccount address if applicable
   - Only collateral tokens can transfer to/from perp DEXes
3. Confirm with user

## Spot / Perp Margin Transfer

1. **Check balance**: `hyperliquid_get_spot_balances` (for spot→perp) or `hyperliquid_get_perp_state` (for perp→spot) — verify sufficient USDC in the source
2. **Build transfer**: `hyperliquid_build_class_transfer` with `amount` and `to_perp`
   - `to_perp: true` = spot → perp
   - `to_perp: false` = perp → spot
3. Confirm with user — show direction and amount

## Staking HYPE

1. **Check balance**: `hyperliquid_get_spot_balances` — verify HYPE holdings before staking
2. **Stake**: `hyperliquid_build_stake` with `wei` (1 HYPE = 1e18 wei)
3. **Unstake**: `hyperliquid_build_unstake` with `wei` — unstaked tokens undergo a **7-day queue** before reaching the spot account
4. Confirm with user — for unstaking, warn about the 7-day delay

## Vault/Subaccount Trading

All build tools accept an optional `vault_address` parameter for delegated trading through a Hyperliquid vault or subaccount. The signing address must have delegated trading permissions on the vault.

## Order Types

| Type | TIF | Behavior |
|------|-----|----------|
| Limit | Gtc | Rests on book until filled or cancelled |
| Limit | Ioc | Fills immediately or cancels remaining |
| Limit | Alo | Add liquidity only — cancels if would cross |
| Market | (Ioc) | Aggressive IOC at user-supplied worst-case price |

"Market" orders are encoded as IOC limits — you must provide a worst-case price (e.g. 5% above mid for buys).

## Pre-checks

STOP — before any trade:
- The user MUST have specified an exact size and price. If they said "buy some BTC" without specifics, ask for the amount and price, then WAIT.
- ALWAYS check margin/balance before building an order.
- ALWAYS check current mid price so the user understands the market context.

## DO NOTs

- **DO NOT** trade without checking account state and current price first
- **DO NOT** assume size or price — always ask the user for exact values
- **DO NOT** confuse spot and perp symbols — set `is_spot: true` for spot markets
- **DO NOT** set leverage via order tools — leverage is managed separately on Hyperliquid
- **DO NOT** ignore precision rules — size and price are auto-validated, but ensure inputs are reasonable
- **DO NOT** poll `hyperliquid_get_historical_orders` or `hyperliquid_get_user_fills` in tight loops — use `hyperliquid_get_order_status` instead
- **DO NOT** build orders without confirming the full details with the user first
