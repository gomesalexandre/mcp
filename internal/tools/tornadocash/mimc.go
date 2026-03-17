package tornadocash

import (
	"math/big"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/iden3/go-iden3-crypto/constants"
	"github.com/iden3/go-iden3-crypto/ff"
)

const mimcSpongeRounds = 220

var mimcConstants []*ff.Element

func init() {
	mimcConstants = generateMiMCSpongeConstants()
}

func generateMiMCSpongeConstants() []*ff.Element {
	cts := make([]*ff.Element, mimcSpongeRounds)
	cts[0] = ff.NewElement()

	c := crypto.Keccak256([]byte("mimcsponge"))
	for i := 1; i < mimcSpongeRounds; i++ {
		c = crypto.Keccak256(c)
		n := new(big.Int).SetBytes(c)
		n.Mod(n, constants.Q)
		cts[i] = ff.NewElement().SetBigInt(n)
	}
	cts[mimcSpongeRounds-1] = ff.NewElement()
	return cts
}

// mimcFeistel performs one full MiMCSponge Feistel permutation (220 rounds, x^5).
func mimcFeistel(xL, xR *ff.Element) (*ff.Element, *ff.Element) {
	outL := ff.NewElement().Set(xL)
	outR := ff.NewElement().Set(xR)

	for i := 0; i < mimcSpongeRounds; i++ {
		t := ff.NewElement().Set(outL)
		if i > 0 {
			t.Add(t, mimcConstants[i])
		}
		t2 := ff.NewElement().Square(t)
		t4 := ff.NewElement().Square(t2)
		t5 := ff.NewElement().Mul(t4, t)

		if i < mimcSpongeRounds-1 {
			newL := ff.NewElement().Add(outR, t5)
			outR = outL
			outL = newL
		} else {
			outR.Add(outR, t5)
		}
	}

	return outL, outR
}

// MiMCSpongeHash computes MiMCSponge(left, right) as used by the Tornado Cash
// Merkle tree. Returns a single field element (the squeezed output).
func MiMCSpongeHash(left, right *big.Int) *big.Int {
	xL := ff.NewElement().SetBigInt(left)
	xR := ff.NewElement()

	xL, xR = mimcFeistel(xL, xR)

	rInput := ff.NewElement().SetBigInt(right)
	xL.Add(xL, rInput)

	xL, _ = mimcFeistel(xL, xR)

	result := new(big.Int)
	xL.ToBigIntRegular(result)
	return result
}
