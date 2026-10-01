package dbft

import (
	"errors"
	"iter"
	"time"

	"go.uber.org/zap"
)

// Config contains initialization and working parameters for dBFT.
type Config[H Hash, Tx Transaction[H]] struct {
	// Logger
	Logger *zap.Logger
	// Timer
	Timer Timer
	// TimePerBlock is the minimum time that needs to pass before another block
	// will be accepted even if there are pending transactions in the node's
	// mempool. This value may be updated every block.
	TimePerBlock func() time.Duration
	// MaxTimePerBlock is the maximum time that may pass before another block is
	// accepted if there are no pending transactions in the node's mempool. This
	// value may be updated every block. If set, enables dynamic block time
	// extension: blocks are accepted with interval from TimePerBlock to
	// MaxTimePerBlock (in CV-less scenario) depending on the presence of
	// transactions in the node's pool, ref.
	// https://github.com/neo-project/neo/issues/4018.
	MaxTimePerBlock func() time.Duration
	// TimestampIncrement increment is the amount of units to add to timestamp
	// if current time is less than that of previous context.
	// By default use millisecond precision.
	TimestampIncrement uint64
	// AntiMEVExtensionEnablingHeight denotes the height starting from which dBFT
	// Anti-MEV extensions should be enabled. -1 means no extension is enabled.
	AntiMEVExtensionEnablingHeight int64
	// GetKeyPair returns an index of the node in the list of validators
	// together with it's key pair.
	GetKeyPair func([]PublicKey) (int, PrivateKey, PublicKey)
	// NewPreBlockFromContext should allocate, fill from Context and return new block.PreBlock.
	NewPreBlockFromContext func(ctx *Context[H, Tx]) PreBlock[H, Tx]
	// NewBlockFromContext should allocate, fill from Context and return new block.Block.
	NewBlockFromContext func(ctx *Context[H, Tx]) Block[H, Tx]
	// RequestTx is a callback which is called when transaction contained
	// in current block can't be found in memory pool. The slice received by
	// this callback MUST NOT be changed.
	RequestTx func(h iter.Seq[H])
	// SubscribeForTxs is a callback which is called when dBFT needs to track incoming
	// mempool transactions. Subscription is supposed to be single-use, no unsubscription
	// is initiated by dBFT, hence it's the user's duty to manage and release resources.
	// This callback is active iff MaxTimePerBlock is set.
	SubscribeForTxs func()
	// StopTxFlow is a callback which is called when the process no longer needs
	// any transactions.
	StopTxFlow func()
	// GetVerified returns a slice of verified transactions
	// to be proposed in a new block.
	GetVerified func() []Tx
	// VerifyPreBlock verifies if preBlock is valid.
	VerifyPreBlock func(b PreBlock[H, Tx]) bool
	// VerifyBlock verifies if block is valid.
	VerifyBlock func(b Block[H, Tx]) bool
	// Broadcast should broadcast payload m to the consensus nodes.
	Broadcast func(m ConsensusPayload[H, Tx])
	// ProcessBlock is called every time new preBlock is accepted.
	ProcessPreBlock func(b PreBlock[H, Tx]) error
	// ProcessBlock is called every time new block is accepted.
	ProcessBlock func(b Block[H, Tx]) error
	// GetBlock should return block with hash.
	GetBlock func(h H) Block[H, Tx]
	// WatchOnly tells if a node should only watch.
	WatchOnly func() bool
	// CurrentHeight returns index of the last accepted block.
	CurrentHeight func() uint32
	// CurrentBlockHash returns hash of the last accepted block.
	CurrentBlockHash func() H
	// GetValidators returns list of the validators.
	// When called with a transaction list it must return
	// list of the validators of the next block.
	// If this function ever returns 0-length slice, dbft will panic.
	GetValidators func(...Tx) []PublicKey
	// NewConsensusPayload is a constructor for payload.ConsensusPayload.
	NewConsensusPayload func(*Context[H, Tx], MessageType, any) ConsensusPayload[H, Tx]
	// NewPrepareRequest is a constructor for payload.PrepareRequest.
	NewPrepareRequest func(ts uint64, nonce uint64, txes []Tx) PrepareRequest[H, Tx]
	// NewPrepareResponse is a constructor for payload.PrepareResponse.
	NewPrepareResponse func(preparationHash H) PrepareResponse[H]
	// NewChangeView is a constructor for payload.ChangeView.
	NewChangeView func(newViewNumber byte, reason ChangeViewReason, timestamp uint64) ChangeView
	// NewPreCommit is a constructor for payload.PreCommit.
	NewPreCommit func(data []byte) PreCommit
	// NewCommit is a constructor for payload.Commit.
	NewCommit func(signature []byte) Commit
	// NewRecoveryRequest is a constructor for payload.RecoveryRequest.
	NewRecoveryRequest func(ts uint64) RecoveryRequest
	// NewRecoveryMessage is a constructor for payload.RecoveryMessage.
	NewRecoveryMessage func() RecoveryMessage[H, Tx]
	// VerifyPrepareRequest can perform external payload verification and returns true iff it was successful.
	VerifyPrepareRequest func(p ConsensusPayload[H, Tx]) error
	// VerifyPrepareResponse performs external PrepareResponse verification and returns nil if it's successful.
	VerifyPrepareResponse func(p ConsensusPayload[H, Tx]) error
	// VerifyPreCommit performs external PreCommit verification and returns nil if it's successful.
	// Note that PreBlock-dependent PreCommit verification should be performed inside PreBlock.Verify
	// callback.
	VerifyPreCommit func(p ConsensusPayload[H, Tx]) error
	// VerifyCommit performs external Commit verification and returns nil if it's successful.
	// Note that Block-dependent Commit verification should be performed inside Block.Verify
	// callback.
	VerifyCommit func(p ConsensusPayload[H, Tx]) error
}

