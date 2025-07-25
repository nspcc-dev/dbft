package dbft

import (
	"fmt"
	"slices"
	"sync"
	"time"

	"go.uber.org/zap"
)

type (
	// DBFT is a dBFT implementation, it includes [Context] (main state)
	// and [Config] (service configuration). Data exposed from these fields
	// is supposed to be read-only, state is changed via methods of this
	// structure.
	DBFT[H Hash] struct {
		Context[H]
		Config[H]

		*sync.Mutex
		cache      cache[H]
		recovering bool
	}
)

// New returns new DBFT instance with specified H and A generic parameters
// using provided options or nil and error if some of the options are missing or invalid.
// H and A generic parameters are used as hash and address representation for
// dBFT consensus messages, blocks and transactions.
func New[H Hash](options ...func(config *Config[H])) (*DBFT[H], error) {
	cfg := defaultConfig[H]()

	for _, option := range options {
		option(cfg)
	}

	if err := checkConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	d := &DBFT[H]{
		Mutex:  new(sync.Mutex),
		Config: *cfg,
		Context: Context[H]{
			Config: cfg,
		},
	}

	return d, nil
}

func (d *DBFT[H]) addTransaction(tx Transaction[H]) {
	d.Transactions[tx.Hash()] = tx
	if d.hasAllTransactions() {
		if d.IsPrimary() || d.Context.WatchOnly() {
			return
		}

		if !d.createAndCheckBlock() {
			return
		}

		d.verifyPreCommitPayloadsAgainstPreBlock()

		d.extendTimer(2)
		d.sendPrepareResponse()
		d.checkPrepare()
	}
}

// Start initializes dBFT instance and starts the protocol if node is primary.
// It accepts the timestamp of the previous block. It should be called once
// per DBFT lifetime.
func (d *DBFT[H]) Start(ts uint64) {
	d.cache = newCache[H]()
	d.initializeConsensus(0, ts)
	if d.IsPrimary() {
		d.sendPrepareRequest(true)
	}
}

// Reset reinitializes dBFT instance with the given timestamp of the previous
// block. It's used if the current consensus state is outdated which happens
// after new block is processed by ledger (the block can come from dBFT or be
// received by other means). The height is to be derived from the configured
// CurrentHeight callback and view will be set to 0.
func (d *DBFT[H]) Reset(ts uint64) {
	d.initializeConsensus(0, ts)
}

func (d *DBFT[H]) initializeConsensus(view byte, ts uint64) {
	d.reset(view, ts)

	var role string

	switch {
	case d.IsPrimary():
		role = "Primary"
	case d.Context.WatchOnly():
		role = "WatchOnly"
	default:
		role = "Backup"
	}

	var logMsg = "initializing dbft"
	if view > 0 {
		logMsg = "changing dbft view"
	}

	d.StopTxFlow()
	d.Logger.Info(logMsg,
		zap.Uint32("height", d.BlockIndex),
		zap.Uint("view", uint(view)),
		zap.Int("index", d.MyIndex),
		zap.String("role", role))

	// Process cached messages if any.
	if msgs := d.cache.getHeight(d.BlockIndex); msgs != nil {
		for _, m := range msgs.prepare {
			d.OnReceive(m)
		}

		for _, m := range msgs.chViews {
			d.OnReceive(m)
		}

		for _, m := range msgs.preCommit {
			d.OnReceive(m)
		}

		for _, m := range msgs.commit {
			d.OnReceive(m)
		}
	}

	if d.Context.WatchOnly() {
		return
	}

	var timeout time.Duration
	if d.IsPrimary() && !d.recovering {
		// Initializing to view 0 means we have just persisted previous block or are starting consensus first time.
		// In both cases we should wait full timeout value.
		// Having non-zero view means we have to start immediately.
		if view == 0 {
			timeout = d.timePerBlock
		}
	} else {
		timeout = d.timePerBlock << (d.ViewNumber + 1)
	}
	if d.lastBlockIndex+1 == d.BlockIndex {
		var ts = d.Timer.Now()
		var diff = ts.Sub(d.lastBlockTime)
		timeout -= diff
		timeout -= d.rttEstimates.avg / 2
		timeout = max(0, timeout)
	}
	d.changeTimer(timeout)
}

