package tornadocash

import (
	"math/big"
	"testing"
)

func mustInsert(t *testing.T, tree *MerkleTree, leaf *big.Int) {
	t.Helper()
	err := tree.Insert(leaf)
	if err != nil {
		t.Fatalf("Insert(%s): %v", leaf, err)
	}
}

func TestNewMerkleTree_ZeroValues(t *testing.T) {
	tree := NewMerkleTree(merkleTreeDepth)

	if tree.Depth != merkleTreeDepth {
		t.Fatalf("depth = %d, want %d", tree.Depth, merkleTreeDepth)
	}
	if len(tree.Zeros) != merkleTreeDepth+1 {
		t.Fatalf("zeros length = %d, want %d", len(tree.Zeros), merkleTreeDepth+1)
	}

	expected, _ := new(big.Int).SetString("21663839004416932945382355908790599225266501822907911457504978515578255421292", 10)
	if tree.Zeros[0].Cmp(expected) != 0 {
		t.Fatalf("zeros[0] = %s, want %s", tree.Zeros[0], expected)
	}

	for i := 1; i <= merkleTreeDepth; i++ {
		want := MiMCSpongeHash(tree.Zeros[i-1], tree.Zeros[i-1])
		if tree.Zeros[i].Cmp(want) != 0 {
			t.Fatalf("zeros[%d] mismatch", i)
		}
	}
}

func TestMerkleTree_EmptyRoot(t *testing.T) {
	tree := NewMerkleTree(merkleTreeDepth)
	root := tree.Root()

	if root.Cmp(tree.Zeros[merkleTreeDepth]) != 0 {
		t.Fatalf("empty root = %s, want zeros[%d] = %s", root, merkleTreeDepth, tree.Zeros[merkleTreeDepth])
	}
}

func TestMerkleTree_InsertChangesRoot(t *testing.T) {
	tree := NewMerkleTree(merkleTreeDepth)
	emptyRoot := new(big.Int).Set(tree.Root())

	mustInsert(t, tree, big.NewInt(42))
	newRoot := tree.Root()

	if newRoot.Cmp(emptyRoot) == 0 {
		t.Fatal("root should change after insert")
	}
}

func TestMerkleTree_InsertFull(t *testing.T) {
	tree := NewMerkleTree(2) // capacity = 4
	for i := 0; i < 4; i++ {
		mustInsert(t, tree, big.NewInt(int64(i+1)))
	}

	err := tree.Insert(big.NewInt(5))
	if err == nil {
		t.Fatal("Insert on full tree should return error")
	}
}

func TestMerkleTree_FindLeafIndex(t *testing.T) {
	tree := NewMerkleTree(merkleTreeDepth)
	leaves := []*big.Int{big.NewInt(100), big.NewInt(200), big.NewInt(300)}
	for _, l := range leaves {
		mustInsert(t, tree, l)
	}

	for i, l := range leaves {
		idx := tree.FindLeafIndex(l)
		if idx != i {
			t.Fatalf("FindLeafIndex(%s) = %d, want %d", l, idx, i)
		}
	}

	idx := tree.FindLeafIndex(big.NewInt(999))
	if idx != -1 {
		t.Fatalf("FindLeafIndex(999) = %d, want -1", idx)
	}
}

func TestMerkleTree_GetPath_VerifyRoundtrip(t *testing.T) {
	tree := NewMerkleTree(merkleTreeDepth)
	leaves := []*big.Int{big.NewInt(11), big.NewInt(22), big.NewInt(33), big.NewInt(44), big.NewInt(55)}
	for _, l := range leaves {
		mustInsert(t, tree, l)
	}

	for leafIdx, leaf := range leaves {
		root, pathElements, pathIndices := tree.GetPath(leafIdx)

		if root.Cmp(tree.Root()) != 0 {
			t.Fatalf("leaf %d: GetPath root != tree.Root()", leafIdx)
		}

		current := new(big.Int).Set(leaf)
		for i := 0; i < tree.Depth; i++ {
			if pathIndices[i] == 0 {
				current = MiMCSpongeHash(current, pathElements[i])
			} else {
				current = MiMCSpongeHash(pathElements[i], current)
			}
		}

		if current.Cmp(root) != 0 {
			t.Fatalf("leaf %d: recomputed root %s != expected %s", leafIdx, current, root)
		}
	}
}

func TestMerkleTree_GetPath_InvalidIndex(t *testing.T) {
	tree := NewMerkleTree(2)
	mustInsert(t, tree, big.NewInt(1))

	root, elems, indices := tree.GetPath(-1)
	if root != nil || elems != nil || indices != nil {
		t.Fatal("GetPath(-1) should return nils")
	}

	root, elems, indices = tree.GetPath(4) // capacity = 4, so index 4 is out of bounds
	if root != nil || elems != nil || indices != nil {
		t.Fatal("GetPath(4) on depth-2 tree should return nils")
	}
}

func TestMerkleTree_DeterministicRoot(t *testing.T) {
	tree1 := NewMerkleTree(merkleTreeDepth)
	tree2 := NewMerkleTree(merkleTreeDepth)

	leaves := []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)}
	for _, l := range leaves {
		mustInsert(t, tree1, l)
		mustInsert(t, tree2, l)
	}

	if tree1.Root().Cmp(tree2.Root()) != 0 {
		t.Fatalf("same leaves should produce same root: %s vs %s", tree1.Root(), tree2.Root())
	}
}

func TestMerkleTree_OrderMatters(t *testing.T) {
	tree1 := NewMerkleTree(merkleTreeDepth)
	mustInsert(t, tree1, big.NewInt(1))
	mustInsert(t, tree1, big.NewInt(2))

	tree2 := NewMerkleTree(merkleTreeDepth)
	mustInsert(t, tree2, big.NewInt(2))
	mustInsert(t, tree2, big.NewInt(1))

	if tree1.Root().Cmp(tree2.Root()) == 0 {
		t.Fatal("different insertion order should produce different roots")
	}
}

func TestMerkleTree_SmallDepth(t *testing.T) {
	tree := NewMerkleTree(2)
	mustInsert(t, tree, big.NewInt(10))
	mustInsert(t, tree, big.NewInt(20))
	mustInsert(t, tree, big.NewInt(30))
	mustInsert(t, tree, big.NewInt(40))

	h01 := MiMCSpongeHash(big.NewInt(10), big.NewInt(20))
	h23 := MiMCSpongeHash(big.NewInt(30), big.NewInt(40))
	expectedRoot := MiMCSpongeHash(h01, h23)

	if tree.Root().Cmp(expectedRoot) != 0 {
		t.Fatalf("depth-2 full tree root = %s, want %s", tree.Root(), expectedRoot)
	}
}

func TestMerkleTree_SmallDepth_Sparse(t *testing.T) {
	tree := NewMerkleTree(2)
	mustInsert(t, tree, big.NewInt(10))

	h0z := MiMCSpongeHash(big.NewInt(10), tree.Zeros[0])
	expectedRoot := MiMCSpongeHash(h0z, tree.Zeros[1])

	if tree.Root().Cmp(expectedRoot) != 0 {
		t.Fatalf("depth-2 sparse tree root = %s, want %s", tree.Root(), expectedRoot)
	}
}