const defaultSecondsPerBlock = time.Second * 15

const defaultTimestampIncrement = uint64(time.Millisecond / time.Nanosecond)

func defaultConfig[H Hash, Tx Transaction[H]]() *Config[H, Tx] {
	// fields which are set to nil must be provided from client
	return &Config[H, Tx]{
		Logger:             zap.NewNop(),
		TimePerBlock:       func() time.Duration { return defaultSecondsPerBlock },
		TimestampIncrement: defaultTimestampIncrement,
		GetKeyPair:         nil,
		RequestTx:          func(iter.Seq[H]) {},
		StopTxFlow:         func() {},
		GetVerified:        func() []Tx { return make([]Tx, 0) },
		VerifyBlock:        func(Block[H, Tx]) bool { return true },
		Broadcast:          func(ConsensusPayload[H, Tx]) {},
		ProcessBlock:       func(Block[H, Tx]) error { return nil },
		GetBlock:           func(H) Block[H, Tx] { return nil },
		WatchOnly:          func() bool { return false },
		CurrentHeight:      nil,
		CurrentBlockHash:   nil,
		GetValidators:      nil,

		VerifyPrepareRequest:  func(ConsensusPayload[H, Tx]) error { return nil },
		VerifyPrepareResponse: func(ConsensusPayload[H, Tx]) error { return nil },
		VerifyCommit:          func(ConsensusPayload[H, Tx]) error { return nil },

		AntiMEVExtensionEnablingHeight: -1,
		VerifyPreBlock:                 func(PreBlock[H, Tx]) bool { return true },
		VerifyPreCommit:                func(ConsensusPayload[H, Tx]) error { return nil },
	}
}

