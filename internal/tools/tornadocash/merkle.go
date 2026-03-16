package tornadocash

import (
	"context"
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	merkleTreeDepth = 20
	logBatchSize    = 10000
)

var depositEventTopic = crypto.Keccak256Hash([]byte("Deposit(bytes32,uint32,uint256)"))

// MerkleTree is a fixed-depth sparse MiMCSponge Merkle tree matching the
// Tornado Cash on-chain structure. Empty subtrees are represented by
// precomputed zero values to avoid unnecessary hashing.
type MerkleTree struct {
	Depth  int
	Leaves []*big.Int
	Zeros  []*big.Int // zeros[i] = hash of empty subtree at height i
}

// NewMerkleTree creates a Merkle tree with precomputed zero values.
func NewMerkleTree(depth int) *MerkleTree {
	zeros := make([]*big.Int, depth+1)
	zeros[0], _ = new(big.Int).SetString("21663839004416932945382355908790599225266501822907911457504978515578255421292", 10)
	for i := 1; i <= depth; i++ {
		zeros[i] = MiMCSpongeHash(zeros[i-1], zeros[i-1])
	}
	return &MerkleTree{
		Depth: depth,
		Zeros: zeros,
	}
}

// Insert adds a leaf to the tree.
func (t *MerkleTree) Insert(leaf *big.Int) {
	t.Leaves = append(t.Leaves, new(big.Int).Set(leaf))
}

// getNode computes the hash at the given (height, index). Height 0 is the
// leaf level. Entire empty subtrees resolve to precomputed zeros.
func (t *MerkleTree) getNode(height, index int) *big.Int {
	if height == 0 {
		if index < len(t.Leaves) {
			return t.Leaves[index]
		}
		return t.Zeros[0]
	}

	firstLeaf := index * (1 << uint(height))
	if firstLeaf >= len(t.Leaves) {
		return t.Zeros[height]
	}

	left := t.getNode(height-1, index*2)
	right := t.getNode(height-1, index*2+1)
	return MiMCSpongeHash(left, right)
}

// Root computes the current Merkle root.
func (t *MerkleTree) Root() *big.Int {
	return t.getNode(t.Depth, 0)
}

// FindLeafIndex returns the index of the given commitment, or -1 if not found.
func (t *MerkleTree) FindLeafIndex(commitment *big.Int) int {
	for i, leaf := range t.Leaves {
		if leaf.Cmp(commitment) == 0 {
			return i
		}
	}
	return -1
}

// GetPath returns the Merkle proof path for the given leaf index.
// pathElements[i] is the sibling hash at height i (0 = leaf level).
// pathIndices[i] is the bit of leafIndex at position i (0 = left, 1 = right).
func (t *MerkleTree) GetPath(leafIndex int) (root *big.Int, pathElements []*big.Int, pathIndices []int) {
	pathElements = make([]*big.Int, t.Depth)
	pathIndices = make([]int, t.Depth)

	for i := 0; i < t.Depth; i++ {
		nodeIdx := leafIndex >> uint(i)
		siblingIdx := nodeIdx ^ 1
		pathElements[i] = t.getNode(i, siblingIdx)
		pathIndices[i] = nodeIdx & 1
	}

	root = t.Root()
	return root, pathElements, pathIndices
}

// BuildTreeFromDeposits fetches all Deposit events from the pool contract and
// builds the Merkle tree. Use fromBlock to avoid scanning from genesis.
func BuildTreeFromDeposits(ctx context.Context, client *ethclient.Client, poolAddress common.Address, fromBlock uint64) (*MerkleTree, error) {
	tree := NewMerkleTree(merkleTreeDepth)

	currentBlock, err := client.BlockNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("get block number: %w", err)
	}

	for from := fromBlock; from <= currentBlock; from += logBatchSize {
		to := from + logBatchSize - 1
		if to > currentBlock {
			to = currentBlock
		}

		query := ethereum.FilterQuery{
			FromBlock: new(big.Int).SetUint64(from),
			ToBlock:   new(big.Int).SetUint64(to),
			Addresses: []common.Address{poolAddress},
			Topics:    [][]common.Hash{{depositEventTopic}},
		}

		logs, err := client.FilterLogs(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("filter logs %d-%d: %w", from, to, err)
		}

		sort.Slice(logs, func(i, j int) bool {
			if logs[i].BlockNumber != logs[j].BlockNumber {
				return logs[i].BlockNumber < logs[j].BlockNumber
			}
			return logs[i].Index < logs[j].Index
		})

		for _, log := range logs {
			if len(log.Topics) < 2 {
				continue
			}
			commitment := new(big.Int).SetBytes(log.Topics[1].Bytes())
			tree.Insert(commitment)
		}
	}

	return tree, nil
}
