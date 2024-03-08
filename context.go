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

	block  Block[H]
	header Block[H]
	// blockProcessed denotes whether Config.ProcessBlock callback was called for the current
	// height. If so, then no second call must happen. After new block is received by the user,
	// dBFT stops any new transaction or messages processing as far as timeouts handling till
	// the next call to Reset.
	blockProcessed bool

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
	lastBlockTime      time.Time // Wall clock time of when the last block was first seen (used for timer adjustments).
	lastBlockIndex     uint32
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

// CountCommitted returns number of received Commit messages not only for the current
// epoch but also for any other epoch.
func (c *Context[H]) CountCommitted() (count int) {
	for i := range c.CommitPayloads {
		if c.CommitPayloads[i] != nil {
			count++
		}
	}

	return
}

// CountFailed returns number of nodes with which no communication was performed
// for this view and that hasn't sent the Commit message at the previous views.
func (c *Context[H]) CountFailed() (count int) {
	for i, hv := range c.LastSeenMessage {
		if c.CommitPayloads[i] == nil && (hv == nil || hv.Height < c.BlockIndex || hv.View < c.ViewNumber) {
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

func (c *Context[H]) reset(view byte, ts uint64) {
	c.MyIndex = -1
	c.lastBlockTimestamp = ts

	if view == 0 {
		c.PrevHash = c.Config.CurrentBlockHash()
		c.BlockIndex = c.Config.CurrentHeight() + 1
		c.Validators = c.Config.GetValidators()

		n := len(c.Validators)
		c.LastChangeViewPayloads = make([]ConsensusPayload[H], n)

		if c.LastSeenMessage == nil {
			c.LastSeenMessage = make([]*HeightView, n)
		}
		c.blockProcessed = false
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
	c.header = nil

	n := len(c.Validators)
	c.ChangeViewPayloads = make([]ConsensusPayload[H], n)
	if view == 0 {
		c.CommitPayloads = make([]ConsensusPayload[H], n)
	}
	c.PreparationPayloads = make([]ConsensusPayload[H], n)

	c.Transactions = make(map[H]Transaction[H])
	c.TransactionHashes = nil
	c.MissingTransactions = nil
	c.PrimaryIndex = c.GetPrimaryIndex(view)
	c.ViewNumber = view

	if c.MyIndex >= 0 {
		c.LastSeenMessage[c.MyIndex] = &HeightView{c.BlockIndex, c.ViewNumber}
	}
}

// Fill initializes consensus when node is a speaker.
func (c *Context[H]) Fill() {
	b := make([]byte, 8)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}

	txx := c.Config.GetVerified()
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

		c.block.SetTransactions(txx)
	}

	return c.block
}

// MakeHeader returns half-filled block for the current epoch.
// All hashable fields will be filled.
func (c *Context[H]) MakeHeader() Block[H] {
	if c.header == nil {
		if !c.RequestSentOrReceived() {
			return nil
		}
		c.header = c.Config.NewBlockFromContext(c)
	}

	return c.header
}

// hasAllTransactions returns true iff all transactions were received
// for the proposed block.
func (c *Context[H]) hasAllTransactions() bool {
	return len(c.TransactionHashes) == len(c.Transactions)
}