func checkConfig[H Hash, Tx Transaction[H]](cfg *Config[H, Tx]) error {
	if cfg.GetKeyPair == nil {
		return errors.New("private key is nil")
	}
	if cfg.Timer == nil {
		return errors.New("Timer is nil")
	}
	if cfg.CurrentHeight == nil {
		return errors.New("CurrentHeight is nil")
	}
	if cfg.CurrentBlockHash == nil {
		return errors.New("CurrentBlockHash is nil")
	}
	if cfg.GetValidators == nil {
		return errors.New("GetValidators is nil")
	}
	if cfg.NewBlockFromContext == nil {
		return errors.New("NewBlockFromContext is nil")
	}
	if cfg.NewConsensusPayload == nil {
		return errors.New("NewConsensusPayload is nil")
	}
	if cfg.NewPrepareRequest == nil {
		return errors.New("NewPrepareRequest is nil")
	}
	if cfg.NewPrepareResponse == nil {
		return errors.New("NewPrepareResponse is nil")
	}
	if cfg.NewChangeView == nil {
		return errors.New("NewChangeView is nil")
	}
	if cfg.NewCommit == nil {
		return errors.New("NewCommit is nil")
	}
	if cfg.NewRecoveryRequest == nil {
		return errors.New("NewRecoveryRequest is nil")
	}
	if cfg.NewRecoveryMessage == nil {
		return errors.New("NewRecoveryMessage is nil")
	}
	if cfg.AntiMEVExtensionEnablingHeight >= 0 {
		if cfg.NewPreBlockFromContext == nil {
			return errors.New("NewPreBlockFromContext is nil")
		}
		if cfg.ProcessPreBlock == nil {
			return errors.New("ProcessPreBlock is nil")
		}
		if cfg.NewPreCommit == nil {
			return errors.New("NewPreCommit is nil")
		}
	} else {
		if cfg.NewPreBlockFromContext != nil {
			return errors.New("NewPreBlockFromContext is set, but AntiMEVExtensionEnablingHeight is not specified")
		}
		if cfg.ProcessPreBlock != nil {
			return errors.New("ProcessPreBlock is set, but AntiMEVExtensionEnablingHeight is not specified")
		}
		if cfg.NewPreCommit != nil {
			return errors.New("NewPreCommit is set, but AntiMEVExtensionEnablingHeight is not specified")
		}
	}
	if (cfg.MaxTimePerBlock == nil) != (cfg.SubscribeForTxs == nil) {
		return errors.New("MaxTimePerBlock and SubscribeForTxs should be specified/not specified at the same time")
	}

	return nil
}

// WithGetKeyPair sets GetKeyPair.
func WithGetKeyPair[H Hash, Tx Transaction[H]](f func(pubs []PublicKey) (int, PrivateKey, PublicKey)) func(config *Config[H, Tx]) {
	return func(cfg *Config[H, Tx]) {
		cfg.GetKeyPair = f
	}
}

// WithLogger sets Logger.
func WithLogger[H Hash, Tx Transaction[H]](log *zap.Logger) func(config *Config[H, Tx]) {
	return func(cfg *Config[H, Tx]) {
		cfg.Logger = log
	}
}

// WithTimer sets Timer.
func WithTimer[H Hash, Tx Transaction[H]](t Timer) func(config *Config[H, Tx]) {
	return func(cfg *Config[H, Tx]) {
		cfg.Timer = t
	}
}

// WithTimePerBlock sets TimePerBlock.
func WithTimePerBlock[H Hash, Tx Transaction[H]](f func() time.Duration) func(config *Config[H, Tx]) {
	return func(cfg *Config[H, Tx]) {
		cfg.TimePerBlock = f
	}
}

// WithMaxTimePerBlock sets MaxTimePerBlock.
func WithMaxTimePerBlock[H Hash, Tx Transaction[H]](f func() time.Duration) func(config *Config[H, Tx]) {
	return func(cfg *Config[H, Tx]) {
		cfg.MaxTimePerBlock = f
	}
}

// WithAntiMEVExtensionEnablingHeight sets AntiMEVExtensionEnablingHeight.
func WithAntiMEVExtensionEnablingHeight[H Hash, Tx Transaction[H]](h int64) func(config *Config[H, Tx]) {
	return func(cfg *Config[H, Tx]) {
		cfg.AntiMEVExtensionEnablingHeight = h
	}
}

// WithTimestampIncrement sets TimestampIncrement.
func WithTimestampIncrement[H Hash, Tx Transaction[H]](u uint64) func(config *Config[H, Tx]) {
	return func(cfg *Config[H, Tx]) {
		cfg.TimestampIncrement = u
	}
}

// WithNewPreBlockFromContext sets NewPreBlockFromContext.
func WithNewPreBlockFromContext[H Hash, Tx Transaction[H]](f func(ctx *Context[H, Tx]) PreBlock[H, Tx]) func(config *Config[H, Tx]) {
	return func(cfg *Config[H, Tx]) {
		cfg.NewPreBlockFromContext = f
	}
}

