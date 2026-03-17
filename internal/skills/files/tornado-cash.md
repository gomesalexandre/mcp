---
name: Tornado Cash Privacy Mixer
description: Deposit and withdraw ETH through Tornado Cash privacy pools on EVM chains
tags: [tornado-cash, privacy, mixer, evm]
---

# Tornado Cash Privacy Mixer

Deposit and withdraw ETH through Tornado Cash privacy pools. Notes must be stored securely — losing a note means permanently locked funds.

## Supported Chains & Pools

| Chain | Denominations |
|-------|--------------|
| Ethereum | 0.1, 1, 10, 100 ETH |
| BSC | 0.1, 1, 10, 100 BNB |
| Polygon | 100, 1000, 10000, 100000 MATIC |
| Avalanche | 10, 100, 500 AVAX |
| Arbitrum | 0.1, 1, 10, 100 ETH |
| Optimism | 0.1, 1, 10, 100 ETH |

Use `tornado_get_pools` to get pool addresses and details.

## Deposit Workflow

### 1. Generate a note

```
tornado_generate_note()
```

Returns `commitment`, `nullifier_hash`, and `note_data` containing `secret`, `nullifier`, and `commitment`.

### 2. Store the note in agent-backend

**CRITICAL**: Store the note BEFORE depositing. If the deposit succeeds but the note is lost, funds are permanently locked.

Store via `POST /agent/protocol-states`:
```json
{
  "protocol": "tornado-cash",
  "label": "<chain>-<denomination>-<short_commitment>",
  "state_data": {
    "secret": "<hex>",
    "nullifier": "<hex>",
    "commitment": "<hex>",
    "chain": "<chain>",
    "denomination": "<denomination>",
    "pool_address": "<pool_address>"
  },
  "data_class": "secret"
}
```

### 3. Check balance

Ensure the user has enough native tokens for the deposit denomination + gas.

### 4. Prepare deposit parameters

```
tornado_deposit(
  chain: "Ethereum",
  commitment: "<commitment_from_step_1>",
  denomination: "1"
)
```

Returns `pool_address`, `function_signature`, `calldata`, and `value_wei`.

### 5. Build the transaction

Use the standard EVM tx workflow:
```
evm_tx_info(address: "<sender>", to: "<pool_address>", data: "<calldata>", value: "<value_wei>")
build_evm_tx(to: "<pool_address>", value: "<value_wei>", data: "<calldata>", ...)
```

### 6. Verify deposit

After the transaction is confirmed:
```
tornado_check_deposit(chain: "Ethereum", pool_address: "<pool_address>", commitment: "<commitment>")
```

## Withdrawal Workflow

### 1. Retrieve the note

Fetch from agent-backend: `GET /agent/protocol-states?protocol=tornado-cash&status=active`

### 2. Compute nullifier hash

```
tornado_compute_nullifier_hash(nullifier: "<nullifier_from_note>")
```

### 3. Check not already spent

```
tornado_check_spent(chain: "Ethereum", pool_address: "<pool_address>", nullifier_hash: "<nullifier_hash>")
```

If `spent: true`, the note has already been withdrawn.

### 4. Get Merkle proof

```
tornado_get_merkle_path(
  chain: "Ethereum",
  pool_address: "<pool_address>",
  commitment: "<commitment>",
  from_block: "<pool_deploy_block>"
)
```

WARNING: This is a heavy operation — may take minutes on pools with many deposits.

Returns `root`, `path_elements`, `path_indices`, and `leaf_index`.

### 5. Generate zk-SNARK proof

The proof must be generated externally (client-side or via a proving service). The MCP server provides all inputs needed for proof generation but does not generate proofs itself.

### 6. Prepare withdrawal parameters

```
tornado_withdraw(
  chain: "Ethereum",
  pool_address: "<pool_address>",
  proof: "<snark_proof_hex>",
  root: "<merkle_root>",
  nullifier_hash: "<nullifier_hash>",
  recipient: "<recipient_address>",
  relayer: "0x0000000000000000000000000000000000000000",
  fee: "0",
  refund: "0"
)
```

Returns `pool_address`, `function_signature`, and `calldata`.

### 7. Build the transaction

```
evm_tx_info(address: "<sender>", to: "<pool_address>", data: "<calldata>", value: "0")
build_evm_tx(to: "<pool_address>", value: "0", data: "<calldata>", ...)
```

### 8. Mark note as used

After successful withdrawal, update the note status:
`PATCH /agent/protocol-states/:id/status` with `{"status": "used"}`

## DO NOTs

- **DO NOT** deposit without storing the note in agent-backend first — lost notes = permanently locked funds
- **DO NOT** reveal note secrets (secret, nullifier) in chat messages or logs
- **DO NOT** reuse notes — each note is single-use
- **DO NOT** delete active notes — mark them as "used" after withdrawal
- **DO NOT** withdraw to the same address that deposited — this defeats the privacy purpose
- **DO NOT** generate proofs — the MCP server provides parameters only; proof generation is client-side