// OnTransaction notifies service about receiving new transaction from the
// proposed list of transactions.
func (d *DBFT[H]) OnTransaction(tx Transaction[H]) {
	// d.Logger.Debug("OnTransaction",
	// 	zap.Bool("backup", d.IsBackup()),
	// 	zap.Bool("not_accepting", d.NotAcceptingPayloadsDueToViewChanging()),
	// 	zap.Bool("request_ok", d.RequestSentOrReceived()),
	// 	zap.Bool("response_sent", d.ResponseSent()),
	// 	zap.Bool("block_sent", d.BlockSent()))
	if !d.IsBackup() || d.NotAcceptingPayloadsDueToViewChanging() ||
		!d.RequestSentOrReceived() || d.ResponseSent() || d.PreCommitSent() ||
		d.CommitSent() || d.BlockSent() || len(d.MissingTransactions) == 0 {
		return
	}

	i := slices.Index(d.MissingTransactions, tx.Hash())
	if i < 0 {
		return
	}
	d.addTransaction(tx)
	// `addTransaction` checks for responses and commits. If this was the last transaction
	// Context could be initialized on a new height, clearing this field.
	if len(d.MissingTransactions) == 0 {
		return
	}
	d.MissingTransactions = slices.Delete(d.MissingTransactions, i, i+1)
}

// OnTimeout advances state machine as if timeout was fired.
func (d *DBFT[H]) OnTimeout(height uint32, view byte) {
	d.onTimeout(height, view, false)
}

// OnNewTransaction advances state machine if transactions subscription is active
// and there's a new transaction added to the node pool.
func (d *DBFT[H]) OnNewTransaction() {
	if !d.txSubscriptionOn {
		return
	}
	d.onTimeout(d.Timer.Height(), d.Timer.View(), true)
}

func (d *DBFT[H]) onTimeout(height uint32, view byte, force bool) {
	if d.Context.WatchOnly() || d.BlockSent() {
		return
	}

	if height != d.BlockIndex || view != d.ViewNumber {
		d.Logger.Debug("timeout: ignore old timer",
			zap.Uint32("height", height),
			zap.Uint("view", uint(view)))

		return
	}

	d.Logger.Debug("timeout",
		zap.Uint32("height", height),
		zap.Uint("view", uint(view)))

	if d.IsPrimary() && !d.RequestSentOrReceived() {
		d.sendPrepareRequest(d.ViewNumber != 0 || d.txSubscriptionOn || force)
	} else if (d.IsPrimary() && d.RequestSentOrReceived()) || d.IsBackup() {
		if d.CommitSent() || d.PreCommitSent() {
			d.Logger.Debug("send recovery to resend commit")
			d.sendRecoveryMessage()
			d.changeTimer(d.timePerBlock << 1)
		} else {
			if d.ViewNumber == 0 && d.MaxTimePerBlock != nil && d.IsBackup() {
				if force {
					delay := d.timePerBlock << 1
					d.changeTimer(delay)
					d.unsubscribeFromTransactions()
					return
				}
				if !d.txSubscriptionOn && len(d.Config.GetVerified()) == 0 {
					d.subscribeForTransactions()
					delay := d.maxTimePerBlock<<1 - d.timePerBlock<<1
					d.changeTimer(delay)
					return
				}
			}
			d.sendChangeView(CVTimeout)
		}
	}
}

