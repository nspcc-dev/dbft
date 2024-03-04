package merkle

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/nspcc-dev/dbft/internal/crypto"
	"github.com/stretchr/testify/require"
)

func TestNewMerkleTree(t *testing.T) {
	t.Run("empty tree must be nil", func(t *testing.T) {
		require.Nil(t, NewMerkleTree())
	})

	t.Run("merkle tree on 1 leave", func(t *testing.T) {
		h := crypto.Uint256{1, 2, 3, 4}
		mt := NewMerkleTree(h)
		require.NotNil(t, mt)
		require.Equal(t, 1, mt.Depth)
		require.Equal(t, h, mt.Root().Hash)
		require.True(t, mt.Root().IsLeaf())
	})

	t.Run("predefined tree on 4 leaves", func(t *testing.T) {
		hashes := make([]crypto.Uint256, 5)
		for i := 0; i < 5; i++ {
			hashes[i] = sha256.Sum256([]byte{byte(i)})
		}

		mt := NewMerkleTree(hashes...)
		require.NotNil(t, mt)
		require.Equal(t, 4, mt.Depth)

		expected, err := hex.DecodeString("f570734e3e3e401dad09b8f51499dfb2f631c803b88487ef65b88baa069430d0")
		require.NoError(t, err)
		require.Equal(t, expected, mt.Root().Hash[:])
	})
}

func TestTreeNode_IsLeaf(t *testing.T) {
	hashes := []crypto.Uint256{{1}, {2}, {3}}

	mt := NewMerkleTree(hashes...)
	require.NotNil(t, mt)
	require.True(t, mt.Root().IsRoot())
	require.False(t, mt.Root().IsLeaf())

	left := mt.Root().Left
	require.NotNil(t, left)
	require.False(t, left.IsRoot())
	require.False(t, left.IsLeaf())

	lleft := left.Left
	require.NotNil(t, lleft)
	require.False(t, lleft.IsRoot())
	require.True(t, lleft.IsLeaf())
}
