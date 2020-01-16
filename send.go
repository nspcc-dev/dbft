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

	return c.withPayload(payload.PrepareRequestType, req)
}

func (d *DBFT) sendPrepareRequest() {
	msg := d.makePrepareRequest()
	d.PreparationPayloads[d.MyIndex] = msg
	d.broadcast(msg)

	delay := d.SecondsPerBlock << (d.ViewNumber + 1)
	if d.ViewNumber == 0 {
		delay -= d.SecondsPerBlock
	}

	d.changeTimer(delay)
	d.checkPrepare()
}

func (c *Context) makeChangeView(ts uint32) payload.ConsensusPayload {
	cv := c.Config.NewChangeView()
	cv.SetNewViewNumber(c.ViewNumber + 1)
	cv.SetTimestamp(ts)

	msg := c.withPayload(payload.ChangeViewType, cv)
	c.ChangeViewPayloads[c.MyIndex] = msg

	return msg
}

func (d *DBFT) sendChangeView() {
	if d.Context.WatchOnly() {
		return
	}

	newView := d.ViewNumber + 1
	d.changeTimer(d.SecondsPerBlock << (newView + 1))

	nc := d.CountCommitted()
	nf := d.CountFailed()

	if nc+nf > d.F() {
		d.Logger.Debug("skip change view", zap.Int("nc", nc), zap.Int("nf", nf))
		d.sendRecoveryRequest()

		return
	}

	d.Logger.Debug("request change view",
		zap.Int("view", int(d.ViewNumber)),
		zap.Uint32("height", d.BlockIndex),
		zap.Int("new_view", int(newView)),
		zap.Int("nc", nc),
		zap.Int("nf", nf))

	msg := d.makeChangeView(uint32(d.Timer.Now().Unix()))
	d.broadcast(msg)
	d.checkChangeView(newView)
}

func (c *Context) makePrepareResponse() payload.ConsensusPayload {
	resp := c.Config.NewPrepareResponse()
	resp.SetPreparationHash(c.PreparationPayloads[c.PrimaryIndex].Hash())

	msg := c.withPayload(payload.PrepareResponseType, resp)
	c.PreparationPayloads[c.MyIndex] = msg

	return msg
}

func (d *DBFT) sendPrepareResponse() {
	msg := d.makePrepareResponse()
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

		return c.withPayload(payload.CommitType, commit)
	}

	return nil
}

func (d *DBFT) sendCommit() {
	msg := d.makeCommit()
	d.CommitPayloads[d.MyIndex] = msg
	d.broadcast(msg)
}

func (d *DBFT) sendRecoveryRequest() {
	req := d.NewRecoveryRequest()
	req.SetTimestamp(uint32(d.Timer.Now().Unix()))
	d.broadcast(d.withPayload(payload.RecoveryRequestType, req))
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

	return c.withPayload(payload.RecoveryMessageType, recovery)
}

func (d *DBFT) sendRecoveryMessage() {
	d.broadcast(d.makeRecoveryMessage())
}

func (c *Context) withPayload(t payload.MessageType, payload interface{}) payload.ConsensusPayload {
	cp := c.Config.NewConsensusPayload()
	cp.SetPrevHash(c.PrevHash)
	cp.SetHeight(c.BlockIndex)
	cp.SetValidatorIndex(uint16(c.MyIndex))
	cp.SetViewNumber(c.ViewNumber)
	cp.SetType(t)
	cp.SetPayload(payload)

	return cp
}
