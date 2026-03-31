---
name: Buy Credits
description: Purchase agent credits by sending USDC to the treasury
tags: [credits, purchase, usdc, treasury]
---

# Buy Credits

Purchase Vultisig agent credits by sending USDC to the treasury wallet. Credits are used for AI conversations with premium models.

## Pricing

- 1 USDC = 1 credit dollar
- Credits are added automatically after the USDC transfer confirms on-chain (usually within 1-2 minutes)
- Supported chains: **Ethereum**, **Arbitrum**, **Base**
- Supported token: **USDC only** (native USDC, not bridged)

## Treasury Address

`0x5864a6dfD50C4D3668FEe65f23890E46cbc54ec6`

This is the same address on all supported chains.

## Pre-checks

STOP — before building a send, verify:

1. The user has specified how much USDC they want to buy (e.g. "$10 of credits", "buy 5 USDC of credits")
2. Check their USDC balance on the chain they want to send from using `evm_get_token_balance`
3. If they don't specify a chain, check balances on all three supported chains and suggest the one with the highest USDC balance
4. If insufficient USDC balance on all chains, tell the user and suggest they swap or bridge USDC first

USDC contract addresses:
- Ethereum: `0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48`
- Arbitrum: `0xaf88d065e77c8cC2239327C5EDb3A432268e5831`
- Base: `0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913`

## Building the Credit Purchase

When the user confirms the amount and chain, emit a send transaction to the treasury:

Call respond_to_user with BOTH text AND actions:

- text: "Send [amount] USDC to the Vultisig treasury on [chain] to purchase credits. Ready to send?"
- actions: [{type: "build_send_tx", title: "Buy Credits", params: {chain: "[chain]", symbol: "USDC", address: "0x5864a6dfD50C4D3668FEe65f23890E46cbc54ec6", amount: "[amount]"}}]

RULES:
- MUST call respond_to_user (not plain text)
- Text MUST end with "Ready to send?"
- Use the exact treasury address above — do NOT modify it

## Post-Confirmation

After the user confirms:
- User confirms → call respond_to_user with actions: [{type: "sign_tx", title: "Sign Credit Purchase"}]
- User cancels → acknowledge, no sign_tx

After the transaction is signed and broadcast:
- Tell the user: "Your USDC transfer has been submitted. Credits will be added to your account automatically within 1-2 minutes once the transaction confirms on [chain]."
- Suggest they use the `check_credits` tool to verify their updated balance after a couple of minutes

## DO NOTs

- DO NOT fabricate or modify the treasury address
- DO NOT suggest sending tokens other than USDC
- DO NOT suggest sending on chains other than Ethereum, Arbitrum, or Base
- DO NOT promise instant credit delivery — it takes 1-2 minutes for on-chain confirmation
- DO NOT assume an amount — always ask if not specified
- NEVER say "Credits added" unless you have verified via check_credits