// OnReceive advances state machine in accordance with msg.
func (d *DBFT[H]) OnReceive(msg ConsensusPayload[H]) {
	if int(msg.ValidatorIndex()) >= len(d.Validators) {
		d.Logger.Error("too big validator index", zap.Uint16("from", msg.ValidatorIndex()))
		return
	}

	if msg.Payload() == nil {
		d.Logger.DPanic("invalid message")
		return
	}

	d.Logger.Debug("received message",
		zap.Stringer("type", msg.Type()),
		zap.Uint16("from", msg.ValidatorIndex()),
		zap.Uint32("height", msg.Height()),
		zap.Uint("view", uint(msg.ViewNumber())),
		zap.Uint32("my_height", d.BlockIndex),
		zap.Uint("my_view", uint(d.ViewNumber)))

	if msg.Height() < d.BlockIndex {
		d.Logger.Debug("ignoring old height", zap.Uint32("height", msg.Height()))
		return
	} else if msg.Height() > d.BlockIndex ||
		(msg.ViewNumber() > d.ViewNumber &&
			msg.Type() != ChangeViewType &&
			msg.Type() != RecoveryMessageType) {
		d.Logger.Debug("caching message from future",
			zap.Uint32("height", msg.Height()),
			zap.Uint("view", uint(msg.ViewNumber())),
			zap.Any("cache", d.cache.mail[msg.Height()]))
		d.cache.addMessage(msg)
		return
	}

	hv := d.LastSeenMessage[msg.ValidatorIndex()]
	if hv == nil || hv.Height < msg.Height() || hv.View < msg.ViewNumber() {
		d.LastSeenMessage[msg.ValidatorIndex()] = &HeightView{msg.Height(), msg.ViewNumber()}
	}

	if d.BlockSent() && msg.Type() != RecoveryRequestType {
		// We've already collected the block, only recovery request must be handled.
		return
	}

	switch msg.Type() {
	case ChangeViewType:
		d.onChangeView(msg)
	case PrepareRequestType:
		d.onPrepareRequest(msg)
	case PrepareResponseType:
		d.onPrepareResponse(msg)
	case CommitType:
		d.onCommit(msg)
	case PreCommitType:
		if !d.isAntiMEVExtensionEnabled() {
			d.Logger.Error(fmt.Sprintf("%s message received but AntiMEVExtension is disabled", PreCommitType),
				zap.Uint16("from", msg.ValidatorIndex()),
			)
			return
		}
		d.onPreCommit(msg)
	case RecoveryRequestType:
		d.onRecoveryRequest(msg)
	case RecoveryMessageType:
		d.onRecoveryMessage(msg)
	default:
		d.Logger.DPanic("wrong message type")
	}
}

func (d *DBFT[H]) onPrepareRequest(msg ConsensusPayload[H]) {
	// ignore prepareRequest if we had already received it or
	// are in process of changing view
	if d.RequestSentOrReceived() { //|| (d.ViewChanging() && !d.MoreThanFNodesCommittedOrLost()) {
		d.Logger.Debug("ignoring PrepareRequest",
			zap.Bool("sor", d.RequestSentOrReceived()),
			zap.Bool("viewChanging", d.ViewChanging()),
			zap.Bool("moreThanF", d.MoreThanFNodesCommittedOrLost()))

		return
	}

	if d.ViewNumber != msg.ViewNumber() {
		d.Logger.Debug("ignoring wrong view number", zap.Uint("view", uint(msg.ViewNumber())))
		return
	} else if uint(msg.ValidatorIndex()) != d.GetPrimaryIndex(d.ViewNumber) {
		d.Logger.Info("ignoring PrepareRequest from wrong node", zap.Uint16("from", msg.ValidatorIndex()))
		return
	}

	if err := d.VerifyPrepareRequest(msg); err != nil {
		// We should change view if we receive signed PrepareRequest from the expected validator but it is invalid.
		d.Logger.Warn("invalid PrepareRequest", zap.Uint16("from", msg.ValidatorIndex()), zap.String("error", err.Error()))
		d.sendChangeView(CVBlockRejectedByPolicy)
		return
	}

	d.extendTimer(2)

	p := msg.GetPrepareRequest()

	d.Timestamp = p.Timestamp()
	d.Nonce = p.Nonce()
	d.TransactionHashes = p.TransactionHashes()

	d.Logger.Info("received PrepareRequest", zap.Uint16("validator", msg.ValidatorIndex()), zap.Int("tx", len(d.TransactionHashes)))
	d.processMissingTx()
	d.updateExistingPayloads(msg)
	d.PreparationPayloads[msg.ValidatorIndex()] = msg

	if !d.hasAllTransactions() || !d.createAndCheckBlock() || d.Context.WatchOnly() {
		return
	}

	d.sendPrepareResponse()
	d.checkPrepare()
}

