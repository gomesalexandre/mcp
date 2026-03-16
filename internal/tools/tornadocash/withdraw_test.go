package tornadocash

import (
	"bytes"
	"math/big"
	"testing"

	ethcommon "github.com/ethereum/go-ethereum/common"
)

func TestEncodeWithdrawCalldata_Typical(t *testing.T) {
	proof := make([]byte, 256) // 8 x uint256 (Groth16 proof)
	for i := range proof {
		proof[i] = byte(i)
	}
	root := make([]byte, 32)
	root[31] = 0x01
	nh := make([]byte, 32)
	nh[31] = 0x02
	recipient := ethcommon.HexToAddress("0x1111111111111111111111111111111111111111")
	relayer := ethcommon.HexToAddress("0x2222222222222222222222222222222222222222")
	fee := big.NewInt(1000)
	refund := big.NewInt(0)

	calldata := encodeWithdrawCalldata(proof, root, nh, recipient, relayer, fee, refund)

	// selector
	if !bytes.Equal(calldata[:4], withdrawSelector) {
		t.Fatalf("selector mismatch: got %x, want %x", calldata[:4], withdrawSelector)
	}

	headSize := 7 * 32
	proofPadded := 256 // 256 is already multiple of 32
	expectedLen := 4 + headSize + 32 + proofPadded
	if len(calldata) != expectedLen {
		t.Fatalf("length = %d, want %d", len(calldata), expectedLen)
	}

	// slot 0: offset to bytes data = headSize = 224 = 0xE0
	slot0 := calldata[4 : 4+32]
	wantOffset := padLeft(big.NewInt(int64(headSize)).Bytes(), 32)
	if !bytes.Equal(slot0, wantOffset) {
		t.Fatalf("slot 0 (offset) = %x, want %x", slot0, wantOffset)
	}

	// slot 1: root
	slot1 := calldata[4+32 : 4+64]
	if !bytes.Equal(slot1, root) {
		t.Fatalf("slot 1 (root) mismatch")
	}

	// slot 2: nullifierHash
	slot2 := calldata[4+64 : 4+96]
	if !bytes.Equal(slot2, nh) {
		t.Fatalf("slot 2 (nullifierHash) mismatch")
	}

	// slot 3: recipient (20 bytes right-aligned in 32-byte slot)
	slot3 := calldata[4+96 : 4+128]
	if !bytes.Equal(slot3[:12], make([]byte, 12)) {
		t.Fatalf("slot 3 leading zeros missing")
	}
	if !bytes.Equal(slot3[12:], recipient.Bytes()) {
		t.Fatalf("slot 3 (recipient) mismatch")
	}

	// slot 4: relayer
	slot4 := calldata[4+128 : 4+160]
	if !bytes.Equal(slot4[12:], relayer.Bytes()) {
		t.Fatalf("slot 4 (relayer) mismatch")
	}

	// slot 5: fee
	slot5 := calldata[4+160 : 4+192]
	wantFee := padLeft(fee.Bytes(), 32)
	if !bytes.Equal(slot5, wantFee) {
		t.Fatalf("slot 5 (fee) = %x, want %x", slot5, wantFee)
	}

	// slot 6: refund (zero)
	slot6 := calldata[4+192 : 4+224]
	if !bytes.Equal(slot6, make([]byte, 32)) {
		t.Fatalf("slot 6 (refund) should be zero, got %x", slot6)
	}

	// data section: proof length
	dataStart := 4 + headSize
	proofLen := new(big.Int).SetBytes(calldata[dataStart : dataStart+32])
	if proofLen.Int64() != 256 {
		t.Fatalf("proof length = %d, want 256", proofLen.Int64())
	}

	// data section: proof content
	proofData := calldata[dataStart+32 : dataStart+32+256]
	if !bytes.Equal(proofData, proof) {
		t.Fatalf("proof data mismatch")
	}
}

func TestEncodeWithdrawCalldata_ZeroFeeRefund(t *testing.T) {
	proof := make([]byte, 256)
	root := make([]byte, 32)
	nh := make([]byte, 32)
	recipient := ethcommon.HexToAddress("0x0000000000000000000000000000000000000001")
	relayer := ethcommon.Address{}
	fee := big.NewInt(0)
	refund := big.NewInt(0)

	calldata := encodeWithdrawCalldata(proof, root, nh, recipient, relayer, fee, refund)

	// fee and refund slots should be all zeros
	slot5 := calldata[4+160 : 4+192]
	slot6 := calldata[4+192 : 4+224]
	if !bytes.Equal(slot5, make([]byte, 32)) {
		t.Fatalf("fee slot should be zero")
	}
	if !bytes.Equal(slot6, make([]byte, 32)) {
		t.Fatalf("refund slot should be zero")
	}
}

