package dbft

import (
	"crypto/rand"
	"encoding/binary"
	"testing"
	"time"

	"github.com/nspcc-dev/dbft/block"
	"github.com/nspcc-dev/dbft/crypto"
	"github.com/nspcc-dev/dbft/payload"
	"github.com/nspcc-dev/dbft/timer"
	"github.com/nspcc-dev/neo-go/pkg/util"
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
	blocks     []block.Block
	verify     func(b block.Block) bool
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

		t.Run("primary send ChangeView on timeout", func(t *testing.T) {
			service.OnTimeout(timer.HV{Height: s.currHeight + 1})

			// if there are many faulty must send RecoveryRequest
			cv := s.tryRecv()
			require.NotNil(t, cv)
			require.Equal(t, payload.RecoveryRequestType, cv.Type())
			require.Nil(t, s.tryRecv())

			// if all nodes are up must send ChangeView
			for i := range service.LastSeenMessage {
				service.LastSeenMessage[i] = &timer.HV{Height: s.currHeight + 1}
			}
			service.OnTimeout(timer.HV{Height: s.currHeight + 1})

			cv = s.tryRecv()
			require.NotNil(t, cv)
			require.Equal(t, payload.ChangeViewType, cv.Type())
			require.EqualValues(t, 1, cv.GetChangeView().NewViewNumber())
			require.Nil(t, s.tryRecv())
		})
	})
}

func TestDBFT_SingleNode(t *testing.T) {
	s := newTestState(0, 1)

	s.currHeight = 2
	service := New(s.getOptions()...)

	service.Start()
	p := s.tryRecv()
	require.NotNil(t, p)
	require.Equal(t, payload.PrepareRequestType, p.Type())
	require.EqualValues(t, 3, p.Height())
	require.EqualValues(t, 0, p.ViewNumber())
	require.NotNil(t, p.Payload())
	require.Equal(t, s.currHash, p.PrevHash())
	require.EqualValues(t, 0, p.ValidatorIndex())

	cm := s.tryRecv()
	require.NotNil(t, cm)
	require.Equal(t, payload.CommitType, cm.Type())
	require.EqualValues(t, s.currHeight+1, cm.Height())
	require.EqualValues(t, 0, cm.ViewNumber())
	require.NotNil(t, cm.Payload())
	require.EqualValues(t, 0, cm.ValidatorIndex())

	b := s.nextBlock()
	require.NotNil(t, b)
	require.Equal(t, s.currHeight+1, b.Index())
}

