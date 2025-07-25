package dbft

import (
	"crypto/rand"
	"encoding/binary"
	"time"
)

// HeightView is a block height/consensus view pair.
type HeightView struct {
	Height uint32
	View   byte
}

// Context is a main dBFT structure which
// contains all information needed for performing transitions.
type Context[H Hash] struct {
	// Config is dBFT's Config instance.
	Config *Config[H]

	// Priv is node's private key.
	Priv PrivateKey
	// Pub is node's public key.
	Pub PublicKey

	preBlock  PreBlock[H]
	preHeader PreBlock[H]
	block     Block[H]
	header    Block[H]
	// blockProcessed denotes whether Config.ProcessBlock callback was called for the current
	// height. If so, then no second call must happen. After new block is received by the user,
	// dBFT stops any new transaction or messages processing as far as timeouts handling till
	// the next call to Reset.
	blockProcessed bool
	// preBlockProcessed is true when Config.ProcessPreBlock callback was
	// invoked for the current height. This happens once and dbft continues
	// to march towards proper commit after that.
	preBlockProcessed bool

	// BlockIndex is current block index.
	BlockIndex uint32
	// ViewNumber is current view number.
	ViewNumber byte
	// Validators is a current validator list.
	Validators []PublicKey
	// MyIndex is an index of the current node in the Validators array.
	// It is equal to -1 if node is not a validator or is WatchOnly.
	MyIndex int
	// PrimaryIndex is an index of the primary node in the current epoch.
	PrimaryIndex uint

	// PrevHash is a hash of the previous block.
	PrevHash H

	// Timestamp is a nanosecond-precision timestamp
	Timestamp uint64
	Nonce     uint64
	// TransactionHashes is a slice of hashes of proposed transactions in the current block.
	TransactionHashes []H
	// MissingTransactions is a slice of hashes containing missing transactions for the current block.
	MissingTransactions []H
	// Transactions is a map containing actual transactions for the current block.
	Transactions map[H]Transaction[H]

	// PreparationPayloads stores consensus Prepare* payloads for the current epoch.
	PreparationPayloads []ConsensusPayload[H]
	// PreCommitPayloads stores consensus PreCommit payloads sent through all epochs
	// as a part of anti-MEV dBFT extension. It is assumed that valid PreCommit
	// payloads can only be sent once by a single node per the whole set of consensus
	// epochs for particular block. Invalid PreCommit payloads are kicked off this
	// list immediately (if PrepareRequest was received for the current round, so
	// it's possible to verify PreCommit against PreBlock built on PrepareRequest)
	// or stored till the corresponding PrepareRequest receiving.
	PreCommitPayloads []ConsensusPayload[H]
	// CommitPayloads stores consensus Commit payloads sent throughout all epochs. It
	// is assumed that valid Commit payload can only be sent once by a single node per
	// the whole set of consensus epochs for particular block. Invalid commit payloads
	// are kicked off this list immediately (if PrepareRequest was received for the
	// current round, so it's possible to verify Commit against it) or stored till
	// the corresponding PrepareRequest receiving.
	CommitPayloads []ConsensusPayload[H]
	// ChangeViewPayloads stores consensus ChangeView payloads for the current epoch.
	ChangeViewPayloads []ConsensusPayload[H]
	// LastChangeViewPayloads stores consensus ChangeView payloads for the last epoch.
	LastChangeViewPayloads []ConsensusPayload[H]
	// LastSeenMessage array stores the height and view of the last seen message, for each validator.
	// If this node never heard a thing from validator i, LastSeenMessage[i] will be nil.
	LastSeenMessage []*HeightView

	lastBlockTimestamp uint64    // ns-precision timestamp from the last header (used for the next block timestamp calculations).
	lastBlockTime      time.Time // Wall clock time of when we started (as in PrepareRequest) creating the last block (used for timer adjustments).
	lastBlockIndex     uint32
	lastBlockView      byte
	timePerBlock       time.Duration // minimum amount of time that need to pass before the pending block will be accepted if there are some transactions in the proposal.
	maxTimePerBlock    time.Duration // maximum amount of time that allowed to pass before the pending block will be accepted even if there's no transactions in the proposal.
	txSubscriptionOn   bool

	prepareSentTime time.Time
	rttEstimates    rtt
}

