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

		service.Start(0)
		require.Nil(t, s.tryRecv())
	})

	t.Run("primary send PrepareRequest on start", func(t *testing.T) {
		s.currHeight = 1
		service := New(s.getOptions()...)

		service.Start(0)
		p := s.tryRecv()
		require.NotNil(t, p)
		require.Equal(t, payload.PrepareRequestType, p.Type())
		require.EqualValues(t, 2, p.Height())
		require.EqualValues(t, 0, p.ViewNumber())
		require.NotNil(t, p.Payload())
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

	service.Start(0)
	p := s.tryRecv()
	require.NotNil(t, p)
	require.Equal(t, payload.PrepareRequestType, p.Type())
	require.EqualValues(t, 3, p.Height())
	require.EqualValues(t, 0, p.ViewNumber())
	require.NotNil(t, p.Payload())
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

		service.Start(0)
		service.OnReceive(p)

		resp := s.tryRecv()
		require.NotNil(t, resp)
		require.Equal(t, payload.PrepareResponseType, resp.Type())
		require.EqualValues(t, s.currHeight+1, resp.Height())
		require.EqualValues(t, 0, resp.ViewNumber())
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

	t.Run("change view on invalid tx", func(t *testing.T) {
		s.currHeight = 4
		service := New(s.getOptions()...)
		txs := []testTx{10}

		service.Start(0)

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
		require.EqualValues(t, s.myIndex, cv.ValidatorIndex())
		require.NotNil(t, cv.Payload())
		require.EqualValues(t, 1, cv.GetChangeView().NewViewNumber())
	})

	t.Run("receive invalid prepare request", func(t *testing.T) {
		s.currHeight = 4
		service := New(s.getOptions()...)
		txs := []testTx{1, 2}
		s.pool.Add(txs[0])

		service.Start(0)

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
			require.EqualValues(t, s.myIndex, resp.ValidatorIndex())
			require.NotNil(t, resp.Payload())
			require.Equal(t, p.Hash(), resp.GetPrepareResponse().PreparationHash())

			// do not send response twice
			service.OnTransaction(txs[1])
			require.Nil(t, s.tryRecv())
		})
	})
}

func TestDBFT_CommitOnTransaction(t *testing.T) {
	s := newTestState(0, 4)
	s.currHeight = 1

	srv := New(s.getOptions()...)
	srv.Start(0)
	require.Nil(t, s.tryRecv())

	tx := testTx(42)
	req := s.getPrepareRequest(2, tx.Hash())
	srv.OnReceive(req)
	srv.OnReceive(s.getPrepareResponse(1, req.Hash()))
	srv.OnReceive(s.getPrepareResponse(3, req.Hash()))
	require.Nil(t, srv.header) // missing transaction.

	// Test state for forming header.
	s1 := &testState{
		count:      s.count,
		pool:       newTestPool(),
		currHeight: 1,
		pubs:       s.pubs,
		privs:      s.privs,
	}
	s1.pool.Add(tx)
	srv1 := New(s1.getOptions()...)
	srv1.Start(0)
	srv1.OnReceive(req)
	srv1.OnReceive(s1.getPrepareResponse(1, req.Hash()))
	srv1.OnReceive(s1.getPrepareResponse(3, req.Hash()))
	require.NotNil(t, srv1.header)

	for _, i := range []uint16{1, 2, 3} {
		require.NoError(t, srv1.header.Sign(s1.privs[i]))
		c := s1.getCommit(i, srv1.header.Signature())
		srv.OnReceive(c)
	}

	require.Nil(t, s.nextBlock())
	srv.OnTransaction(tx)
	require.NotNil(t, s.nextBlock())
}

