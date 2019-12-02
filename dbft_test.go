package dbft

import (
	"crypto/rand"
	"encoding/binary"
	"testing"

	"github.com/CityOfZion/neo-go/pkg/util"
	"github.com/nspcc-dev/dbft/block"
	"github.com/nspcc-dev/dbft/crypto"
	"github.com/nspcc-dev/dbft/payload"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
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
	pool       *testPool
}

type (
	testTx   uint64
	testPool struct {
		storage map[util.Uint256]testTx
	}
)

const debugTests = false

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

func TestDBFT_OnReceiveRequestSendResponse(t *testing.T) {
	s := newTestState(2, 7)

	t.Run("receive request from primary", func(t *testing.T) {
		s.currHeight = 4
		service := New(s.getOptions()...)
		txs := []testTx{1}
		s.pool.Add(txs[0])

		p := s.getPrepareRequest(5, txs[0].Hash())

		service.Start()
		service.OnReceive(p)

		resp := s.tryRecv()
		require.NotNil(t, resp)
		require.Equal(t, payload.PrepareResponseType, resp.Type())
		require.EqualValues(t, s.currHeight+1, resp.Height())
		require.EqualValues(t, 0, resp.ViewNumber())
		require.Equal(t, s.currHash, resp.PrevHash())
		require.EqualValues(t, s.myIndex, resp.ValidatorIndex())
		require.NotNil(t, resp.Payload())
		require.Equal(t, p.Hash(), resp.GetPrepareResponse().PreparationHash())

		// do nothing on second receive
		service.OnReceive(p)
		require.Nil(t, s.tryRecv())
	})

	t.Run("receive invalid prepare request", func(t *testing.T) {
		s.currHeight = 4
		service := New(s.getOptions()...)
		txs := []testTx{1, 2}
		s.pool.Add(txs[0])

		service.Start()

		t.Run("wrong primary index", func(t *testing.T) {
			p := s.getPrepareRequest(4, txs[0].Hash())
			service.OnReceive(p)
			require.Nil(t, s.tryRecv())
		})

		t.Run("old height", func(t *testing.T) {
			p := s.getPrepareRequest(5, txs[0].Hash())
			p.SetHeight(3)
			service.OnReceive(p)
			require.Nil(t, s.tryRecv())
		})

		t.Run("does not have all transactions", func(t *testing.T) {
			p := s.getPrepareRequest(5, txs[0].Hash(), txs[1].Hash())
			service.OnReceive(p)
			require.Nil(t, s.tryRecv())

			// do nothing with already present transaction
			service.OnTransaction(txs[0])
			require.Nil(t, s.tryRecv())

			service.OnTransaction(txs[1])
			resp := s.tryRecv()
			require.NotNil(t, resp)
			require.Equal(t, payload.PrepareResponseType, resp.Type())
			require.EqualValues(t, s.currHeight+1, resp.Height())
			require.EqualValues(t, 0, resp.ViewNumber())
			require.Equal(t, s.currHash, resp.PrevHash())
			require.EqualValues(t, s.myIndex, resp.ValidatorIndex())
			require.NotNil(t, resp.Payload())
			require.Equal(t, p.Hash(), resp.GetPrepareResponse().PreparationHash())

			// do not send response twice
			service.OnTransaction(txs[1])
			require.Nil(t, s.tryRecv())
		})
	})
}

func TestDBFT_OnReceiveCommit(t *testing.T) {
	s := newTestState(2, 4)
	t.Run("send commit after enough responses", func(t *testing.T) {
		s.currHeight = 1
		service := New(s.getOptions()...)
		service.Start()

		req := s.tryRecv()
		require.NotNil(t, req)

		resp := s.getPrepareResponse(1, req.Hash())
		service.OnReceive(resp)
		require.Nil(t, s.tryRecv())

		resp = s.getPrepareResponse(0, req.Hash())
		service.OnReceive(resp)

		cm := s.tryRecv()
		require.NotNil(t, cm)
		require.Equal(t, payload.CommitType, cm.Type())
		require.EqualValues(t, s.currHeight+1, cm.Height())
		require.EqualValues(t, 0, cm.ViewNumber())
		require.Equal(t, s.currHash, cm.PrevHash())
		require.EqualValues(t, s.myIndex, cm.ValidatorIndex())
		require.NotNil(t, cm.Payload())

		pub := s.pubs[s.myIndex]
		require.NoError(t, service.header.Verify(pub, cm.GetCommit().Signature()))
	})
}

