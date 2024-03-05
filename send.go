package dbft

import (
	"go.uber.org/zap"
)

func (d *DBFT[H, A]) broadcast(msg ConsensusPayload[H, A]) {
	d.Logger.Debug("broadcasting message",
		zap.Stringer("type", msg.Type()),
		zap.Uint32("height", d.BlockIndex),
		zap.Uint("view", uint(d.ViewNumber)))

	msg.SetValidatorIndex(uint16(d.MyIndex))
	d.Broadcast(msg)
}

func (c *Context[H, A]) makePrepareRequest() ConsensusPayload[H, A] {
	c.Fill()

	req := c.Config.NewPrepareRequest()
	req.SetTimestamp(c.Timestamp)
	req.SetNonce(c.Nonce)
	req.SetNextConsensus(c.NextConsensus)
	req.SetTransactionHashes(c.TransactionHashes)

	return c.Config.NewConsensusPayload(c, PrepareRequestType, req)
}

func (d *DBFT[H, A]) sendPrepareRequest() {
	msg := d.makePrepareRequest()
	d.PreparationPayloads[d.MyIndex] = msg
	d.broadcast(msg)

	delay := d.SecondsPerBlock << (d.ViewNumber + 1)
	if d.ViewNumber == 0 {
		delay -= d.SecondsPerBlock
	}

	d.Logger.Info("sending PrepareRequest", zap.Uint32("height", d.BlockIndex), zap.Uint("view", uint(d.ViewNumber)))
	d.changeTimer(delay)
	d.checkPrepare()
}

func (c *Context[H, A]) makeChangeView(ts uint64, reason ChangeViewReason) ConsensusPayload[H, A] {
	cv := c.Config.NewChangeView(c.ViewNumber+1, reason, ts)

	msg := c.Config.NewConsensusPayload(c, ChangeViewType, cv)
	c.ChangeViewPayloads[c.MyIndex] = msg

	return msg
}

func (d *DBFT[H, A]) sendChangeView(reason ChangeViewReason) {
	if d.Context.WatchOnly() {
		return
	}

	newView := d.ViewNumber + 1
	d.changeTimer(d.SecondsPerBlock << (newView + 1))

	nc := d.CountCommitted()
	nf := d.CountFailed()

	if reason == CVTimeout && nc+nf > d.F() {
		d.Logger.Info("skip change view", zap.Int("nc", nc), zap.Int("nf", nf))
		d.sendRecoveryRequest()

		return
	}

	// Timeout while missing transactions, set the real reason.
	if !d.hasAllTransactions() && reason == CVTimeout {
		reason = CVTxNotFound
	}

	d.Logger.Info("request change view",
		zap.Int("view", int(d.ViewNumber)),
		zap.Uint32("height", d.BlockIndex),
		zap.Stringer("reason", reason),
		zap.Int("new_view", int(newView)),
		zap.Int("nc", nc),
		zap.Int("nf", nf))

	msg := d.makeChangeView(uint64(d.Timer.Now().UnixNano()), reason)
	d.StopTxFlow()
	d.broadcast(msg)
	d.checkChangeView(newView)
}

func (c *Context[H, A]) makePrepareResponse() ConsensusPayload[H, A] {
	resp := c.Config.NewPrepareResponse()
	resp.SetPreparationHash(c.PreparationPayloads[c.PrimaryIndex].Hash())

	msg := c.Config.NewConsensusPayload(c, PrepareResponseType, resp)
	c.PreparationPayloads[c.MyIndex] = msg

	return msg
}

func (d *DBFT[H, A]) sendPrepareResponse() {
	msg := d.makePrepareResponse()
	d.Logger.Info("sending PrepareResponse", zap.Uint32("height", d.BlockIndex), zap.Uint("view", uint(d.ViewNumber)))
	d.StopTxFlow()
	d.broadcast(msg)
}

func (c *Context[H, A]) makeCommit() ConsensusPayload[H, A] {
	if msg := c.CommitPayloads[c.MyIndex]; msg != nil {
		return msg
	}

	if b := c.MakeHeader(); b != nil {
		var sign []byte
		if err := b.Sign(c.Priv); err == nil {
			sign = b.Signature()
		}

		commit := c.Config.NewCommit(sign)

		return c.Config.NewConsensusPayload(c, CommitType, commit)
	}

	return nil
}

func (d *DBFT[H, A]) sendCommit() {
	msg := d.makeCommit()
	d.CommitPayloads[d.MyIndex] = msg
	d.Logger.Info("sending Commit", zap.Uint32("height", d.BlockIndex), zap.Uint("view", uint(d.ViewNumber)))
	d.broadcast(msg)
}

func (d *DBFT[H, A]) sendRecoveryRequest() {
	// If we're here, something is wrong, we either missing some messages or
	// transactions or both, so re-request missing transactions here too.
	if d.RequestSentOrReceived() && !d.hasAllTransactions() {
		d.processMissingTx()
	}
	req := d.NewRecoveryRequest()
	req.SetTimestamp(uint64(d.Timer.Now().UnixNano()))
	d.broadcast(d.Config.NewConsensusPayload(&d.Context, RecoveryRequestType, req))
}

func (c *Context[H, A]) makeRecoveryMessage() ConsensusPayload[H, A] {
	recovery := c.Config.NewRecoveryMessage()

	for _, p := range c.PreparationPayloads {
		if p != nil {
			recovery.AddPayload(p)
		}
	}

	cv := c.LastChangeViewPayloads
	// if byte(msg.ViewNumber) == c.ViewNumber {
	// 	cv = c.changeViewPayloads
	// }
	for _, p := range cv {
		if p != nil {
			recovery.AddPayload(p)
		}
	}

	if c.CommitSent() {
		for _, p := range c.CommitPayloads {
			if p != nil {
				recovery.AddPayload(p)
			}
		}
	}

	return c.Config.NewConsensusPayload(c, RecoveryMessageType, recovery)
}

func (d *DBFT[H, A]) sendRecoveryMessage() {
	d.broadcast(d.makeRecoveryMessage())
}
