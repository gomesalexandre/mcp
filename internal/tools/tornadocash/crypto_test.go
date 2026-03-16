package tornadocash

import (
	"encoding/hex"
	"testing"
)

func TestPedersenHash_Hello(t *testing.T) {
	// circomlibjs test vector: PedersenHash("Hello") packed point
	expectedPackedHex := "0e90d7d613ab8b5ea7f4f8bc537db6bb0fa2e5e97bbac1c1f609ef9e6a35fd8b"

	point, err := pedersenHashPoint([]byte("Hello"))
	if err != nil {
		t.Fatalf("pedersenHashPoint: %v", err)
	}

	packed := point.Compress()
	gotHex := hex.EncodeToString(packed[:])
	if gotHex != expectedPackedHex {
		t.Fatalf("packed point mismatch:\n  got:  %s\n  want: %s", gotHex, expectedPackedHex)
	}
}

func TestPedersenHash_XCoordinate(t *testing.T) {
	// PedersenHash returns X coordinate of the point
	x, err := PedersenHash([]byte("Hello"))
	if err != nil {
		t.Fatalf("PedersenHash: %v", err)
	}
	if x == nil || x.Sign() == 0 {
		t.Fatal("expected non-zero X coordinate")
	}

	// verify consistency: X from PedersenHash equals X from full point
	point, err := pedersenHashPoint([]byte("Hello"))
	if err != nil {
		t.Fatalf("pedersenHashPoint: %v", err)
	}
	if x.Cmp(point.X) != 0 {
		t.Fatalf("X coordinate mismatch: PedersenHash=%s, point.X=%s", x, point.X)
	}
}

func TestBasePointsInSubGroup(t *testing.T) {
	for i, bp := range basePoints {
		if !bp.InCurve() {
			t.Errorf("base point %d not on curve", i)
		}
		if !bp.InSubGroup() {
			t.Errorf("base point %d not in subgroup", i)
		}
	}
}

func TestGenerateNote(t *testing.T) {
	secret, nullifier, err := GenerateNote()
	if err != nil {
		t.Fatalf("GenerateNote: %v", err)
	}
	if len(secret) != 31 {
		t.Fatalf("secret length: got %d, want 31", len(secret))
	}
	if len(nullifier) != 31 {
		t.Fatalf("nullifier length: got %d, want 31", len(nullifier))
	}

	// two calls should produce different values
	secret2, nullifier2, err := GenerateNote()
	if err != nil {
		t.Fatalf("GenerateNote second call: %v", err)
	}
	if hex.EncodeToString(secret) == hex.EncodeToString(secret2) {
		t.Fatal("two GenerateNote calls produced identical secrets")
	}
	if hex.EncodeToString(nullifier) == hex.EncodeToString(nullifier2) {
		t.Fatal("two GenerateNote calls produced identical nullifiers")
	}
}

func TestPedersenCommitment(t *testing.T) {
	secret, nullifier, err := GenerateNote()
	if err != nil {
		t.Fatalf("GenerateNote: %v", err)
	}

	commitment, err := PedersenCommitment(nullifier, secret)
	if err != nil {
		t.Fatalf("PedersenCommitment: %v", err)
	}
	if commitment == nil || commitment.Sign() == 0 {
		t.Fatal("expected non-zero commitment")
	}

	// same inputs should produce the same commitment
	commitment2, err := PedersenCommitment(nullifier, secret)
	if err != nil {
		t.Fatalf("PedersenCommitment second call: %v", err)
	}
	if commitment.Cmp(commitment2) != 0 {
		t.Fatal("same inputs produced different commitments")
	}
}

func TestNullifierHash(t *testing.T) {
	_, nullifier, err := GenerateNote()
	if err != nil {
		t.Fatalf("GenerateNote: %v", err)
	}

	nh, err := NullifierHash(nullifier)
	if err != nil {
		t.Fatalf("NullifierHash: %v", err)
	}
	if nh == nil || nh.Sign() == 0 {
		t.Fatal("expected non-zero nullifier hash")
	}

	// same input should produce same hash
	nh2, err := NullifierHash(nullifier)
	if err != nil {
		t.Fatalf("NullifierHash second call: %v", err)
	}
	if nh.Cmp(nh2) != 0 {
		t.Fatal("same nullifier produced different hashes")
	}
}

func TestPedersenCommitment_InvalidInputs(t *testing.T) {
	_, err := PedersenCommitment(make([]byte, 30), make([]byte, 31))
	if err == nil {
		t.Fatal("expected error for 30-byte nullifier")
	}

	_, err = PedersenCommitment(make([]byte, 31), make([]byte, 32))
	if err == nil {
		t.Fatal("expected error for 32-byte secret")
	}
}

func TestNullifierHash_InvalidInput(t *testing.T) {
	_, err := NullifierHash(make([]byte, 30))
	if err == nil {
		t.Fatal("expected error for 30-byte nullifier")
	}
}

func TestBigIntToBytes32(t *testing.T) {
	x, err := PedersenHash([]byte("Hello"))
	if err != nil {
		t.Fatalf("PedersenHash: %v", err)
	}

	b := BigIntToBytes32(x)
	if len(b) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(b))
	}

	// leading bytes should be zero-padded if x < 2^248
	gotHex := hex.EncodeToString(b[:])
	if len(gotHex) != 64 {
		t.Fatalf("expected 64 hex chars, got %d", len(gotHex))
	}
}
