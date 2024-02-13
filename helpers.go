package dbft

import (
	"github.com/nspcc-dev/dbft/crypto"
	"github.com/nspcc-dev/dbft/payload"
)

type (
	// inbox is a structure storing messages from a single epoch.
	inbox[H crypto.Hash, A crypto.Address] struct {
		prepare map[uint16]payload.ConsensusPayload[H, A]
		chViews map[uint16]payload.ConsensusPayload[H, A]
		commit  map[uint16]payload.ConsensusPayload[H, A]
	}

	// cache is an auxiliary structure storing messages
	// from future epochs.
	cache[H crypto.Hash, A crypto.Address] struct {
		mail map[uint32]*inbox[H, A]
	}
)

func newInbox[H crypto.Hash, A crypto.Address]() *inbox[H, A] {
	return &inbox[H, A]{
		prepare: make(map[uint16]payload.ConsensusPayload[H, A]),
		chViews: make(map[uint16]payload.ConsensusPayload[H, A]),
		commit:  make(map[uint16]payload.ConsensusPayload[H, A]),
	}
}

func newCache[H crypto.Hash, A crypto.Address]() cache[H, A] {
	return cache[H, A]{
		mail: make(map[uint32]*inbox[H, A]),
	}
}

func (c *cache[H, A]) getHeight(h uint32) *inbox[H, A] {
	if m, ok := c.mail[h]; ok {
		delete(c.mail, h)
		return m
	}

	return nil
}

func (c *cache[H, A]) addMessage(m payload.ConsensusPayload[H, A]) {
	msgs, ok := c.mail[m.Height()]
	if !ok {
		msgs = newInbox[H, A]()
		c.mail[m.Height()] = msgs
	}

	switch m.Type() {
	case payload.PrepareRequestType, payload.PrepareResponseType:
		msgs.prepare[m.ValidatorIndex()] = m
	case payload.ChangeViewType:
		msgs.chViews[m.ValidatorIndex()] = m
	case payload.CommitType:
		msgs.commit[m.ValidatorIndex()] = m
	}
}
