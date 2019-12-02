package dbft

import (
	"crypto/rand"
	"testing"

	"github.com/CityOfZion/neo-go/pkg/util"
	"github.com/nspcc-dev/dbft/block"
	"github.com/nspcc-dev/dbft/crypto"
	"github.com/nspcc-dev/dbft/payload"
	"github.com/stretchr/testify/require"
)

type Payload = payload.ConsensusPayload

type testState struct {
	myIndex    int
	count      int
	privs      []crypto.PrivateKey
	pubs       []crypto.PublicKey
	ch         []Payload
	currHeight uint32
	currHash   util.Uint256
}

func TestDBFT_OnStartPrimarySendPrepareRequest(t *testing.T) {
	s := newTestState(2, 7)

	t.Run("backup sends nothing on start", func(t *testing.T) {
		s.currHeight = 0
		service := New(s.getOptions()...)

		service.Start()
		require.Nil(t, s.tryRecv())
	})

	t.Run("primary send PrepareRequest on start", func(t *testing.T) {
		s.currHeight = 1
		service := New(s.getOptions()...)

		service.Start()
		p := s.tryRecv()
		require.NotNil(t, p)
		require.Equal(t, payload.PrepareRequestType, p.Type())
		require.EqualValues(t, 2, p.Height())
		require.EqualValues(t, 0, p.ViewNumber())
		require.NotNil(t, p.Payload())
		require.Equal(t, s.currHash, p.PrevHash())
		require.EqualValues(t, 2, p.ValidatorIndex())
	})
}

func newTestState(myIndex int, count int) *testState {
	s := &testState{
		myIndex: myIndex,
		count:   count,
	}

	s.privs, s.pubs = getTestValidators(count)

	return s
}

func (s *testState) tryRecv() Payload {
	if len(s.ch) == 0 {
		return nil
	}

	p := s.ch[0]
	s.ch = s.ch[1:]

	return p
}

func (s *testState) getOptions() []Option {
	return []Option{
		WithCurrentHeight(func() uint32 { return s.currHeight }),
		WithCurrentBlockHash(func() util.Uint256 { return s.currHash }),
		WithGetValidators(func(...block.Transaction) []crypto.PublicKey { return s.pubs }),
		WithKeyPair(s.privs[s.myIndex], s.pubs[s.myIndex]),
		WithBroadcast(func(p Payload) { s.ch = append(s.ch, p) }),
	}
}

func getTestValidators(n int) (privs []crypto.PrivateKey, pubs []crypto.PublicKey) {
	for i := 0; i < n; i++ {
		priv, pub := crypto.Generate(rand.Reader)
		privs = append(privs, priv)
		pubs = append(pubs, pub)
	}

	return
}
