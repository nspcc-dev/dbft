package dbft

type (
	// inbox is a structure storing messages from a single epoch.
	inbox[H Hash, A Address] struct {
		prepare map[uint16]ConsensusPayload[H, A]
		chViews map[uint16]ConsensusPayload[H, A]
		commit  map[uint16]ConsensusPayload[H, A]
	}

	// cache is an auxiliary structure storing messages
	// from future epochs.
	cache[H Hash, A Address] struct {
		mail map[uint32]*inbox[H, A]
	}
)

func newInbox[H Hash, A Address]() *inbox[H, A] {
	return &inbox[H, A]{
		prepare: make(map[uint16]ConsensusPayload[H, A]),
		chViews: make(map[uint16]ConsensusPayload[H, A]),
		commit:  make(map[uint16]ConsensusPayload[H, A]),
	}
}

func newCache[H Hash, A Address]() cache[H, A] {
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

func (c *cache[H, A]) addMessage(m ConsensusPayload[H, A]) {
	msgs, ok := c.mail[m.Height()]
	if !ok {
		msgs = newInbox[H, A]()
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
