package tornadocash

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// PoolInfo describes a single Tornado Cash mixer pool.
type PoolInfo struct {
	Chain        string
	Token        string
	Denomination string
	Address      common.Address
	ValueWei     *big.Int
}

var (
	wei17 = new(big.Int).Exp(big.NewInt(10), big.NewInt(17), nil) // 0.1 token
	wei18 = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil) // 1 token
	wei19 = new(big.Int).Exp(big.NewInt(10), big.NewInt(19), nil) // 10 tokens
	wei20 = new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil) // 100 tokens
	wei21 = new(big.Int).Exp(big.NewInt(10), big.NewInt(21), nil) // 1000 tokens
	wei22 = new(big.Int).Exp(big.NewInt(10), big.NewInt(22), nil) // 10000 tokens
	wei23 = new(big.Int).Exp(big.NewInt(10), big.NewInt(23), nil) // 100000 tokens
)

func mulWei(n int64, unit *big.Int) *big.Int {
	return new(big.Int).Mul(big.NewInt(n), unit)
}

// allPools contains every known Tornado Cash native-token pool across
// supported EVM chains. Addresses sourced from on-chain deployments.
var allPools = []PoolInfo{
	// Ethereum
	{Chain: "Ethereum", Token: "ETH", Denomination: "0.1", Address: common.HexToAddress("0x12D66f87A04A9E220743712cE6d9bB1B5616B8Fc"), ValueWei: wei17},
	{Chain: "Ethereum", Token: "ETH", Denomination: "1", Address: common.HexToAddress("0x47CE0C6eD5B0Ce3d3A51fdb1C52DC66a7c3c2936"), ValueWei: wei18},
	{Chain: "Ethereum", Token: "ETH", Denomination: "10", Address: common.HexToAddress("0x910Cbd523D972eb0a6f4cAe4618aD62622b39DbF"), ValueWei: wei19},
	{Chain: "Ethereum", Token: "ETH", Denomination: "100", Address: common.HexToAddress("0xA160cdAB225685dA1d56aa342Ad8841c3b53f291"), ValueWei: wei20},

	// BSC
	{Chain: "BSC", Token: "BNB", Denomination: "0.1", Address: common.HexToAddress("0x84443CFd09A48AF6eF360C6976C5392aC5023a1f"), ValueWei: wei17},
	{Chain: "BSC", Token: "BNB", Denomination: "1", Address: common.HexToAddress("0xd47438C816c9E7f2E2888E060936a499Af9582b3"), ValueWei: wei18},
	{Chain: "BSC", Token: "BNB", Denomination: "10", Address: common.HexToAddress("0x330bdFADE01eE9bF63C209Ee33102DD334618e0a"), ValueWei: wei19},
	{Chain: "BSC", Token: "BNB", Denomination: "100", Address: common.HexToAddress("0x1E34A77868E19A6647b1f2F47B51ed72dEDE95DD"), ValueWei: wei20},

	// Polygon
	{Chain: "Polygon", Token: "MATIC", Denomination: "100", Address: common.HexToAddress("0x1E34A77868E19A6647b1f2F47B51ed72dEDE95DD"), ValueWei: wei20},
	{Chain: "Polygon", Token: "MATIC", Denomination: "1000", Address: common.HexToAddress("0xdf231d99Ff8b6c6CBF4E9B9a945CbAcEf9339178"), ValueWei: wei21},
	{Chain: "Polygon", Token: "MATIC", Denomination: "10000", Address: common.HexToAddress("0xaf4c0B70B2Ea9FB7487C7CbB37aDa259579FE040"), ValueWei: wei22},
	{Chain: "Polygon", Token: "MATIC", Denomination: "100000", Address: common.HexToAddress("0xa5C2254e4253490C54cef0a4347fddb8f75A4998"), ValueWei: wei23},

	// Avalanche
	{Chain: "Avalanche", Token: "AVAX", Denomination: "10", Address: common.HexToAddress("0x330bdFADE01eE9bF63C209Ee33102DD334618e0a"), ValueWei: wei19},
	{Chain: "Avalanche", Token: "AVAX", Denomination: "100", Address: common.HexToAddress("0x1E34A77868E19A6647b1f2F47B51ed72dEDE95DD"), ValueWei: wei20},
	{Chain: "Avalanche", Token: "AVAX", Denomination: "500", Address: common.HexToAddress("0xaf8d1839c3c67cf571aa74B5c12398d4901147B3"), ValueWei: mulWei(5, wei20)},

	// Arbitrum
	{Chain: "Arbitrum", Token: "ETH", Denomination: "0.1", Address: common.HexToAddress("0x84443CFd09A48AF6eF360C6976C5392aC5023a1f"), ValueWei: wei17},
	{Chain: "Arbitrum", Token: "ETH", Denomination: "1", Address: common.HexToAddress("0xd47438C816c9E7f2E2888E060936a499Af9582b3"), ValueWei: wei18},
	{Chain: "Arbitrum", Token: "ETH", Denomination: "10", Address: common.HexToAddress("0x330bdFADE01eE9bF63C209Ee33102DD334618e0a"), ValueWei: wei19},
	{Chain: "Arbitrum", Token: "ETH", Denomination: "100", Address: common.HexToAddress("0x1E34A77868E19A6647b1f2F47B51ed72dEDE95DD"), ValueWei: wei20},

	// Optimism
	{Chain: "Optimism", Token: "ETH", Denomination: "0.1", Address: common.HexToAddress("0x84443CFd09A48AF6eF360C6976C5392aC5023a1f"), ValueWei: wei17},
	{Chain: "Optimism", Token: "ETH", Denomination: "1", Address: common.HexToAddress("0xd47438C816c9E7f2E2888E060936a499Af9582b3"), ValueWei: wei18},
	{Chain: "Optimism", Token: "ETH", Denomination: "10", Address: common.HexToAddress("0x330bdFADE01eE9bF63C209Ee33102DD334618e0a"), ValueWei: wei19},
	{Chain: "Optimism", Token: "ETH", Denomination: "100", Address: common.HexToAddress("0x1E34A77868E19A6647b1f2F47B51ed72dEDE95DD"), ValueWei: wei20},
}

// GetPool returns the pool matching the given chain and denomination.
func GetPool(chain, denomination string) (*PoolInfo, error) {
	for i := range allPools {
		if allPools[i].Chain == chain && allPools[i].Denomination == denomination {
			return &allPools[i], nil
		}
	}
	return nil, fmt.Errorf("no pool for chain %q denomination %q", chain, denomination)
}

// GetPoolByAddress returns the pool at the given address on the given chain.
func GetPoolByAddress(chain string, addr common.Address) (*PoolInfo, error) {
	for i := range allPools {
		if allPools[i].Chain == chain && allPools[i].Address == addr {
			return &allPools[i], nil
		}
	}
	return nil, fmt.Errorf("no pool at %s on chain %q", addr.Hex(), chain)
}

// ListPools returns all pools for a given chain. If denomination is non-empty,
// filters to that denomination only.
func ListPools(chain, denomination string) []PoolInfo {
	var result []PoolInfo
	for _, p := range allPools {
		if p.Chain != chain {
			continue
		}
		if denomination != "" && p.Denomination != denomination {
			continue
		}
		result = append(result, p)
	}
	return result
}

// SupportedChains returns the list of chains that have Tornado Cash pools.
var SupportedChains = []string{"Ethereum", "BSC", "Polygon", "Avalanche", "Arbitrum", "Optimism"}
