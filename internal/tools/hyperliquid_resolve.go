package tools

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/vultisig/mcp/internal/hyperliquid"
)

func resolveAssetIndex(meta *hyperliquid.PerpMeta, coin string) (int, int, error) {
	for i, asset := range meta.Universe {
		if strings.EqualFold(asset.Name, coin) {
			return i, asset.SzDecimals, nil
		}
	}
	return 0, 0, fmt.Errorf("coin %q not found in Hyperliquid perpetuals universe", coin)
}

func resolveSpotAssetIndex(meta *hyperliquid.SpotMeta, coin string) (int, int, error) {
	if strings.HasPrefix(coin, "@") {
		idxStr := coin[1:]
		idx, err := strconv.Atoi(idxStr)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid spot index format %q: %w", coin, err)
		}
		for _, pair := range meta.Universe {
			if pair.Index == idx {
				szDecimals := 0
				if len(pair.Tokens) > 0 {
					for _, tok := range meta.Tokens {
						if tok.Index == pair.Tokens[0] {
							szDecimals = tok.SzDecimals
							break
						}
					}
				}
				return 10000 + pair.Index, szDecimals, nil
			}
		}
		return 0, 0, fmt.Errorf("spot index @%d not found in Hyperliquid spot universe", idx)
	}

	for _, pair := range meta.Universe {
		if strings.EqualFold(pair.Name, coin) {
			szDecimals := 0
			if len(pair.Tokens) > 0 {
				for _, tok := range meta.Tokens {
					if tok.Index == pair.Tokens[0] {
						szDecimals = tok.SzDecimals
						break
					}
				}
			}
			return 10000 + pair.Index, szDecimals, nil
		}
	}

	for _, pair := range meta.Universe {
		parts := strings.SplitN(pair.Name, "/", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], coin) {
			szDecimals := 0
			if len(pair.Tokens) > 0 {
				for _, tok := range meta.Tokens {
					if tok.Index == pair.Tokens[0] {
						szDecimals = tok.SzDecimals
						break
					}
				}
			}
			return 10000 + pair.Index, szDecimals, nil
		}
	}

	return 0, 0, fmt.Errorf("coin %q not found in Hyperliquid spot universe", coin)
}

func truncateToDecimals(value string, decimals int) (string, error) {
	r := new(big.Rat)
	_, ok := r.SetString(value)
	if !ok {
		return "", fmt.Errorf("invalid number: %q", value)
	}
	if r.Sign() <= 0 {
		return "", fmt.Errorf("value must be positive: %q", value)
	}

	dotIdx := strings.Index(value, ".")
	if dotIdx < 0 {
		return value, nil
	}

	fracPart := value[dotIdx+1:]
	if len(fracPart) <= decimals {
		return value, nil
	}

	if decimals == 0 {
		return value[:dotIdx], nil
	}

	return value[:dotIdx+1+decimals], nil
}

// normalizePrice ensures price conforms to Hyperliquid tick rules.
// Max 5 significant figures. Max (MAX_DECIMALS - szDecimals) decimal places.
// MAX_DECIMALS = 6 for perps, 8 for spot.
// Integer prices always pass through.
func normalizePrice(value string, szDecimals int, isSpot bool) (string, error) {
	r := new(big.Rat)
	_, ok := r.SetString(value)
	if !ok {
		return "", fmt.Errorf("invalid price: %q", value)
	}
	if r.Sign() <= 0 {
		return "", fmt.Errorf("price must be positive: %q", value)
	}

	maxDecimals := 6
	if isSpot {
		maxDecimals = 8
	}
	allowedDecimals := maxDecimals - szDecimals
	if allowedDecimals < 0 {
		allowedDecimals = 0
	}

	dotIdx := strings.Index(value, ".")
	if dotIdx < 0 {
		return value, nil
	}

	fracPart := value[dotIdx+1:]
	if len(fracPart) > allowedDecimals {
		if allowedDecimals == 0 {
			value = value[:dotIdx]
		} else {
			value = value[:dotIdx+1+allowedDecimals]
		}
	}

	value = strings.TrimRight(value, "0")
	value = strings.TrimRight(value, ".")

	sigFigs := countSigFigs(value)
	if sigFigs > 5 {
		return "", fmt.Errorf("price %q has %d significant figures (max 5)", value, sigFigs)
	}

	return value, nil
}

func countSigFigs(value string) int {
	s := strings.Replace(value, ".", "", 1)
	s = strings.TrimLeft(s, "0")
	s = strings.TrimRight(s, "0")
	if s == "" {
		return 1
	}

	dotIdx := strings.Index(value, ".")
	if dotIdx >= 0 {
		fracPart := value[dotIdx+1:]
		trimmedFrac := strings.TrimRight(fracPart, "0")
		intPart := value[:dotIdx]
		intTrimmed := strings.TrimLeft(intPart, "0")
		if intTrimmed == "" || intTrimmed == "0" {
			return len(strings.TrimLeft(fracPart[:len(trimmedFrac)], "0"))
		}
		return len(intTrimmed) + len(trimmedFrac)
	}

	return len(strings.TrimLeft(strings.TrimRight(s, "0"), "0"))
}

func resolveCoinForAPI(coin string, isSpot bool, spotMeta *hyperliquid.SpotMeta) string {
	if !isSpot {
		return coin
	}

	if strings.HasPrefix(coin, "@") {
		return coin
	}

	if strings.Contains(coin, "/") {
		return coin
	}

	if spotMeta == nil {
		return coin
	}

	for _, pair := range spotMeta.Universe {
		parts := strings.SplitN(pair.Name, "/", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], coin) {
			if pair.IsCanonical {
				return pair.Name
			}
			return fmt.Sprintf("@%d", pair.Index)
		}
	}

	return coin
}