// N returns total number of validators.
func (c *Context[H]) N() int { return len(c.Validators) }

// F returns number of validators which can be faulty.
func (c *Context[H]) F() int { return (len(c.Validators) - 1) / 3 }

// M returns number of validators which must function correctly.
func (c *Context[H]) M() int { return len(c.Validators) - c.F() }

// GetPrimaryIndex returns index of a primary node for the specified view.
func (c *Context[H]) GetPrimaryIndex(viewNumber byte) uint {
	p := (int(c.BlockIndex) - int(viewNumber)) % len(c.Validators)
	if p >= 0 {
		return uint(p)
	}

	return uint(p + len(c.Validators))
}

// IsPrimary returns true iff node is primary for current height and view.
func (c *Context[H]) IsPrimary() bool { return c.MyIndex == int(c.PrimaryIndex) }

// IsBackup returns true iff node is backup for current height and view.
func (c *Context[H]) IsBackup() bool {
	return c.MyIndex >= 0 && !c.IsPrimary()
}

// WatchOnly returns true iff node takes no active part in consensus.
func (c *Context[H]) WatchOnly() bool { return c.MyIndex < 0 || c.Config.WatchOnly() }

// CountCommitted returns number of received Commit (or PreCommit for anti-MEV
// extension) messages not only for the current epoch but also for any other epoch.
func (c *Context[H]) CountCommitted() (count int) {
	for i := range c.CommitPayloads {
		// Consider both Commit and PreCommit payloads since both Commit and PreCommit
		// phases are one-directional (do not impose view change).
		if c.CommitPayloads[i] != nil || c.PreCommitPayloads[i] != nil {
			count++
		}
	}

	return
}

// CountFailed returns number of nodes with which no communication was performed
// for this view and that hasn't sent the Commit message at the previous views.
func (c *Context[H]) CountFailed() (count int) {
	for i, hv := range c.LastSeenMessage {
		if (c.CommitPayloads[i] == nil && c.PreCommitPayloads[i] == nil) &&
			(hv == nil || hv.Height < c.BlockIndex || hv.View < c.ViewNumber) {
			count++
		}
	}

	return
}

// RequestSentOrReceived returns true iff PrepareRequest
// was sent or received for the current epoch.
func (c *Context[H]) RequestSentOrReceived() bool {
	return c.PreparationPayloads[c.PrimaryIndex] != nil
}

// ResponseSent returns true iff Prepare* message was sent for the current epoch.
func (c *Context[H]) ResponseSent() bool {
	return !c.WatchOnly() && c.PreparationPayloads[c.MyIndex] != nil
}

// PreCommitSent returns true iff PreCommit message was sent for the current epoch
// assuming that the node can't go further than current epoch after PreCommit was sent.
func (c *Context[H]) PreCommitSent() bool {
	return !c.WatchOnly() && c.PreCommitPayloads[c.MyIndex] != nil
}

// CommitSent returns true iff Commit message was sent for the current epoch
// assuming that the node can't go further than current epoch after commit was sent.
func (c *Context[H]) CommitSent() bool {
	return !c.WatchOnly() && c.CommitPayloads[c.MyIndex] != nil
}

// BlockSent returns true iff block was formed AND sent for the current height.
// Once block is sent, the consensus stops new transactions and messages processing
// as far as timeouts handling.
//
// Implementation note: the implementation of BlockSent differs from the C#'s one.
// In C# algorithm they use ConsensusContext's Block.Transactions null check to define
// whether block was formed, and the only place where the block can be formed is
// in the ConsensusContext's CreateBlock function right after enough Commits receiving.
// On the contrary, in our implementation we don't have access to the block's
// Transactions field as far as we can't use block null check, because there are
// several places where the call to CreateBlock happens (one of them is right after
// PrepareRequest receiving). Thus, we have a separate Context.blockProcessed field
// for the described purpose.
func (c *Context[H]) BlockSent() bool { return c.blockProcessed }

// ViewChanging returns true iff node is in a process of changing view.
func (c *Context[H]) ViewChanging() bool {
	if c.WatchOnly() {
		return false
	}

	cv := c.ChangeViewPayloads[c.MyIndex]

	return cv != nil && cv.GetChangeView().NewViewNumber() > c.ViewNumber
}

