package tornadocash

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/decred/dcrd/crypto/blake256"
	"github.com/iden3/go-iden3-crypto/babyjub"
)

const (
	windowSize         = 4
	nWindowsPerSegment = 50
	bitsPerSegment     = windowSize * nWindowsPerSegment
	genPointPrefix     = "PedersenGenerator"
)

// base points are deterministic constants computed from the "PedersenGenerator" prefix
// using BLAKE-256 hashing. Only 3 are needed (covers up to 600 bits).
var basePoints []*babyjub.Point

func init() {
	basePoints = make([]*babyjub.Point, 3)
	for i := range basePoints {
		basePoints[i] = computeBasePoint(i)
	}
}

func computeBasePoint(idx int) *babyjub.Point {
	for tryIdx := 0; ; tryIdx++ {
		s := fmt.Sprintf("%s_%s_%s", genPointPrefix, padLeftZeros(idx, 32), padLeftZeros(tryIdx, 32))
		h := blake256.Sum256([]byte(s))
		h[31] &= 0xBF // clear bit 254 to ensure Y < field prime

		p, err := new(babyjub.Point).Decompress(h)
		if err != nil {
			continue
		}

		p8 := babyjub.NewPoint().Mul(big.NewInt(8), p)
		if !p8.InSubGroup() {
			continue
		}

		return p8
	}
}

func getBasePoint(idx int) *babyjub.Point {
	if idx < len(basePoints) {
		return basePoints[idx]
	}
	return computeBasePoint(idx)
}

func padLeftZeros(n, width int) string {
	s := fmt.Sprintf("%d", n)
	for len(s) < width {
		s = "0" + s
	}
	return s
}

func buffer2bits(buf []byte) []bool {
	bits := make([]bool, len(buf)*8)
	for i, b := range buf {
		for j := uint(0); j < 8; j++ {
			bits[i*8+int(j)] = (b>>j)&1 == 1
		}
	}
	return bits
}

// circomlib Pedersen hash: 4-bit windows, 50 windows per segment, each segment
// uses a dedicated base point. Within each window the first 3 bits encode a
// magnitude 1-8 and the 4th bit is a sign. The per-window values are combined
// into a single scalar per segment via powers of 32, then the segment scalar
// multiplies the base point. Results are accumulated via point addition.
func pedersenHashPoint(msg []byte) (*babyjub.Point, error) {
	bits := buffer2bits(msg)
	nBits := len(bits)
	if nBits == 0 {
		return nil, fmt.Errorf("empty message")
	}

	nSegments := (nBits-1)/bitsPerSegment + 1
	acc := babyjub.NewPointProjective()

	for s := 0; s < nSegments; s++ {
		nWindows := nWindowsPerSegment
		if s == nSegments-1 {
			remaining := nBits - (nSegments-1)*bitsPerSegment
			nWindows = (remaining-1)/windowSize + 1
		}

		escalar := big.NewInt(0)
		exp := big.NewInt(1)

		for w := 0; w < nWindows; w++ {
			o := s*bitsPerSegment + w*windowSize

			windowVal := big.NewInt(1)
			for b := 0; b < windowSize-1 && o < nBits; b++ {
				if bits[o] {
					windowVal.Add(windowVal, new(big.Int).Lsh(big.NewInt(1), uint(b)))
				}
				o++
			}

			if o < nBits {
				if bits[o] {
					windowVal.Neg(windowVal)
				}
				o++
			}

			term := new(big.Int).Mul(windowVal, exp)
			escalar.Add(escalar, term)
			exp.Lsh(exp, uint(windowSize+1))
		}

		if escalar.Sign() < 0 {
			escalar.Add(escalar, babyjub.SubOrder)
		}

		bp := getBasePoint(s)
		segPoint := babyjub.NewPoint().Mul(escalar, bp)
		segProj := segPoint.Projective()
		acc.Add(acc, segProj)
	}

	return acc.Affine(), nil
}

// PedersenHash computes the circomlib-compatible Pedersen hash and returns the
// X coordinate of the resulting BabyJubjub point.
func PedersenHash(msg []byte) (*big.Int, error) {
	p, err := pedersenHashPoint(msg)
	if err != nil {
		return nil, err
	}
	return p.X, nil
}

// GenerateNote creates random 31-byte secret and nullifier for a Tornado Cash note.
func GenerateNote() (secret, nullifier []byte, err error) {
	secret = make([]byte, 31)
	nullifier = make([]byte, 31)

	_, err = rand.Read(nullifier)
	if err != nil {
		return nil, nil, fmt.Errorf("generate nullifier: %w", err)
	}

	_, err = rand.Read(secret)
	if err != nil {
		return nil, nil, fmt.Errorf("generate secret: %w", err)
	}

	return secret, nullifier, nil
}

// PedersenCommitment computes PedersenHash(nullifier || secret) as used by
// Tornado Cash deposit circuits. Both slices must be exactly 31 bytes.
func PedersenCommitment(nullifier, secret []byte) (*big.Int, error) {
	if len(nullifier) != 31 || len(secret) != 31 {
		return nil, fmt.Errorf("nullifier and secret must each be 31 bytes")
	}
	preimage := make([]byte, 62)
	copy(preimage[:31], nullifier)
	copy(preimage[31:], secret)
	return PedersenHash(preimage)
}

// NullifierHash computes PedersenHash(nullifier) as used by Tornado Cash
// withdrawal circuits. The slice must be exactly 31 bytes.
func NullifierHash(nullifier []byte) (*big.Int, error) {
	if len(nullifier) != 31 {
		return nil, fmt.Errorf("nullifier must be exactly 31 bytes")
	}
	return PedersenHash(nullifier)
}

// BigIntToBytes32 converts a big.Int to a 32-byte big-endian representation.
func BigIntToBytes32(n *big.Int) [32]byte {
	var result [32]byte
	b := n.Bytes()
	copy(result[32-len(b):], b)
	return result
}
