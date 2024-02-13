package dbft

import (
	"testing"

	"github.com/nspcc-dev/neo-go/pkg/util"
	"github.com/stretchr/testify/require"

	"github.com/nspcc-dev/dbft/payload"
)

func TestMessageCache(t *testing.T) {
	c := newCache[util.Uint256, util.Uint160]()

	p1 := payload.NewConsensusPayload()
	p1.SetHeight(3)
	p1.SetType(payload.PrepareRequestType)
	c.addMessage(p1)

	p2 := payload.NewConsensusPayload()
	p2.SetHeight(4)
	p2.SetType(payload.ChangeViewType)
	c.addMessage(p2)

	p3 := payload.NewConsensusPayload()
	p3.SetHeight(4)
	p3.SetType(payload.CommitType)
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