func (d *DBFT[H]) processMissingTx() {
	for _, h := range d.TransactionHashes {
		if _, ok := d.Transactions[h]; ok {
			continue
		}
		if tx := d.GetTx(h); tx == nil {
			d.MissingTransactions = append(d.MissingTransactions, h)
		} else {
			d.Transactions[h] = tx
		}
	}

	if len(d.MissingTransactions) != 0 {
		d.Logger.Info("missing tx",
			zap.Int("count", len(d.MissingTransactions)))
		d.RequestTx(d.MissingTransactions...)
	}
}

// createAndCheckBlock is a prepareRequest-level helper that creates and checks
// the new proposed block, if it's fine it returns true, if something is wrong
// with it, it sends a changeView request and returns false. It's only valid to
// call it when all transactions for this block are already collected.
func (d *DBFT[H]) createAndCheckBlock() bool {
	var blockOK bool
	if d.isAntiMEVExtensionEnabled() {
		b := d.Context.CreatePreBlock()
		blockOK = d.VerifyPreBlock(b)
		if !blockOK {
			d.Logger.Warn("proposed preBlock fails verification")
		}
	} else {
		b := d.Context.CreateBlock()
		blockOK = d.VerifyBlock(b)
		if !blockOK {
			d.Logger.Warn("proposed block fails verification")
		}
	}
	if !blockOK {
		d.sendChangeView(CVTxInvalid)
		return false
	}
	return true
}

// updateExistingPayloads is called _only_ from onPrepareRequest, it validates
// payloads we may have received before PrepareRequest.
func (d *DBFT[H]) updateExistingPayloads(msg ConsensusPayload[H]) {
	for i, m := range d.PreparationPayloads {
		if m != nil && m.Type() == PrepareResponseType {
			resp := m.GetPrepareResponse()
			if resp != nil && resp.PreparationHash() != msg.Hash() {
				d.PreparationPayloads[i] = nil
			}
		}
	}

	if d.isAntiMEVExtensionEnabled() {
		d.verifyPreCommitPayloadsAgainstPreBlock()
		// Commits can't be verified, we have no idea what's the header.
	} else {
		d.verifyCommitPayloadsAgainstHeader()
	}
}

// verifyPreCommitPayloadsAgainstPreBlock performs verification of PreCommit payloads
// against generated PreBlock.
func (d *DBFT[H]) verifyPreCommitPayloadsAgainstPreBlock() {
	if !d.hasAllTransactions() {
		return
	}
	for i, m := range d.PreCommitPayloads {
		if m != nil && m.ViewNumber() == d.ViewNumber {
			if preBlock := d.CreatePreBlock(); preBlock != nil {
				pub := d.Validators[m.ValidatorIndex()]
				if err := preBlock.Verify(pub, m.GetPreCommit().Data()); err != nil {
					d.PreCommitPayloads[i] = nil
					d.Logger.Warn("PreCommit verification failed",
						zap.Uint16("from", m.ValidatorIndex()),
						zap.Error(err))
				}
			}
		}
	}
}