// WithNewBlockFromContext sets NewBlockFromContext.
func WithNewBlockFromContext[H Hash, Tx Transaction[H]](f func(ctx *Context[H, Tx]) Block[H, Tx]) func(config *Config[H, Tx]) {
	return func(cfg *Config[H, Tx]) {
		cfg.NewBlockFromContext = f
	}
}

// WithRequestTx sets RequestTx.
func WithRequestTx[H Hash, Tx Transaction[H]](f func(hs iter.Seq[H])) func(config *Config[H, Tx]) {
	return func(cfg *Config[H, Tx]) {
		cfg.RequestTx = f
	}
}

// WithSubscribeForTxs sets SubscribeForTxs.
func WithSubscribeForTxs[H Hash, Tx Transaction[H]](f func()) func(config *Config[H, Tx]) {
	return func(cfg *Config[H, Tx]) {
		cfg.SubscribeForTxs = f
	}
}

// WithStopTxFlow sets StopTxFlow.
func WithStopTxFlow[H Hash, Tx Transaction[H]](f func()) func(config *Config[H, Tx]) {
	return func(cfg *Config[H, Tx]) {
		cfg.StopTxFlow = f
	}
}

// WithGetVerified sets GetVerified.
func WithGetVerified[H Hash, Tx Transaction[H]](f func() []Tx) func(config *Config[H, Tx]) {
	return func(cfg *Config[H, Tx]) {
		cfg.GetVerified = f
	}
}

// WithVerifyPreBlock sets VerifyPreBlock.
func WithVerifyPreBlock[H Hash, Tx Transaction[H]](f func(b PreBlock[H, Tx]) bool) func(config *Config[H, Tx]) {
	return func(cfg *Config[H, Tx]) {
		cfg.VerifyPreBlock = f
	}
}

// WithVerifyBlock sets VerifyBlock.
func WithVerifyBlock[H Hash, Tx Transaction[H]](f func(b Block[H, Tx]) bool) func(config *Config[H, Tx]) {
	return func(cfg *Config[H, Tx]) {
		cfg.VerifyBlock = f
	}
}

// WithBroadcast sets Broadcast.
func WithBroadcast[H Hash, Tx Transaction[H]](f func(m ConsensusPayload[H, Tx])) func(config *Config[H, Tx]) {
	return func(cfg *Config[H, Tx]) {
		cfg.Broadcast = f
	}
}

// WithProcessBlock sets ProcessBlock callback. Note that for anti-MEV extension
// disabled non-nil error return is a no-op.
func WithProcessBlock[H Hash, Tx Transaction[H]](f func(b Block[H, Tx]) error) func(config *Config[H, Tx]) {
	return func(cfg *Config[H, Tx]) {
		cfg.ProcessBlock = f
	}
}

// WithProcessPreBlock sets ProcessPreBlock.
func WithProcessPreBlock[H Hash, Tx Transaction[H]](f func(b PreBlock[H, Tx]) error) func(config *Config[H, Tx]) {
	return func(cfg *Config[H, Tx]) {
		cfg.ProcessPreBlock = f
	}
}

// WithGetBlock sets GetBlock.
func WithGetBlock[H Hash, Tx Transaction[H]](f func(h H) Block[H, Tx]) func(config *Config[H, Tx]) {
	return func(cfg *Config[H, Tx]) {
		cfg.GetBlock = f
	}
}

// WithWatchOnly sets WatchOnly.
func WithWatchOnly[H Hash, Tx Transaction[H]](f func() bool) func(config *Config[H, Tx]) {
	return func(cfg *Config[H, Tx]) {
		cfg.WatchOnly = f
	}
}

// WithCurrentHeight sets CurrentHeight.
func WithCurrentHeight[H Hash, Tx Transaction[H]](f func() uint32) func(config *Config[H, Tx]) {
	return func(cfg *Config[H, Tx]) {
		cfg.CurrentHeight = f
	}
}

// WithCurrentBlockHash sets CurrentBlockHash.
func WithCurrentBlockHash[H Hash, Tx Transaction[H]](f func() H) func(config *Config[H, Tx]) {
	return func(cfg *Config[H, Tx]) {
		cfg.CurrentBlockHash = f
	}
}

// WithGetValidators sets GetValidators.
func WithGetValidators[H Hash, Tx Transaction[H]](f func(txs ...Tx) []PublicKey) func(config *Config[H, Tx]) {
	return func(cfg *Config[H, Tx]) {
		cfg.GetValidators = f
	}
}