func TestDBFT_OnReceiveRequestSendResponse(t *testing.T) {
	s := newTestState(2, 7)
	s.verify = func(b block.Block) bool {
		for _, tx := range b.Transactions() {
			if tx.(testTx)%10 == 0 {
				return false
			}
		}

		return true
	}

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

		t.Run("receive response from primary", func(t *testing.T) {
			resp := s.getPrepareResponse(5, p.Hash())

			service.OnReceive(resp)
			require.Nil(t, s.tryRecv())
		})
	})

	t.Run("change view on invalid block", func(t *testing.T) {
		s.currHeight = 4
		service := New(s.getOptions()...)
		txs := []testTx{0}
		s.pool.Add(txs[0])

		service.Start()

		for i := range service.LastSeenMessage {
			service.LastSeenMessage[i] = &timer.HV{Height: s.currHeight + 1}
		}

		p := s.getPrepareRequest(5, txs[0].Hash())

		service.OnReceive(p)

		cv := s.tryRecv()
		require.NotNil(t, cv)
		require.Equal(t, payload.ChangeViewType, cv.Type())
		require.EqualValues(t, s.currHeight+1, cv.Height())
		require.EqualValues(t, 0, cv.ViewNumber())
		require.Equal(t, s.currHash, cv.PrevHash())
		require.EqualValues(t, s.myIndex, cv.ValidatorIndex())
		require.NotNil(t, cv.Payload())
		require.EqualValues(t, 1, cv.GetChangeView().NewViewNumber())
	})

	t.Run("change view on invalid tx", func(t *testing.T) {
		s.currHeight = 4
		service := New(s.getOptions()...)
		txs := []testTx{10}

		service.Start()

		for i := range service.LastSeenMessage {
			service.LastSeenMessage[i] = &timer.HV{Height: s.currHeight + 1}
		}

		p := s.getPrepareRequest(5, txs[0].Hash())

		service.OnReceive(p)
		require.Nil(t, s.tryRecv())

		service.OnTransaction(testTx(10))

		cv := s.tryRecv()
		require.NotNil(t, cv)
		require.Equal(t, payload.ChangeViewType, cv.Type())
		require.EqualValues(t, s.currHeight+1, cv.Height())
		require.EqualValues(t, 0, cv.ViewNumber())
		require.Equal(t, s.currHash, cv.PrevHash())
		require.EqualValues(t, s.myIndex, cv.ValidatorIndex())
		require.NotNil(t, cv.Payload())
		require.EqualValues(t, 1, cv.GetChangeView().NewViewNumber())
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

		t.Run("send recovery message on timeout", func(t *testing.T) {
			service.OnTimeout(timer.HV{Height: 1})
			require.Nil(t, s.tryRecv())

			service.OnTimeout(timer.HV{Height: s.currHeight + 1})

			r := s.tryRecv()
			require.NotNil(t, r)
			require.Equal(t, payload.RecoveryMessageType, r.Type())
		})

		t.Run("process block after enough commits", func(t *testing.T) {
			s0 := s.copyWithIndex(0)
			require.NoError(t, service.header.Sign(s0.privs[0]))
			c0 := s0.getCommit(0, service.header.Signature())
			service.OnReceive(c0)
			require.Nil(t, s.tryRecv())
			require.Nil(t, s.nextBlock())

			s1 := s.copyWithIndex(1)
			require.NoError(t, service.header.Sign(s1.privs[1]))
			c1 := s1.getCommit(1, service.header.Signature())
			service.OnReceive(c1)
			require.Nil(t, s.tryRecv())

			b := s.nextBlock()
			require.NotNil(t, b)
			require.Equal(t, s.currHeight+1, b.Index())
		})
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

func TestDBFT_OnReceiveChangeView(t *testing.T) {
	s := newTestState(2, 4)
	t.Run("change view correctly", func(t *testing.T) {
		s.currHeight = 6
		service := New(s.getOptions()...)
		service.Start()

		resp := s.getChangeView(1, 1)
		service.OnReceive(resp)
		require.Nil(t, s.tryRecv())

		resp = s.getChangeView(0, 1)
		service.OnReceive(resp)
		require.Nil(t, s.tryRecv())

		service.OnTimeout(timer.HV{Height: s.currHeight + 1})
		cv := s.tryRecv()
		require.NotNil(t, cv)
		require.Equal(t, payload.ChangeViewType, cv.Type())

		t.Run("primary sends prepare request after timeout", func(t *testing.T) {
			service.OnTimeout(timer.HV{Height: s.currHeight + 1, View: 1})
			pr := s.tryRecv()
			require.NotNil(t, pr)
			require.Equal(t, payload.PrepareRequestType, pr.Type())
		})
	})
}

func TestDBFT_Invalid(t *testing.T) {
	t.Run("without keys", func(t *testing.T) {
		require.Nil(t, New())
	})

	priv, pub := crypto.Generate(rand.Reader)
	require.NotNil(t, priv)
	require.NotNil(t, pub)

	opts := []Option{WithKeyPair(priv, pub)}
	t.Run("without CurrentHeight", func(t *testing.T) {
		require.Nil(t, New(opts...))
	})

	opts = append(opts, WithCurrentHeight(func() uint32 { return 0 }))
	t.Run("without CurrentBlockHash", func(t *testing.T) {
		require.Nil(t, New(opts...))
	})

	opts = append(opts, WithCurrentBlockHash(func() util.Uint256 { return util.Uint256{} }))
	t.Run("without GetValidators", func(t *testing.T) {
		require.Nil(t, New(opts...))
	})

	opts = append(opts, WithGetValidators(func(...block.Transaction) []crypto.PublicKey {
		return []crypto.PublicKey{pub}
	}))
	t.Run("with all defaults", func(t *testing.T) {
		d := New(opts...)
		require.NotNil(t, d)
		require.NotNil(t, d.Config.RequestTx)
		require.NotNil(t, d.Config.GetTx)
		require.NotNil(t, d.Config.GetVerified)
		require.NotNil(t, d.Config.VerifyBlock)
		require.NotNil(t, d.Config.Broadcast)
		require.NotNil(t, d.Config.ProcessBlock)
		require.NotNil(t, d.Config.GetBlock)
		require.NotNil(t, d.Config.WatchOnly)
	})
}

func (s testState) getChangeView(from uint16, view byte) Payload {
	cv := payload.NewChangeView()
	cv.SetNewViewNumber(view)

	p := s.getPayload(from)
	p.SetType(payload.ChangeViewType)
	p.SetPayload(cv)

	return p
}

func (s testState) getRecoveryRequest(from uint16) Payload {
	p := s.getPayload(from)
	p.SetType(payload.RecoveryRequestType)
	p.SetPayload(payload.NewRecoveryRequest())

	return p
}

func (s testState) getCommit(from uint16, sign []byte) Payload {
	c := payload.NewCommit()
	c.SetSignature(sign)

	p := s.getPayload(from)
	p.SetType(payload.CommitType)
	p.SetPayload(c)

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
	req.SetNextConsensus(s.nextConsensus())

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

func (s *testState) nextBlock() block.Block {
	if len(s.blocks) == 0 {
		return nil
	}

	b := s.blocks[0]
	s.blocks = s.blocks[1:]

	return b
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

func (s testState) nextConsensus(...crypto.PublicKey) util.Uint160 {
	return util.Uint160{1}
}

func (s *testState) getOptions() []Option {
	opts := []Option{
		WithCurrentHeight(func() uint32 { return s.currHeight }),
		WithCurrentBlockHash(func() util.Uint256 { return s.currHash }),
		WithGetValidators(func(...block.Transaction) []crypto.PublicKey { return s.pubs }),
		WithKeyPair(s.privs[s.myIndex], s.pubs[s.myIndex]),
		WithBroadcast(func(p Payload) { s.ch = append(s.ch, p) }),
		WithGetTx(s.pool.Get),
		WithProcessBlock(func(b block.Block) { s.blocks = append(s.blocks, b) }),
		WithGetConsensusAddress(s.nextConsensus),
		WithWatchOnly(func() bool { return false }),
		WithGetBlock(func(h util.Uint256) block.Block { return nil }),
		WithTimer(timer.New()),
		WithTxPerBlock(5),
		WithLogger(zap.NewNop()),
		WithNewBlockFromContext(NewBlockFromContext),
		WithSecondsPerBlock(time.Second * 10),
		WithRequestTx(func(...util.Uint256) {}),
		WithGetVerified(func(_ int) []block.Transaction { return []block.Transaction{} }),

		WithNewConsensusPayload(payload.NewConsensusPayload),
		WithNewPrepareRequest(payload.NewPrepareRequest),
		WithNewPrepareResponse(payload.NewPrepareResponse),
		WithNewChangeView(payload.NewChangeView),
		WithNewCommit(payload.NewCommit),
		WithNewRecoveryRequest(payload.NewRecoveryRequest),
		WithNewRecoveryMessage(payload.NewRecoveryMessage),
	}

	verify := s.verify
	if verify == nil {
		verify = func(b block.Block) bool { return true }
	}

	opts = append(opts, WithVerifyBlock(verify))

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