// verifyCommitPayloadsAgainstHeader performs verification of commit payloads
// against generated header.
func (d *DBFT[H]) verifyCommitPayloadsAgainstHeader() {
	for i, m := range d.CommitPayloads {
		if m != nil && m.ViewNumber() == d.ViewNumber {
			if header := d.MakeHeader(); header != nil {
				pub := d.Validators[m.ValidatorIndex()]
				if header.Verify(pub, m.GetCommit().Signature()) != nil {
					d.CommitPayloads[i] = nil
					d.Logger.Warn("can't validate commit signature")
				}
			}
		}
	}
}

func (d *DBFT[H]) onPrepareResponse(msg ConsensusPayload[H]) {
	if d.ViewNumber != msg.ViewNumber() {
		d.Logger.Debug("ignoring wrong view number", zap.Uint("view", uint(msg.ViewNumber())))
		return
	} else if uint(msg.ValidatorIndex()) == d.GetPrimaryIndex(d.ViewNumber) {
		d.Logger.Debug("ignoring PrepareResponse from primary node", zap.Uint16("from", msg.ValidatorIndex()))
		return
	}

	// ignore PrepareResponse if in process of changing view
	m := d.PreparationPayloads[msg.ValidatorIndex()]
	if m != nil || d.ViewChanging() && !d.MoreThanFNodesCommittedOrLost() {
		d.Logger.Debug("ignoring PrepareResponse",
			zap.Bool("dup", m != nil),
			zap.Bool("sor", d.RequestSentOrReceived()),
			zap.Bool("viewChanging", d.ViewChanging()),
			zap.Bool("moreThanF", d.MoreThanFNodesCommittedOrLost()))
		return
	}

	if err := d.VerifyPrepareResponse(msg); err != nil {
		d.Logger.Warn("invalid PrepareResponse", zap.Uint16("from", msg.ValidatorIndex()), zap.String("error", err.Error()))
		return
	}
	d.Logger.Info("received PrepareResponse", zap.Uint16("validator", msg.ValidatorIndex()))
	d.PreparationPayloads[msg.ValidatorIndex()] = msg

	if m = d.PreparationPayloads[d.GetPrimaryIndex(d.ViewNumber)]; m != nil {
		req := m.GetPrepareRequest()
		if req == nil {
			d.Logger.DPanic("unexpected nil prepare request")
			return
		}

		prepHash := msg.GetPrepareResponse().PreparationHash()
		if h := m.Hash(); prepHash != h {
			d.PreparationPayloads[msg.ValidatorIndex()] = nil
			d.Logger.Debug("hash mismatch",
				zap.Stringer("primary", h),
				zap.Stringer("received", prepHash))

			return
		}
	}

	if d.IsPrimary() && !d.prepareSentTime.IsZero() && !d.recovering {
		d.rttEstimates.addTime(time.Since(d.prepareSentTime))
	}

	d.extendTimer(2)

	if !d.Context.WatchOnly() && !d.CommitSent() && (!d.isAntiMEVExtensionEnabled() || !d.PreCommitSent()) && d.RequestSentOrReceived() {
		d.checkPrepare()
	}
}

func (d *DBFT[H]) onChangeView(msg ConsensusPayload[H]) {
	p := msg.GetChangeView()

	if p.NewViewNumber() <= d.ViewNumber {
		d.Logger.Debug("ignoring old ChangeView", zap.Uint("new_view", uint(p.NewViewNumber())))
		d.onRecoveryRequest(msg)

		return
	}

	if d.CommitSent() || d.PreCommitSent() {
		d.Logger.Debug("ignoring ChangeView: preCommit or commit sent")
		d.sendRecoveryMessage()
		return
	}

	m := d.ChangeViewPayloads[msg.ValidatorIndex()]
	if m != nil && p.NewViewNumber() < m.GetChangeView().NewViewNumber() {
		return
	}

	d.Logger.Info("received ChangeView",
		zap.Uint("validator", uint(msg.ValidatorIndex())),
		zap.Stringer("reason", p.Reason()),
		zap.Uint("new view", uint(p.NewViewNumber())),
	)

	d.ChangeViewPayloads[msg.ValidatorIndex()] = msg
	d.checkChangeView(p.NewViewNumber())
}