// WithNewConsensusPayload sets NewConsensusPayload.
func WithNewConsensusPayload[H Hash, Tx Transaction[H]](f func(ctx *Context[H, Tx], typ MessageType, msg any) ConsensusPayload[H, Tx]) func(config *Config[H, Tx]) {
	return func(cfg *Config[H, Tx]) {
		cfg.NewConsensusPayload = f
	}
}

// WithNewPrepareRequest sets NewPrepareRequest.
func WithNewPrepareRequest[H Hash, Tx Transaction[H]](f func(ts uint64, nonce uint64, transactionHashes []Tx) PrepareRequest[H, Tx]) func(config *Config[H, Tx]) {
	return func(cfg *Config[H, Tx]) {
		cfg.NewPrepareRequest = f
	}
}

// WithNewPrepareResponse sets NewPrepareResponse.
func WithNewPrepareResponse[H Hash, Tx Transaction[H]](f func(preparationHash H) PrepareResponse[H]) func(config *Config[H, Tx]) {
	return func(cfg *Config[H, Tx]) {
		cfg.NewPrepareResponse = f
	}
}

// WithNewChangeView sets NewChangeView.
func WithNewChangeView[H Hash, Tx Transaction[H]](f func(newViewNumber byte, reason ChangeViewReason, ts uint64) ChangeView) func(config *Config[H, Tx]) {
	return func(cfg *Config[H, Tx]) {
		cfg.NewChangeView = f
	}
}

// WithNewCommit sets NewCommit.
func WithNewCommit[H Hash, Tx Transaction[H]](f func(signature []byte) Commit) func(config *Config[H, Tx]) {
	return func(cfg *Config[H, Tx]) {
		cfg.NewCommit = f
	}
}

// WithNewPreCommit sets NewPreCommit.
func WithNewPreCommit[H Hash, Tx Transaction[H]](f func(signature []byte) PreCommit) func(config *Config[H, Tx]) {
	return func(cfg *Config[H, Tx]) {
		cfg.NewPreCommit = f
	}
}

// WithNewRecoveryRequest sets NewRecoveryRequest.
func WithNewRecoveryRequest[H Hash, Tx Transaction[H]](f func(ts uint64) RecoveryRequest) func(config *Config[H, Tx]) {
	return func(cfg *Config[H, Tx]) {
		cfg.NewRecoveryRequest = f
	}
}

// WithNewRecoveryMessage sets NewRecoveryMessage.
func WithNewRecoveryMessage[H Hash, Tx Transaction[H]](f func() RecoveryMessage[H, Tx]) func(config *Config[H, Tx]) {
	return func(cfg *Config[H, Tx]) {
		cfg.NewRecoveryMessage = f
	}
}

// WithVerifyPrepareRequest sets VerifyPrepareRequest.
func WithVerifyPrepareRequest[H Hash, Tx Transaction[H]](f func(prepareReq ConsensusPayload[H, Tx]) error) func(config *Config[H, Tx]) {
	return func(cfg *Config[H, Tx]) {
		cfg.VerifyPrepareRequest = f
	}
}

// WithVerifyPrepareResponse sets VerifyPrepareResponse.
func WithVerifyPrepareResponse[H Hash, Tx Transaction[H]](f func(prepareResp ConsensusPayload[H, Tx]) error) func(config *Config[H, Tx]) {
	return func(cfg *Config[H, Tx]) {
		cfg.VerifyPrepareResponse = f
	}
}

// WithVerifyPreCommit sets VerifyPreCommit.
func WithVerifyPreCommit[H Hash, Tx Transaction[H]](f func(preCommit ConsensusPayload[H, Tx]) error) func(config *Config[H, Tx]) {
	return func(cfg *Config[H, Tx]) {
		cfg.VerifyPreCommit = f
	}
}

// WithVerifyCommit sets VerifyCommit.
func WithVerifyCommit[H Hash, Tx Transaction[H]](f func(commit ConsensusPayload[H, Tx]) error) func(config *Config[H, Tx]) {
	return func(cfg *Config[H, Tx]) {
		cfg.VerifyCommit = f
	}
}
