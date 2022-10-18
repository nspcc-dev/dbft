package dbft

import (
	"github.com/nspcc-dev/dbft/payload"
	"go.uber.org/zap"
)

func (d *DBFT) broadcast(msg payload.ConsensusPayload) {
	d.Logger.Debug("broadcasting message",
		zap.Stringer("type", msg.Type()),
		zap.Uint32("height", d.BlockIndex),
		zap.Uint("view", uint(d.ViewNumber)))

	msg.SetValidatorIndex(uint16(d.MyIndex))
	d.Broadcast(msg)
}

func (c *Context) makePrepareRequest() payload.ConsensusPayload {
	c.Fill()

	req := c.Config.NewPrepareRequest()
	req.SetTimestamp(c.Timestamp)
	req.SetNonce(c.Nonce)
	req.SetNextConsensus(c.NextConsensus)
	req.SetTransactionHashes(c.TransactionHashes)

	return c.Config.NewConsensusPayload(c, payload.PrepareRequestType, req)
}

func (d *DBFT) sendPrepareRequest() {
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

func (c *Context) makeChangeView(ts uint64, reason payload.ChangeViewReason) payload.ConsensusPayload {
	cv := c.Config.NewChangeView()
	cv.SetNewViewNumber(c.ViewNumber + 1)
	cv.SetTimestamp(ts)
	cv.SetReason(reason)

	msg := c.Config.NewConsensusPayload(c, payload.ChangeViewType, cv)
	c.ChangeViewPayloads[c.MyIndex] = msg

	return msg
}

func (d *DBFT) sendChangeView(reason payload.ChangeViewReason) {
	if d.Context.WatchOnly() {
		return
	}

	newView := d.ViewNumber + 1
	d.changeTimer(d.SecondsPerBlock << (newView + 1))

	nc := d.CountCommitted()
	nf := d.CountFailed()

	if reason == payload.CVTimeout && nc+nf > d.F() {
		d.Logger.Info("skip change view", zap.Int("nc", nc), zap.Int("nf", nf))
		d.sendRecoveryRequest()

		return
	}

	// Timeout while missing transactions, set the real reason.
	if !d.hasAllTransactions() && reason == payload.CVTimeout {
		reason = payload.CVTxNotFound
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

func (c *Context) makePrepareResponse() payload.ConsensusPayload {
	resp := c.Config.NewPrepareResponse()
	resp.SetPreparationHash(c.PreparationPayloads[c.PrimaryIndex].Hash())

	msg := c.Config.NewConsensusPayload(c, payload.PrepareResponseType, resp)
	c.PreparationPayloads[c.MyIndex] = msg

	return msg
}

func (d *DBFT) sendPrepareResponse() {
	msg := d.makePrepareResponse()
	d.Logger.Info("sending PrepareResponse", zap.Uint32("height", d.BlockIndex), zap.Uint("view", uint(d.ViewNumber)))
	d.StopTxFlow()
	d.broadcast(msg)
}

func (c *Context) makeCommit() payload.ConsensusPayload {
	if msg := c.CommitPayloads[c.MyIndex]; msg != nil {
		return msg
	}

	if b := c.MakeHeader(); b != nil {
		var sign []byte
		if err := b.Sign(c.Priv); err == nil {
			sign = b.Signature()
		}

		commit := c.Config.NewCommit()
		commit.SetSignature(sign)

		return c.Config.NewConsensusPayload(c, payload.CommitType, commit)
	}

	return nil
}

func (d *DBFT) sendCommit() {
	msg := d.makeCommit()
	d.CommitPayloads[d.MyIndex] = msg
	d.Logger.Info("sending Commit", zap.Uint32("height", d.BlockIndex), zap.Uint("view", uint(d.ViewNumber)))
	d.broadcast(msg)
}

func (d *DBFT) sendRecoveryRequest() {
	// If we're here, something is wrong, we either missing some messages or
	// transactions or both, so re-request missing transactions here too.
	if d.RequestSentOrReceived() && !d.hasAllTransactions() {
		d.processMissingTx()
	}
	req := d.NewRecoveryRequest()
	req.SetTimestamp(uint64(d.Timer.Now().UnixNano()))
	d.broadcast(d.Config.NewConsensusPayload(&d.Context, payload.RecoveryRequestType, req))
}

func (c *Context) makeRecoveryMessage() payload.ConsensusPayload {
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

	return c.Config.NewConsensusPayload(c, payload.RecoveryMessageType, recovery)
}

func (d *DBFT) sendRecoveryMessage() {
	d.broadcast(d.makeRecoveryMessage())
}

// defaultNewConsensusPayload is default function for creating
// consensus payload of specific type.
func defaultNewConsensusPayload(c *Context, t payload.MessageType, msg interface{}) payload.ConsensusPayload {
	cp := payload.NewConsensusPayload()
	cp.SetHeight(c.BlockIndex)
	cp.SetValidatorIndex(uint16(c.MyIndex))
	cp.SetViewNumber(c.ViewNumber)
	cp.SetType(t)
	cp.SetPayload(msg)

	return cp
}
