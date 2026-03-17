package tools

import (
	"testing"

	"github.com/vultisig/mcp/internal/hyperliquid"
)

func TestCountSigFigs(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"100", 1},
		{"123", 3},
		{"1000", 1},
		{"1001", 4},
		{"0.001234", 4},
		{"0.00100", 1},
		{"12345", 5},
		{"10", 1},
		{"1.5", 2},
		{"1.50", 2},
		{"0.0050", 1},
		{"2080", 3},
		{"2080.5", 5},
		{"0", 1},
		{"0.0", 1},
	}

	for _, tt := range tests {
		got := countSigFigs(tt.input)
		if got != tt.want {
			t.Errorf("countSigFigs(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestNormalizePrice(t *testing.T) {
	tests := []struct {
		value      string
		szDecimals int
		isSpot     bool
		want       string
		wantErr    bool
	}{
		{"2080", 2, false, "2080", false},
		{"2080.5", 2, false, "2080.5", false},
		{"100", 0, false, "100", false},
		{"0.001234", 2, false, "0.0012", false},
		{"123456", 0, false, "123456", false},
		{"0", 0, false, "", true},
		{"-1", 0, false, "", true},
		{"abc", 0, false, "", true},
		{"1.2345", 0, true, "1.2345", false},
		{"1.2345", 0, false, "1.2345", false},
		{"1.23456789", 0, true, "", true},
		{"1.23456789", 0, false, "", true},
	}

	for _, tt := range tests {
		got, err := normalizePrice(tt.value, tt.szDecimals, tt.isSpot)
		if tt.wantErr {
			if err == nil {
				t.Errorf("normalizePrice(%q, %d, %v) expected error, got %q", tt.value, tt.szDecimals, tt.isSpot, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("normalizePrice(%q, %d, %v) unexpected error: %v", tt.value, tt.szDecimals, tt.isSpot, err)
			continue
		}
		if got != tt.want {
			t.Errorf("normalizePrice(%q, %d, %v) = %q, want %q", tt.value, tt.szDecimals, tt.isSpot, got, tt.want)
		}
	}
}

func TestTruncateToDecimals(t *testing.T) {
	tests := []struct {
		value    string
		decimals int
		want     string
		wantErr  bool
	}{
		{"1.23456", 2, "1.23", false},
		{"1.23456", 5, "1.23456", false},
		{"1.23456", 8, "1.23456", false},
		{"100", 2, "100", false},
		{"1.23456", 0, "1", false},
		{"0.001", 2, "0.00", false},
		{"0", 2, "", true},
		{"-1.5", 2, "", true},
		{"abc", 2, "", true},
	}

	for _, tt := range tests {
		got, err := truncateToDecimals(tt.value, tt.decimals)
		if tt.wantErr {
			if err == nil {
				t.Errorf("truncateToDecimals(%q, %d) expected error, got %q", tt.value, tt.decimals, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("truncateToDecimals(%q, %d) unexpected error: %v", tt.value, tt.decimals, err)
			continue
		}
		if got != tt.want {
			t.Errorf("truncateToDecimals(%q, %d) = %q, want %q", tt.value, tt.decimals, got, tt.want)
		}
	}
}

func TestResolveAssetIndex(t *testing.T) {
	meta := &hyperliquid.PerpMeta{
		Universe: []hyperliquid.PerpAsset{
			{Name: "BTC", SzDecimals: 5},
			{Name: "ETH", SzDecimals: 4},
			{Name: "SOL", SzDecimals: 2},
		},
	}

	tests := []struct {
		coin      string
		wantIdx   int
		wantSzDec int
		wantErr   bool
	}{
		{"BTC", 0, 5, false},
		{"ETH", 1, 4, false},
		{"eth", 1, 4, false},
		{"SOL", 2, 2, false},
		{"DOGE", 0, 0, true},
	}

	for _, tt := range tests {
		idx, szDec, err := resolveAssetIndex(meta, tt.coin)
		if tt.wantErr {
			if err == nil {
				t.Errorf("resolveAssetIndex(%q) expected error", tt.coin)
			}
			continue
		}
		if err != nil {
			t.Errorf("resolveAssetIndex(%q) unexpected error: %v", tt.coin, err)
			continue
		}
		if idx != tt.wantIdx || szDec != tt.wantSzDec {
			t.Errorf("resolveAssetIndex(%q) = (%d, %d), want (%d, %d)", tt.coin, idx, szDec, tt.wantIdx, tt.wantSzDec)
		}
	}
}

func TestResolveSpotAssetIndex(t *testing.T) {
	meta := &hyperliquid.SpotMeta{
		Universe: []hyperliquid.SpotPair{
			{Name: "PURR/USDC", Index: 1, Tokens: []int{100}, IsCanonical: true},
			{Name: "HYPE/USDC", Index: 2, Tokens: []int{101}, IsCanonical: true},
		},
		Tokens: []hyperliquid.SpotToken{
			{Index: 100, SzDecimals: 2},
			{Index: 101, SzDecimals: 4},
		},
	}

	tests := []struct {
		coin    string
		wantIdx int
		wantSz  int
		wantErr bool
	}{
		{"PURR/USDC", 10001, 2, false},
		{"PURR", 10001, 2, false},
		{"purr", 10001, 2, false},
		{"@1", 10001, 2, false},
		{"@2", 10002, 4, false},
		{"@999", 0, 0, true},
		{"NOTFOUND", 0, 0, true},
	}

	for _, tt := range tests {
		idx, sz, err := resolveSpotAssetIndex(meta, tt.coin)
		if tt.wantErr {
			if err == nil {
				t.Errorf("resolveSpotAssetIndex(%q) expected error", tt.coin)
			}
			continue
		}
		if err != nil {
			t.Errorf("resolveSpotAssetIndex(%q) unexpected error: %v", tt.coin, err)
			continue
		}
		if idx != tt.wantIdx || sz != tt.wantSz {
			t.Errorf("resolveSpotAssetIndex(%q) = (%d, %d), want (%d, %d)", tt.coin, idx, sz, tt.wantIdx, tt.wantSz)
		}
	}
}

func TestResolveCoinForAPI(t *testing.T) {
	meta := &hyperliquid.SpotMeta{
		Universe: []hyperliquid.SpotPair{
			{Name: "PURR/USDC", Index: 1, IsCanonical: true},
			{Name: "HYPE/USDC", Index: 2, IsCanonical: false},
		},
	}

	tests := []struct {
		coin   string
		isSpot bool
		want   string
	}{
		{"ETH", false, "ETH"},
		{"PURR/USDC", true, "PURR/USDC"},
		{"PURR", true, "PURR/USDC"},
		{"HYPE", true, "@2"},
		{"@5", true, "@5"},
		{"NOTFOUND", true, "NOTFOUND"},
	}

	for _, tt := range tests {
		got := resolveCoinForAPI(tt.coin, tt.isSpot, meta)
		if got != tt.want {
			t.Errorf("resolveCoinForAPI(%q, %v) = %q, want %q", tt.coin, tt.isSpot, got, tt.want)
		}
	}
}