// NotAcceptingPayloadsDueToViewChanging returns true if node should not accept new payloads.
func (c *Context[H]) NotAcceptingPayloadsDueToViewChanging() bool {
	return c.ViewChanging() && !c.MoreThanFNodesCommittedOrLost()
}

// MoreThanFNodesCommittedOrLost returns true iff a number of nodes which either committed
// or are faulty is more than maximum amount of allowed faulty nodes.
// A possible attack can happen if the last node to commit is malicious and either sends change view after his
// commit to stall nodes in a higher view, or if he refuses to send recovery messages. In addition, if a node
// asking change views loses network or crashes and comes back when nodes are committed in more than one higher
// numbered view, it is possible for the node accepting recovery to commit in any of the higher views, thus
// potentially splitting nodes among views and stalling the network.
func (c *Context[H]) MoreThanFNodesCommittedOrLost() bool {
	return c.CountCommitted()+c.CountFailed() > c.F()
}

// Header returns current header from context. May be nil in case if no
// header is constructed yet. Do not change the resulting header.
func (c *Context[H]) Header() Block[H] {
	return c.header
}

// PreHeader returns current preHeader from context. May be nil in case if no
// preHeader is constructed yet. Do not change the resulting preHeader.
func (c *Context[H]) PreHeader() PreBlock[H] {
	return c.preHeader
}

// PreBlock returns current PreBlock from context. May be nil in case if no
// PreBlock is constructed yet (even if PreHeader is already constructed).
// External changes in the PreBlock will be seen by dBFT.
func (c *Context[H]) PreBlock() PreBlock[H] {
	return c.preBlock
}

func (c *Context[H]) reset(view byte, ts uint64) {
	c.MyIndex = -1
	c.prepareSentTime = time.Time{}
	c.lastBlockTimestamp = ts
	c.unsubscribeFromTransactions()

	if view == 0 {
		c.PrevHash = c.Config.CurrentBlockHash()
		c.BlockIndex = c.Config.CurrentHeight() + 1
		c.Validators = c.Config.GetValidators()
		c.timePerBlock = c.Config.TimePerBlock()
		if c.Config.MaxTimePerBlock != nil {
			c.maxTimePerBlock = c.Config.MaxTimePerBlock()
		}

		n := len(c.Validators)
		c.LastChangeViewPayloads = emptyReusableSlice(c.LastChangeViewPayloads, n)

		c.LastSeenMessage = emptyReusableSlice(c.LastSeenMessage, n)
		c.blockProcessed = false
		c.preBlockProcessed = false
	} else {
		for i := range c.Validators {
			m := c.ChangeViewPayloads[i]
			if m != nil && m.GetChangeView().NewViewNumber() >= view {
				c.LastChangeViewPayloads[i] = m
			} else {
				c.LastChangeViewPayloads[i] = nil
			}
		}
	}

	c.MyIndex, c.Priv, c.Pub = c.Config.GetKeyPair(c.Validators)

	c.block = nil
	c.preBlock = nil
	c.header = nil
	c.preHeader = nil

	n := len(c.Validators)
	c.ChangeViewPayloads = emptyReusableSlice(c.ChangeViewPayloads, n)
	if view == 0 {
		c.PreCommitPayloads = emptyReusableSlice(c.PreCommitPayloads, n)
		c.CommitPayloads = emptyReusableSlice(c.CommitPayloads, n)
	}
	c.PreparationPayloads = emptyReusableSlice(c.PreparationPayloads, n)

	if c.Transactions == nil { // Init.
		c.Transactions = make(map[H]Transaction[H])
	} else { // Regular use.
		clear(c.Transactions)
	}
	c.TransactionHashes = nil
	if c.MissingTransactions != nil {
		c.MissingTransactions = c.MissingTransactions[:0]
	}
	c.PrimaryIndex = c.GetPrimaryIndex(view)
	c.ViewNumber = view

	if c.MyIndex >= 0 {
		c.LastSeenMessage[c.MyIndex] = &HeightView{c.BlockIndex, c.ViewNumber}
	}
}

func emptyReusableSlice[E any](s []E, n int) []E {
	if len(s) == n {
		clear(s)
		return s
	}
	return make([]E, n)
}