func (d *DBFT[H]) onPreCommit(msg ConsensusPayload[H]) {
	existing := d.PreCommitPayloads[msg.ValidatorIndex()]
	if existing != nil {
		if existing.Hash() != msg.Hash() {
			d.Logger.Warn("rejecting preCommit due to existing",
				zap.Uint("validator", uint(msg.ValidatorIndex())),
				zap.Uint("existing view", uint(existing.ViewNumber())),
				zap.Uint("view", uint(msg.ViewNumber())),
				zap.Stringer("existing hash", existing.Hash()),
				zap.Stringer("hash", msg.Hash()),
			)
		}
		return
	}
	d.PreCommitPayloads[msg.ValidatorIndex()] = msg
	if d.ViewNumber == msg.ViewNumber() {
		if err := d.VerifyPreCommit(msg); err != nil {
			d.PreCommitPayloads[msg.ValidatorIndex()] = nil
			d.Logger.Warn("invalid PreCommit", zap.Uint16("from", msg.ValidatorIndex()), zap.String("error", err.Error()))
			return
		}

		d.Logger.Info("received PreCommit", zap.Uint("validator", uint(msg.ValidatorIndex())))
		d.extendTimer(4)

		if !d.hasAllTransactions() {
			return
		}
		preBlock := d.CreatePreBlock()
		if preBlock != nil {
			pub := d.Validators[msg.ValidatorIndex()]
			if err := preBlock.Verify(pub, msg.GetPreCommit().Data()); err == nil {
				d.checkPreCommit()
			} else {
				d.PreCommitPayloads[msg.ValidatorIndex()] = nil
				d.Logger.Warn("invalid preCommit data",
					zap.Uint("validator", uint(msg.ValidatorIndex())),
					zap.Error(err),
				)
			}
		}
		return
	}

	d.Logger.Info("received preCommit for different view",
		zap.Uint("validator", uint(msg.ValidatorIndex())),
		zap.Uint("view", uint(msg.ViewNumber())),
	)
}

func (d *DBFT[H]) onCommit(msg ConsensusPayload[H]) {
	existing := d.CommitPayloads[msg.ValidatorIndex()]
	if existing != nil {
		if existing.Hash() != msg.Hash() {
			d.Logger.Warn("rejecting commit due to existing",
				zap.Uint("validator", uint(msg.ValidatorIndex())),
				zap.Uint("existing view", uint(existing.ViewNumber())),
				zap.Uint("view", uint(msg.ViewNumber())),
				zap.Stringer("existing hash", existing.Hash()),
				zap.Stringer("hash", msg.Hash()),
			)
		}
		return
	}
	d.CommitPayloads[msg.ValidatorIndex()] = msg
	if d.ViewNumber == msg.ViewNumber() {
		if err := d.VerifyCommit(msg); err != nil {
			d.CommitPayloads[msg.ValidatorIndex()] = nil
			d.Logger.Warn("invalid Commit", zap.Uint16("from", msg.ValidatorIndex()), zap.String("error", err.Error()))
			return
		}

		d.Logger.Info("received Commit", zap.Uint("validator", uint(msg.ValidatorIndex())))
		d.extendTimer(4)
		header := d.MakeHeader()
		if header != nil {
			pub := d.Validators[msg.ValidatorIndex()]
			if err := header.Verify(pub, msg.GetCommit().Signature()); err == nil {
				d.checkCommit()
			} else {
				d.CommitPayloads[msg.ValidatorIndex()] = nil
				d.Logger.Warn("invalid commit signature",
					zap.Uint("validator", uint(msg.ValidatorIndex())),
					zap.Error(err),
				)
			}
		}

		return
	}

	d.Logger.Info("received commit for different view",
		zap.Uint("validator", uint(msg.ValidatorIndex())),
		zap.Uint("view", uint(msg.ViewNumber())),
	)
}

