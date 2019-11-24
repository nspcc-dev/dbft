package dbft

import (
	"github.com/nspcc-dev/dbft/payload"
	"github.com/spaolacci/murmur3"
)

type (
	// inbox is a structure storing messages from a single epoch.
	inbox struct {
		prepare map[uint16]payload.ConsensusPayload
		chViews map[uint16]payload.ConsensusPayload
		commit  map[uint16]payload.ConsensusPayload
	}

	// cache is an auxiliary structure storing messages
	// from future epochs.
	cache struct {
		mail map[uint32]*inbox
	}
)

func fetchID(data []byte) uint64 {
	return murmur3.Sum64(data)
}

func newInbox() *inbox {
	return &inbox{
		prepare: make(map[uint16]payload.ConsensusPayload),
		chViews: make(map[uint16]payload.ConsensusPayload),
		commit:  make(map[uint16]payload.ConsensusPayload),
	}
}

func newCache() cache {
	return cache{
		mail: make(map[uint32]*inbox),
	}
}

func (c *cache) getHeight(h uint32) *inbox {
	if m, ok := c.mail[h]; ok {
		delete(c.mail, h)
		return m
	}

	return nil
}

func (c *cache) addMessage(m payload.ConsensusPayload) {
	msgs, ok := c.mail[m.Height()]
	if !ok {
		msgs = newInbox()
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