// Fill initializes consensus when node is a speaker. It doesn't perform any
// context modifications if MaxTimePerBlock extension is enabled and there are
// no transactions in the memory pool and force is not set.
func (c *Context[H]) Fill(force bool) bool {
	txx := c.Config.GetVerified()
	if c.Config.MaxTimePerBlock != nil && !force && len(txx) == 0 {
		return false
	}

	b := make([]byte, 8)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}

	c.Nonce = binary.LittleEndian.Uint64(b)
	c.TransactionHashes = make([]H, len(txx))

	for i := range txx {
		h := txx[i].Hash()
		c.TransactionHashes[i] = h
		c.Transactions[h] = txx[i]
	}

	c.Timestamp = c.lastBlockTimestamp + c.Config.TimestampIncrement
	if now := c.getTimestamp(); now > c.Timestamp {
		c.Timestamp = now
	}
	return true
}

// getTimestamp returns nanoseconds-precision timestamp using
// current context config.
func (c *Context[H]) getTimestamp() uint64 {
	return uint64(c.Config.Timer.Now().UnixNano()) / c.Config.TimestampIncrement * c.Config.TimestampIncrement
}

// CreateBlock returns resulting block for the current epoch.
func (c *Context[H]) CreateBlock() Block[H] {
	if c.block == nil {
		if c.block = c.MakeHeader(); c.block == nil {
			return nil
		}

		txx := make([]Transaction[H], len(c.TransactionHashes))

		for i, h := range c.TransactionHashes {
			txx[i] = c.Transactions[h]
		}

		// Anti-MEV extension properly sets PreBlock transactions once during PreBlock
		// construction and then never updates these transactions in the dBFT context.
		// Thus, user must not reuse txx if anti-MEV extension is enabled. However,
		// we don't skip a call to Block.SetTransactions since it may be used as a
		// signal to the user's code to finalize the block.
		c.block.SetTransactions(txx)
	}

	return c.block
}

// CreatePreBlock returns PreBlock for the current epoch.
func (c *Context[H]) CreatePreBlock() PreBlock[H] {
	if c.preBlock == nil {
		if c.preBlock = c.MakePreHeader(); c.preBlock == nil {
			return nil
		}

		txx := make([]Transaction[H], len(c.TransactionHashes))

		for i, h := range c.TransactionHashes {
			txx[i] = c.Transactions[h]
		}

		c.preBlock.SetTransactions(txx)
	}

	return c.preBlock
}

// isAntiMEVExtensionEnabled returns whether Anti-MEV dBFT extension is enabled
// at the currently processing block height.
func (c *Context[H]) isAntiMEVExtensionEnabled() bool {
	return c.Config.AntiMEVExtensionEnablingHeight >= 0 && uint32(c.Config.AntiMEVExtensionEnablingHeight) <= c.BlockIndex
}

// MakeHeader returns half-filled block for the current epoch.
// All hashable fields will be filled.
func (c *Context[H]) MakeHeader() Block[H] {
	if c.header == nil {
		if !c.RequestSentOrReceived() {
			return nil
		}
		// For anti-MEV dBFT extension it's important to have PreBlock processed and
		// all envelopes decrypted, because a single PrepareRequest is not enough to
		// construct proper Block.
		if c.isAntiMEVExtensionEnabled() {
			if !c.preBlockProcessed {
				return nil
			}
		}
		c.header = c.Config.NewBlockFromContext(c)
	}

	return c.header
}

// MakePreHeader returns half-filled block for the current epoch.
// All hashable fields will be filled.
func (c *Context[H]) MakePreHeader() PreBlock[H] {
	if c.preHeader == nil {
		if !c.RequestSentOrReceived() {
			return nil
		}
		c.preHeader = c.Config.NewPreBlockFromContext(c)
	}

	return c.preHeader
}

// hasAllTransactions returns true iff all transactions were received
// for the proposed block.
func (c *Context[H]) hasAllTransactions() bool {
	return len(c.TransactionHashes) == len(c.Transactions)
}

func (c *Context[H]) subscribeForTransactions() {
	c.txSubscriptionOn = true
	c.Config.SubscribeForTxs()
}

func (c *Context[H]) unsubscribeFromTransactions() {
	c.txSubscriptionOn = false
}
