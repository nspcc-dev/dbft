package dbft

import (
	"go.uber.org/zap"
)

func (d *DBFT[H]) checkPrepare() {
	if !d.hasAllTransactions() {
		d.Logger.Debug("check prepare: some transactions are missing", zap.Any("hashes", d.MissingTransactions))
		return
	}

	count := 0
	hasRequest := false

	for _, msg := range d.PreparationPayloads {
		if msg != nil {
			if msg.ViewNumber() == d.ViewNumber {
				count++
			}

			if msg.Type() == PrepareRequestType {
				hasRequest = true
			}
		}
	}

	d.Logger.Debug("check preparations", zap.Bool("hasReq", hasRequest),
		zap.Int("count", count),
		zap.Int("M", d.M()))

	if hasRequest && count >= d.M() {
		d.sendCommit()
		d.changeTimer(d.SecondsPerBlock)
		d.checkCommit()
	}
}

func (d *DBFT[H]) checkCommit() {
	if !d.hasAllTransactions() {
		d.Logger.Debug("check commit: some transactions are missing", zap.Any("hashes", d.MissingTransactions))
		return
	}

	// return if we received commits from other nodes
	// before receiving PrepareRequest from Speaker
	count := 0

	for _, msg := range d.CommitPayloads {
		if msg != nil && msg.ViewNumber() == d.ViewNumber {
			count++
		}
	}

	if count < d.M() {
		d.Logger.Debug("not enough to commit", zap.Int("count", count))
		return
	}

	d.lastBlockIndex = d.BlockIndex
	d.lastBlockTime = d.Timer.Now()
	d.block = d.CreateBlock()
	hash := d.block.Hash()

	d.Logger.Info("approving block",
		zap.Uint32("height", d.BlockIndex),
		zap.Stringer("hash", hash),
		zap.Int("tx_count", len(d.block.Transactions())),
		zap.Stringer("merkle", d.block.MerkleRoot()),
		zap.Stringer("prev", d.block.PrevHash()))

	d.blockProcessed = true
	d.ProcessBlock(d.block)

	// Do not initialize consensus process immediately. It's the caller's duty to
	// start the new block acceptance process and call Reset at the
	// new height.
}

func (d *DBFT[H]) checkChangeView(view byte) {
	if d.ViewNumber >= view {
		return
	}

	count := 0

	for _, msg := range d.ChangeViewPayloads {
		if msg != nil && msg.GetChangeView().NewViewNumber() >= view {
			count++
		}
	}

	if count < d.M() {
		return
	}

	if !d.Context.WatchOnly() {
		msg := d.ChangeViewPayloads[d.MyIndex]
		if msg != nil && msg.GetChangeView().NewViewNumber() < view {
			d.broadcast(d.makeChangeView(uint64(d.Timer.Now().UnixNano()), CVChangeAgreement))
		}
	}

	d.initializeConsensus(view, d.lastBlockTimestamp)
}