func TestDBFT_OnReceiveRecoveryRequest(t *testing.T) {
	s := newTestState(2, 4)
	t.Run("send recovery message", func(t *testing.T) {
		s.currHeight = 1
		service := New(s.getOptions()...)
		service.Start()

		req := s.tryRecv()
		require.NotNil(t, req)

		resp := s.getPrepareResponse(1, req.Hash())
		service.OnReceive(resp)
		require.Nil(t, s.tryRecv())

		resp = s.getPrepareResponse(0, req.Hash())
		service.OnReceive(resp)
		cm := s.tryRecv()
		require.NotNil(t, cm)

		rr := s.getRecoveryRequest(3)
		service.OnReceive(rr)
		rm := s.tryRecv()
		require.NotNil(t, rm)
		require.Equal(t, payload.RecoveryMessageType, rm.Type())

		other := s.copyWithIndex(3)
		srv2 := New(other.getOptions()...)
		srv2.Start()
		srv2.OnReceive(rm)

		r2 := other.tryRecv()
		require.NotNil(t, r2)
		require.Equal(t, payload.PrepareResponseType, r2.Type())

		cm2 := other.tryRecv()
		require.NotNil(t, cm2)
		require.Equal(t, payload.CommitType, cm2.Type())
		pub := other.pubs[other.myIndex]
		require.NoError(t, service.header.Verify(pub, cm2.GetCommit().Signature()))

		// send commit once during recovery
		require.Nil(t, s.tryRecv())
	})
}

func (s testState) getRecoveryRequest(from uint16) Payload {
	p := s.getPayload(from)
	p.SetType(payload.RecoveryRequestType)
	p.SetPayload(payload.NewRecoveryRequest())

	return p
}

func (s testState) getPrepareResponse(from uint16, phash util.Uint256) Payload {
	resp := payload.NewPrepareResponse()
	resp.SetPreparationHash(phash)

	p := s.getPayload(from)
	p.SetType(payload.PrepareResponseType)
	p.SetPayload(resp)

	return p
}

func (s testState) getPrepareRequest(from uint16, hashes ...util.Uint256) Payload {
	req := payload.NewPrepareRequest()
	req.SetTransactionHashes(hashes)

	p := s.getPayload(from)
	p.SetType(payload.PrepareRequestType)
	p.SetPayload(req)

	return p
}

func (s testState) getPayload(from uint16) Payload {
	p := payload.NewConsensusPayload()
	p.SetPrevHash(s.currHash)
	p.SetHeight(s.currHeight + 1)
	p.SetValidatorIndex(from)

	return p
}

func newTestState(myIndex int, count int) *testState {
	s := &testState{
		myIndex: myIndex,
		count:   count,
		pool:    newTestPool(),
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

func (s testState) copyWithIndex(myIndex int) *testState {
	return &testState{
		myIndex:    myIndex,
		count:      s.count,
		privs:      s.privs,
		pubs:       s.pubs,
		currHeight: s.currHeight,
		currHash:   s.currHash,
		pool:       newTestPool(),
	}
}

func (s *testState) getOptions() []Option {
	opts := []Option{
		WithCurrentHeight(func() uint32 { return s.currHeight }),
		WithCurrentBlockHash(func() util.Uint256 { return s.currHash }),
		WithGetValidators(func(...block.Transaction) []crypto.PublicKey { return s.pubs }),
		WithKeyPair(s.privs[s.myIndex], s.pubs[s.myIndex]),
		WithBroadcast(func(p Payload) { s.ch = append(s.ch, p) }),
		WithGetTx(s.pool.Get),
	}

	if debugTests {
		cfg := zap.NewDevelopmentConfig()
		cfg.DisableStacktrace = true
		logger, _ := cfg.Build()
		opts = append(opts, WithLogger(logger))
	}

	return opts
}

func getTestValidators(n int) (privs []crypto.PrivateKey, pubs []crypto.PublicKey) {
	for i := 0; i < n; i++ {
		priv, pub := crypto.Generate(rand.Reader)
		privs = append(privs, priv)
		pubs = append(pubs, pub)
	}

	return
}

func (tx testTx) Hash() (h util.Uint256) {
	binary.LittleEndian.PutUint64(h[:], uint64(tx))
	return
}

func newTestPool() *testPool {
	return &testPool{
		storage: make(map[util.Uint256]testTx),
	}
}

func (p *testPool) Add(tx testTx) {
	p.storage[tx.Hash()] = tx
}

func (p *testPool) Get(h util.Uint256) block.Transaction {
	if tx, ok := p.storage[h]; ok {
		return tx
	}

	return nil
}