func (d *DBFT[H]) onRecoveryRequest(msg ConsensusPayload[H]) {
	// Only validators are allowed to send consensus messages.
	if d.Context.WatchOnly() {
		return
	}

	if !d.CommitSent() && (!d.isAntiMEVExtensionEnabled() || !d.PreCommitSent()) {
		// Ignore the message if our index is not in F+1 range of the
		// next (%N) ones from the sender. This limits recovery
		// messages to be broadcasted through the network and F+1
		// guarantees that at least one node responds.

		if (d.MyIndex-int(msg.ValidatorIndex())+d.N()-1)%d.N() > d.F() {
			return
		}
	}

	d.sendRecoveryMessage()
}

func (d *DBFT[H]) onRecoveryMessage(msg ConsensusPayload[H]) {
	d.Logger.Debug("recovery message received", zap.Any("dump", msg))

	var (
		validPrepResp, validChViews   int
		validPreCommits, validCommits int
		validPrepReq, totalPrepReq    int
		recovery                      = msg.GetRecoveryMessage()
		total                         = len(d.Validators)
	)

	// isRecovering is always set to false again after OnRecoveryMessageReceived
	d.recovering = true

	defer func() {
		d.Logger.Sugar().Debugf("recovering finished cv=%d/%d preq=%d/%d presp=%d/%d pco=%d/%d co=%d/%d",
			validChViews, total,
			validPrepReq, totalPrepReq,
			validPrepResp, total,
			validPreCommits, total,
			validCommits, total)
		d.recovering = false
	}()

	if msg.ViewNumber() > d.ViewNumber {
		if d.CommitSent() || d.PreCommitSent() {
			return
		}

		for _, m := range recovery.GetChangeViews(msg, d.Validators) {
			validChViews++
			d.OnReceive(m)
		}
	}

	if msg.ViewNumber() == d.ViewNumber && !(d.ViewChanging() && !d.MoreThanFNodesCommittedOrLost()) && !d.CommitSent() && (!d.isAntiMEVExtensionEnabled() || !d.PreCommitSent()) {
		if !d.RequestSentOrReceived() {
			prepReq := recovery.GetPrepareRequest(msg, d.Validators, uint16(d.PrimaryIndex))
			if prepReq != nil {
				totalPrepReq, validPrepReq = 1, 1
				d.OnReceive(prepReq)
			}
			// If the node is primary, then wait until timer fires to send PrepareRequest
			// to avoid rush in blocks submission, #74.
		}

		for _, m := range recovery.GetPrepareResponses(msg, d.Validators) {
			validPrepResp++
			d.OnReceive(m)
		}
	}

	// Ensure we know about all (pre) commits from lower view numbers.
	if msg.ViewNumber() <= d.ViewNumber {
		for _, m := range recovery.GetPreCommits(msg, d.Validators) {
			validPreCommits++
			d.OnReceive(m)
		}

		for _, m := range recovery.GetCommits(msg, d.Validators) {
			validCommits++
			d.OnReceive(m)
		}
	}
}

func (d *DBFT[H]) changeTimer(delay time.Duration) {
	d.Logger.Debug("reset timer",
		zap.Uint32("h", d.BlockIndex),
		zap.Int("v", int(d.ViewNumber)),
		zap.Duration("delay", delay))
	d.Timer.Reset(d.BlockIndex, d.ViewNumber, delay)
}

func (d *DBFT[H]) extendTimer(count int) {
	if !d.CommitSent() && (!d.isAntiMEVExtensionEnabled() || !d.PreCommitSent()) && !d.ViewChanging() {
		d.Timer.Extend(time.Duration(count) * d.timePerBlock / time.Duration(d.M()))
	}
}
