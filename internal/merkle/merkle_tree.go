package merkle

import (
	"github.com/nspcc-dev/dbft/internal/crypto"
)

type (
	// Tree represents a merkle tree with specified depth.
	Tree struct {
		Depth int

		root *TreeNode
	}

	// TreeNode represents inner node of a merkle tree.
	TreeNode struct {
		Hash   crypto.Uint256
		Parent *TreeNode
		Left   *TreeNode
		Right  *TreeNode
	}
)

// NewMerkleTree returns new merkle tree built on hashes.
func NewMerkleTree(hashes ...crypto.Uint256) *Tree {
	if len(hashes) == 0 {
		return nil
	}

	nodes := make([]TreeNode, len(hashes))
	for i := range nodes {
		nodes[i].Hash = hashes[i]
	}

	mt := &Tree{root: buildTree(nodes...)}
	mt.Depth = 1

	for node := mt.root; node.Left != nil; node = node.Left {
		mt.Depth++
	}

	return mt
}

// Root returns m's root.
func (m *Tree) Root() *TreeNode {
	return m.root
}

func buildTree(leaves ...TreeNode) *TreeNode {
	l := len(leaves)
	if l == 1 {
		return &leaves[0]
	}

	parents := make([]TreeNode, (l+1)/2)
	for i := 0; i < len(parents); i++ {
		parents[i].Left = &leaves[i*2]
		leaves[i*2].Parent = &parents[i]

		if i*2+1 == l {
			parents[i].Right = parents[i].Left
		} else {
			parents[i].Right = &leaves[i*2+1]
			leaves[i*2+1].Parent = &parents[i]
		}

		data := append(parents[i].Left.Hash[:], parents[i].Right.Hash[:]...)
		parents[i].Hash = crypto.Hash256(data)
	}

	return buildTree(parents...)
}

// IsLeaf returns true iff n is a leaf.
func (n *TreeNode) IsLeaf() bool { return n.Left == nil && n.Right == nil }

// IsRoot returns true iff n is a root.
func (n *TreeNode) IsRoot() bool { return n.Parent == nil }
