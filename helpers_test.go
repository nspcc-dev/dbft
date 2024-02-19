package dbft

import (
	"testing"

	"github.com/nspcc-dev/dbft/crypto"
	"github.com/nspcc-dev/dbft/payload"
	"github.com/stretchr/testify/require"
)

func TestMessageCache(t *testing.T) {
	c := newCache[crypto.Uint256, crypto.Uint160]()

	p1 := payload.NewConsensusPayload()
	p1.SetHeight(3)
	p1.SetType(PrepareRequestType)
	c.addMessage(p1)

	p2 := payload.NewConsensusPayload()
	p2.SetHeight(4)
	p2.SetType(ChangeViewType)
	c.addMessage(p2)

	p3 := payload.NewConsensusPayload()
	p3.SetHeight(4)
	p3.SetType(CommitType)
	c.addMessage(p3)

	box := c.getHeight(3)
	require.Len(t, box.chViews, 0)
	require.Len(t, box.prepare, 1)
	require.Len(t, box.commit, 0)

	box = c.getHeight(4)
	require.Len(t, box.chViews, 1)
	require.Len(t, box.prepare, 0)
	require.Len(t, box.commit, 1)
}