func TestEncodeWithdrawCalldata_MaxUint256Fee(t *testing.T) {
	proof := make([]byte, 256)
	root := make([]byte, 32)
	nh := make([]byte, 32)
	recipient := ethcommon.HexToAddress("0x0000000000000000000000000000000000000001")
	relayer := ethcommon.Address{}

	maxUint256 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	fee := new(big.Int).Set(maxUint256)
	refund := new(big.Int).Set(maxUint256)

	calldata := encodeWithdrawCalldata(proof, root, nh, recipient, relayer, fee, refund)

	slot5 := calldata[4+160 : 4+192]
	slot6 := calldata[4+192 : 4+224]

	wantMax := make([]byte, 32)
	for i := range wantMax {
		wantMax[i] = 0xFF
	}
	if !bytes.Equal(slot5, wantMax) {
		t.Fatalf("fee slot should be max uint256, got %x", slot5)
	}
	if !bytes.Equal(slot6, wantMax) {
		t.Fatalf("refund slot should be max uint256, got %x", slot6)
	}
}

func TestEncodeWithdrawCalldata_OddProofLength(t *testing.T) {
	// proof not a multiple of 32 — should be right-padded
	proof := make([]byte, 100)
	for i := range proof {
		proof[i] = 0xAB
	}
	root := make([]byte, 32)
	nh := make([]byte, 32)
	recipient := ethcommon.Address{}
	relayer := ethcommon.Address{}
	fee := big.NewInt(0)
	refund := big.NewInt(0)

	calldata := encodeWithdrawCalldata(proof, root, nh, recipient, relayer, fee, refund)

	headSize := 7 * 32
	paddedProofLen := 128 // 100 padded to next multiple of 32
	expectedLen := 4 + headSize + 32 + paddedProofLen
	if len(calldata) != expectedLen {
		t.Fatalf("length = %d, want %d", len(calldata), expectedLen)
	}

	// proof length field should be 100 (actual), not 128 (padded)
	dataStart := 4 + headSize
	proofLen := new(big.Int).SetBytes(calldata[dataStart : dataStart+32])
	if proofLen.Int64() != 100 {
		t.Fatalf("proof length = %d, want 100", proofLen.Int64())
	}

	// first 100 bytes should be 0xAB
	proofData := calldata[dataStart+32 : dataStart+32+100]
	for i, b := range proofData {
		if b != 0xAB {
			t.Fatalf("proof byte %d = 0x%02x, want 0xAB", i, b)
		}
	}

	// padding bytes (100..127) should be zero
	padding := calldata[dataStart+32+100 : dataStart+32+128]
	if !bytes.Equal(padding, make([]byte, 28)) {
		t.Fatalf("padding should be zero, got %x", padding)
	}
}

func TestEncodeWithdrawCalldata_SmallProof(t *testing.T) {
	proof := []byte{0x01}
	root := make([]byte, 32)
	nh := make([]byte, 32)
	recipient := ethcommon.Address{}
	relayer := ethcommon.Address{}
	fee := big.NewInt(0)
	refund := big.NewInt(0)

	calldata := encodeWithdrawCalldata(proof, root, nh, recipient, relayer, fee, refund)

	headSize := 7 * 32
	// 1 byte padded to 32
	expectedLen := 4 + headSize + 32 + 32
	if len(calldata) != expectedLen {
		t.Fatalf("length = %d, want %d", len(calldata), expectedLen)
	}

	dataStart := 4 + headSize
	proofLen := new(big.Int).SetBytes(calldata[dataStart : dataStart+32])
	if proofLen.Int64() != 1 {
		t.Fatalf("proof length = %d, want 1", proofLen.Int64())
	}

	if calldata[dataStart+32] != 0x01 {
		t.Fatalf("proof byte 0 = 0x%02x, want 0x01", calldata[dataStart+32])
	}
	// remaining 31 bytes should be zero
	if !bytes.Equal(calldata[dataStart+33:dataStart+64], make([]byte, 31)) {
		t.Fatal("small proof padding should be zero")
	}
}

func TestPadLeft(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		size int
		want []byte
	}{
		{"exact", []byte{1, 2, 3}, 3, []byte{1, 2, 3}},
		{"pad", []byte{0xFF}, 4, []byte{0, 0, 0, 0xFF}},
		{"oversize trims from high end", []byte{0, 0, 1, 2}, 2, []byte{1, 2}},
		{"empty", []byte{}, 2, []byte{0, 0}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := padLeft(tc.in, tc.size)
			if !bytes.Equal(got, tc.want) {
				t.Fatalf("padLeft(%x, %d) = %x, want %x", tc.in, tc.size, got, tc.want)
			}
		})
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		name     string
		in       []byte
		multiple int
		want     []byte
	}{
		{"exact multiple", []byte{1, 2, 3, 4}, 4, []byte{1, 2, 3, 4}},
		{"needs pad", []byte{1, 2, 3}, 4, []byte{1, 2, 3, 0}},
		{"single byte", []byte{0xAB}, 32, append([]byte{0xAB}, make([]byte, 31)...)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := padRight(tc.in, tc.multiple)
			if !bytes.Equal(got, tc.want) {
				t.Fatalf("padRight(%x, %d) = %x, want %x", tc.in, tc.multiple, got, tc.want)
			}
		})
	}
}