func TestDBFT_OnReceiveCommit(t *testing.T) {
	s := newTestState(2, 4)
	t.Run("send commit after enough responses", func(t *testing.T) {
		s.currHeight = 1
		service := New(s.getOptions()...)
		service.Start(0)

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
		service.Start(0)

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
		srv2.Start(0)
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
		service.Start(0)

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

// TestDBFT_FourGoodNodesDeadlock checks that the following liveness lock is not really
// a liveness lock and there's a way to accept block in this situation.
// 0 :> [type |-> "cv", view |-> 1]           <--- this is the primary at view 1
// 1 :> [type |-> "cv", view |-> 1]           <--- this is the primary at view 0
// 2 :> [type |-> "commitSent", view |-> 0]
// 3 :> [type |-> "commitSent", view |-> 1]
//
// Test structure note: the test is organized to reproduce the liveness lock scenario
// described in https://github.com/neo-project/neo-modules/issues/792#issue-1609058923
// at the section named "1. Liveness lock with four non-faulty nodes". However, some
// steps are rearranged so that it's possible to reach the target network state described
// above. It is done because dbft implementation contains additional constraints comparing
// to the TLA+ model. See the steps
func TestDBFT_FourGoodNodesDeadlock(t *testing.T) {
	r0 := newTestState(0, 4)
	r0.currHeight = 4
	s0 := New(r0.getOptions()...)
	s0.Start(0)

	r1 := r0.copyWithIndex(1)
	s1 := New(r1.getOptions()...)
	s1.Start(0)

	r2 := r0.copyWithIndex(2)
	s2 := New(r2.getOptions()...)
	s2.Start(0)

	r3 := r0.copyWithIndex(3)
	s3 := New(r3.getOptions()...)
	s3.Start(0)

	// Step 1. The primary (at view 0) replica 1 sends the PrepareRequest message.
	reqV0 := r1.tryRecv()
	require.NotNil(t, reqV0)
	require.Equal(t, payload.PrepareRequestType, reqV0.Type())

	// Step 2 will be performed later, see the comment to Step 2.

	// Step 3. The backup (at view 0) replica 0 receives the PrepareRequest of
	// view 0 and broadcasts its PrepareResponse.
	s0.OnReceive(reqV0)
	resp0V0 := r0.tryRecv()
	require.NotNil(t, resp0V0)
	require.Equal(t, payload.PrepareResponseType, resp0V0.Type())

	// Step 4 will be performed later, see the comment to Step 4.

	// Step 5. The backup (at view 0) replica 2 receives the PrepareRequest of
	// view 0 and broadcasts its PrepareResponse.
	s2.OnReceive(reqV0)
	resp2V0 := r2.tryRecv()
	require.NotNil(t, resp2V0)
	require.Equal(t, payload.PrepareResponseType, resp2V0.Type())

	// Step 6. The backup (at view 0) replica 2 collects M prepare messages (from
	// itself and replicas 0, 1) and broadcasts the Commit message for view 0.
	s2.OnReceive(resp0V0)
	cm2V0 := r2.tryRecv()
	require.NotNil(t, cm2V0)
	require.Equal(t, payload.CommitType, cm2V0.Type())

	// Step 7. The backup (at view 0) replica 3 decides to change its view
	// (possible on timeout) and sends the ChangeView message.
	s3.OnReceive(resp0V0)
	s3.OnReceive(resp2V0)
	s3.OnTimeout(timer.HV{Height: r3.currHeight + 1, View: 0})
	cv3V0 := r3.tryRecv()
	require.NotNil(t, cv3V0)
	require.Equal(t, payload.ChangeViewType, cv3V0.Type())

	// Step 2. The primary (at view 0) replica 1 decides to change its view
	// (possible on timeout after receiving at least M non-commit messages from the
	// current view) and sends the ChangeView message.
	s1.OnReceive(resp0V0)
	s1.OnReceive(cv3V0)
	s1.OnTimeout(timer.HV{Height: r1.currHeight + 1, View: 0})
	cv1V0 := r1.tryRecv()
	require.NotNil(t, cv1V0)
	require.Equal(t, payload.ChangeViewType, cv1V0.Type())

	// Step 4. The backup (at view 0) replica 0 decides to change its view
	// (possible on timeout after receiving at least M non-commit messages from the
	// current view) and sends the ChangeView message.
	s0.OnReceive(cv3V0)
	s0.OnTimeout(timer.HV{Height: r0.currHeight + 1, View: 0})
	cv0V0 := r0.tryRecv()
	require.NotNil(t, cv0V0)
	require.Equal(t, payload.ChangeViewType, cv0V0.Type())

	// Step 8. The primary (at view 0) replica 1 collects M ChangeView messages
	// (from itself and replicas 1, 3) and changes its view to 1.
	s1.OnReceive(cv0V0)
	require.Equal(t, uint8(1), s1.ViewNumber)

	// Step 9. The backup (at view 0) replica 0 collects M ChangeView messages
	// (from itself and replicas 0, 3) and changes its view to 1.
	s0.OnReceive(cv1V0)
	require.Equal(t, uint8(1), s0.ViewNumber)

	// Step 10. The primary (at view 1) replica 0 sends the PrepareRequest message.
	s0.OnTimeout(timer.HV{Height: r0.currHeight + 1, View: 1})
	reqV1 := r0.tryRecv()
	require.NotNil(t, reqV1)
	require.Equal(t, payload.PrepareRequestType, reqV1.Type())

	// Step 11. The backup (at view 1) replica 1 receives the PrepareRequest of
	// view 1 and sends the PrepareResponse.
	s1.OnReceive(reqV1)
	resp1V1 := r1.tryRecv()
	require.NotNil(t, resp1V1)
	require.Equal(t, payload.PrepareResponseType, resp1V1.Type())

	// Steps 12, 13 will be performed later, see the comments to Step 12, 13.

	// Step 14. The backup (at view 0) replica 3 collects M ChangeView messages
	// (from itself and replicas 0, 1) and changes its view to 1.
	s3.OnReceive(cv0V0)
	s3.OnReceive(cv1V0)
	require.Equal(t, uint8(1), s3.ViewNumber)

	// Intermediate step A. It is added to make step 14 possible. The backup (at
	// view 1) replica 3 doesn't receive anything for a long time and sends
	// RecoveryRequest.
	s3.OnTimeout(timer.HV{Height: r3.currHeight + 1, View: 1})
	rcvr3V1 := r3.tryRecv()
	require.NotNil(t, rcvr3V1)
	require.Equal(t, payload.RecoveryRequestType, rcvr3V1.Type())

	// Intermediate step B. The backup (at view 1) replica 1 should receive any
	// message from replica 3 to be able to change view. However, it couldn't be
	// PrepareResponse because replica 1 will immediately commit then. Thus, the
	// only thing that remains is to receive RecoveryRequest from replica 3.
	// Replica 1 then should answer with Recovery message.
	s1.OnReceive(rcvr3V1)
	rcvrResp1V1 := r1.tryRecv()
	require.NotNil(t, rcvrResp1V1)
	require.Equal(t, payload.RecoveryMessageType, rcvrResp1V1.Type())

	// Intermediate step C. The primary (at view 1) replica 0 should receive
	// RecoveryRequest from replica 3. The purpose of this step is the same as
	// in Intermediate step B.
	s0.OnReceive(rcvr3V1)
	rcvrResp0V1 := r0.tryRecv()
	require.NotNil(t, rcvrResp0V1)
	require.Equal(t, payload.RecoveryMessageType, rcvrResp0V1.Type())

	// Step 12. According to the neo-project/neo#792, at this step the backup (at view 1)
	// replica 1 decides to change its view (possible on timeout) and sends the
	// ChangeView message. However, the recovery message will be broadcast instead
	// of CV, because there's additional condition: too much (>F) "lost" or committed
	// nodes are present, see https://github.com/roman-khimov/dbft/blob/b769eb3e0f070d6eabb9443a5931eb4a2e46c538/send.go#L68.
	// Replica 1 aware of replica 0 that has sent the PrepareRequest for view 1.
	// It can also be aware of replica 2 that has committed at view 0, but it won't
	// change the situation. The final way to allow CV is to receive something
	// except from PrepareResponse from replica 3 to remove replica 3 from the list
	// of "lost" nodes. That's why we'he added Intermediate steps A and B.
	//
	// After that replica 1 is allowed to send the CV message.
	s1.OnTimeout(timer.HV{Height: r1.currHeight + 1, View: 1})
	cv1V1 := r1.tryRecv()
	require.NotNil(t, cv1V1)
	require.Equal(t, payload.ChangeViewType, cv1V1.Type())

	// Step 13. The primary (at view 1) replica 0 decides to change its view
	// (possible on timeout) and sends the ChangeView message.
	s0.OnReceive(resp1V1)
	s0.OnTimeout(timer.HV{Height: r0.currHeight + 1, View: 1})
	cv0V1 := r0.tryRecv()
	require.NotNil(t, cv0V1)
	require.Equal(t, payload.ChangeViewType, cv0V1.Type())

	// Step 15. The backup (at view 1) replica 3 receives PrepareRequest of view
	// 1 and broadcasts its PrepareResponse.
	s3.OnReceive(reqV1)
	resp3V1 := r3.tryRecv()
	require.NotNil(t, resp3V1)
	require.Equal(t, payload.PrepareResponseType, resp3V1.Type())

	// Step 16. The backup (at view 1) replica 3 collects M prepare messages and
	// broadcasts the Commit message for view 1.
	s3.OnReceive(resp1V1)
	cm3V1 := r3.tryRecv()
	require.NotNil(t, cm3V1)
	require.Equal(t, payload.CommitType, cm3V1.Type())

	// Intermediate step D. It is needed to enable step 17 and to check that
	// MoreThanFNodesCommittedOrLost works properly and counts Commit messages from
	// any view.
	s0.OnReceive(cm2V0)
	s0.OnReceive(cm3V1)

	// Step 17. The issue says that "The rest of undelivered messages eventually
	// reaches their receivers, but it doesn't change the node's states.", but it's
	// not true, the aim of the test is to show that replicas 0 and 1 still can
	// commit at view 1 even after CV sent.
	s0.OnReceive(resp3V1)
	cm0V1 := r0.tryRecv()
	require.NotNil(t, cm0V1)
	require.Equal(t, payload.CommitType, cm0V1.Type())

	s1.OnReceive(cm0V1)
	s1.OnReceive(resp3V1)
	cm1V1 := r1.tryRecv()
	require.NotNil(t, cm1V1)
	require.Equal(t, payload.CommitType, cm1V1.Type())

	// Finally, send missing Commit message to replicas 0 and 1, they should accept
	// the block.
	require.Nil(t, r0.nextBlock())
	s0.OnReceive(cm1V1)
	require.NotNil(t, r0.nextBlock())

	require.Nil(t, r1.nextBlock())
	s1.OnReceive(cm3V1)
	require.NotNil(t, r1.nextBlock())

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
		WithLogger(zap.NewNop()),
		WithNewBlockFromContext(NewBlockFromContext),
		WithSecondsPerBlock(time.Second * 10),
		WithRequestTx(func(...util.Uint256) {}),
		WithGetVerified(func() []block.Transaction { return []block.Transaction{} }),

		WithNewConsensusPayload(defaultNewConsensusPayload),
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
