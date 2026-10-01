package dbft

type (
	// inbox is a structure storing messages from a single epoch.
	inbox[H Hash, Tx Transaction[H]] struct {
		prepare   map[uint16]ConsensusPayload[H, Tx]
		chViews   map[uint16]ConsensusPayload[H, Tx]
		preCommit map[uint16]ConsensusPayload[H, Tx]
		commit    map[uint16]ConsensusPayload[H, Tx]
	}

	// cache is an auxiliary structure storing messages
	// from future epochs.
	cache[H Hash, Tx Transaction[H]] struct {
		mail map[uint32]*inbox[H, Tx]
	}
)

func newInbox[H Hash, Tx Transaction[H]]() *inbox[H, Tx] {
	return &inbox[H, Tx]{
		prepare:   make(map[uint16]ConsensusPayload[H, Tx]),
		chViews:   make(map[uint16]ConsensusPayload[H, Tx]),
		preCommit: make(map[uint16]ConsensusPayload[H, Tx]),
		commit:    make(map[uint16]ConsensusPayload[H, Tx]),
	}
}

func newCache[H Hash, Tx Transaction[H]]() cache[H, Tx] {
	return cache[H, Tx]{
		mail: make(map[uint32]*inbox[H, Tx]),
	}
}

func (c *cache[H, Tx]) getHeight(h uint32) *inbox[H, Tx] {
	if m, ok := c.mail[h]; ok {
		delete(c.mail, h)
		return m
	}

	return nil
}

func (c *cache[H, Tx]) addMessage(m ConsensusPayload[H, Tx]) {
	msgs, ok := c.mail[m.Height()]
	if !ok {
		msgs = newInbox[H, Tx]()
		c.mail[m.Height()] = msgs
	}

	switch m.Type() {
	case PrepareRequestType, PrepareResponseType:
		msgs.prepare[m.ValidatorIndex()] = m
	case ChangeViewType:
		msgs.chViews[m.ValidatorIndex()] = m
	case PreCommitType:
		msgs.preCommit[m.ValidatorIndex()] = m
	case CommitType:
		msgs.commit[m.ValidatorIndex()] = m
	default:
		// Others are recoveries and we don't currently use them.
		// Theoretically messages could be extracted.
	}
}
