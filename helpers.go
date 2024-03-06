package dbft

type (
	// inbox is a structure storing messages from a single epoch.
	inbox[H Hash] struct {
		prepare map[uint16]ConsensusPayload[H]
		chViews map[uint16]ConsensusPayload[H]
		commit  map[uint16]ConsensusPayload[H]
	}

	// cache is an auxiliary structure storing messages
	// from future epochs.
	cache[H Hash] struct {
		mail map[uint32]*inbox[H]
	}
)

func newInbox[H Hash]() *inbox[H] {
	return &inbox[H]{
		prepare: make(map[uint16]ConsensusPayload[H]),
		chViews: make(map[uint16]ConsensusPayload[H]),
		commit:  make(map[uint16]ConsensusPayload[H]),
	}
}

func newCache[H Hash]() cache[H] {
	return cache[H]{
		mail: make(map[uint32]*inbox[H]),
	}
}

func (c *cache[H]) getHeight(h uint32) *inbox[H] {
	if m, ok := c.mail[h]; ok {
		delete(c.mail, h)
		return m
	}

	return nil
}

func (c *cache[H]) addMessage(m ConsensusPayload[H]) {
	msgs, ok := c.mail[m.Height()]
	if !ok {
		msgs = newInbox[H]()
		c.mail[m.Height()] = msgs
	}

	switch m.Type() {
	case PrepareRequestType, PrepareResponseType:
		msgs.prepare[m.ValidatorIndex()] = m
	case ChangeViewType:
		msgs.chViews[m.ValidatorIndex()] = m
	case CommitType:
		msgs.commit[m.ValidatorIndex()] = m
	}
}
