package dbft

import (
	"errors"
	"time"

	"go.uber.org/zap"
)

// Config contains initialization and working parameters for dBFT.
type Config[H Hash] struct {
	// Logger
	Logger *zap.Logger
	// Timer
	Timer Timer
	// SecondsPerBlock is the number of seconds that
	// need to pass before another block will be accepted.
	SecondsPerBlock time.Duration
	// TimestampIncrement increment is the amount of units to add to timestamp
	// if current time is less than that of previous context.
	// By default use millisecond precision.
	TimestampIncrement uint64
	// GetKeyPair returns an index of the node in the list of validators
	// together with it's key pair.
	GetKeyPair func([]PublicKey) (int, PrivateKey, PublicKey)
	// NewBlockFromContext should allocate, fill from Context and return new block.Block.
	NewBlockFromContext func(ctx *Context[H]) Block[H]
	// RequestTx is a callback which is called when transaction contained
	// in current block can't be found in memory pool.
	RequestTx func(h ...H)
	// StopTxFlow is a callback which is called when the process no longer needs
	// any transactions.
	StopTxFlow func()
	// GetTx returns a transaction from memory pool.
	GetTx func(h H) Transaction[H]
	// GetVerified returns a slice of verified transactions
	// to be proposed in a new block.
	GetVerified func() []Transaction[H]
	// VerifyBlock verifies if block is valid.
	VerifyBlock func(b Block[H]) bool
	// Broadcast should broadcast payload m to the consensus nodes.
	Broadcast func(m ConsensusPayload[H])
	// ProcessBlock is called every time new block is accepted.
	ProcessBlock func(b Block[H])
	// GetBlock should return block with hash.
	GetBlock func(h H) Block[H]
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
	GetValidators func(...Transaction[H]) []PublicKey
	// NewConsensusPayload is a constructor for payload.ConsensusPayload.
	NewConsensusPayload func(*Context[H], MessageType, any) ConsensusPayload[H]
	// NewPrepareRequest is a constructor for payload.PrepareRequest.
	NewPrepareRequest func(ts uint64, nonce uint64, transactionHashes []H) PrepareRequest[H]
	// NewPrepareResponse is a constructor for payload.PrepareResponse.
	NewPrepareResponse func(preparationHash H) PrepareResponse[H]
	// NewChangeView is a constructor for payload.ChangeView.
	NewChangeView func(newViewNumber byte, reason ChangeViewReason, timestamp uint64) ChangeView
	// NewCommit is a constructor for payload.Commit.
	NewCommit func(signature []byte) Commit
	// NewRecoveryRequest is a constructor for payload.RecoveryRequest.
	NewRecoveryRequest func(ts uint64) RecoveryRequest
	// NewRecoveryMessage is a constructor for payload.RecoveryMessage.
	NewRecoveryMessage func() RecoveryMessage[H]
	// VerifyPrepareRequest can perform external payload verification and returns true iff it was successful.
	VerifyPrepareRequest func(p ConsensusPayload[H]) error
	// VerifyPrepareResponse performs external PrepareResponse verification and returns nil if it's successful.
	VerifyPrepareResponse func(p ConsensusPayload[H]) error
}

const defaultSecondsPerBlock = time.Second * 15

const defaultTimestampIncrement = uint64(time.Millisecond / time.Nanosecond)

func defaultConfig[H Hash]() *Config[H] {
	// fields which are set to nil must be provided from client
	return &Config[H]{
		Logger:             zap.NewNop(),
		SecondsPerBlock:    defaultSecondsPerBlock,
		TimestampIncrement: defaultTimestampIncrement,
		GetKeyPair:         nil,
		RequestTx:          func(...H) {},
		StopTxFlow:         func() {},
		GetTx:              func(H) Transaction[H] { return nil },
		GetVerified:        func() []Transaction[H] { return make([]Transaction[H], 0) },
		VerifyBlock:        func(Block[H]) bool { return true },
		Broadcast:          func(ConsensusPayload[H]) {},
		ProcessBlock:       func(Block[H]) {},
		GetBlock:           func(H) Block[H] { return nil },
		WatchOnly:          func() bool { return false },
		CurrentHeight:      nil,
		CurrentBlockHash:   nil,
		GetValidators:      nil,

		VerifyPrepareRequest:  func(ConsensusPayload[H]) error { return nil },
		VerifyPrepareResponse: func(ConsensusPayload[H]) error { return nil },
	}
}

func checkConfig[H Hash](cfg *Config[H]) error {
	if cfg.GetKeyPair == nil {
		return errors.New("private key is nil")
	} else if cfg.Timer == nil {
		return errors.New("Timer is nil")
	} else if cfg.CurrentHeight == nil {
		return errors.New("CurrentHeight is nil")
	} else if cfg.CurrentBlockHash == nil {
		return errors.New("CurrentBlockHash is nil")
	} else if cfg.GetValidators == nil {
		return errors.New("GetValidators is nil")
	} else if cfg.NewBlockFromContext == nil {
		return errors.New("NewBlockFromContext is nil")
	} else if cfg.NewConsensusPayload == nil {
		return errors.New("NewConsensusPayload is nil")
	} else if cfg.NewPrepareRequest == nil {
		return errors.New("NewPrepareRequest is nil")
	} else if cfg.NewPrepareResponse == nil {
		return errors.New("NewPrepareResponse is nil")
	} else if cfg.NewChangeView == nil {
		return errors.New("NewChangeView is nil")
	} else if cfg.NewCommit == nil {
		return errors.New("NewCommit is nil")
	} else if cfg.NewRecoveryRequest == nil {
		return errors.New("NewRecoveryRequest is nil")
	} else if cfg.NewRecoveryMessage == nil {
		return errors.New("NewRecoveryMessage is nil")
	}

	return nil
}

// WithGetKeyPair sets GetKeyPair.
func WithGetKeyPair[H Hash](f func(pubs []PublicKey) (int, PrivateKey, PublicKey)) func(config *Config[H]) {
	return func(cfg *Config[H]) {
		cfg.GetKeyPair = f
	}
}

// WithLogger sets Logger.
func WithLogger[H Hash](log *zap.Logger) func(config *Config[H]) {
	return func(cfg *Config[H]) {
		cfg.Logger = log
	}
}

// WithTimer sets Timer.
func WithTimer[H Hash](t Timer) func(config *Config[H]) {
	return func(cfg *Config[H]) {
		cfg.Timer = t
	}
}

// WithSecondsPerBlock sets SecondsPerBlock.
func WithSecondsPerBlock[H Hash](d time.Duration) func(config *Config[H]) {
	return func(cfg *Config[H]) {
		cfg.SecondsPerBlock = d
	}
}

// WithTimestampIncrement sets TimestampIncrement.
func WithTimestampIncrement[H Hash](u uint64) func(config *Config[H]) {
	return func(cfg *Config[H]) {
		cfg.TimestampIncrement = u
	}
}

// WithNewBlockFromContext sets NewBlockFromContext.
func WithNewBlockFromContext[H Hash](f func(ctx *Context[H]) Block[H]) func(config *Config[H]) {
	return func(cfg *Config[H]) {
		cfg.NewBlockFromContext = f
	}
}

// WithRequestTx sets RequestTx.
func WithRequestTx[H Hash](f func(h ...H)) func(config *Config[H]) {
	return func(cfg *Config[H]) {
		cfg.RequestTx = f
	}
}

// WithStopTxFlow sets StopTxFlow.
func WithStopTxFlow[H Hash](f func()) func(config *Config[H]) {
	return func(cfg *Config[H]) {
		cfg.StopTxFlow = f
	}
}

// WithGetTx sets GetTx.
func WithGetTx[H Hash](f func(h H) Transaction[H]) func(config *Config[H]) {
	return func(cfg *Config[H]) {
		cfg.GetTx = f
	}
}

// WithGetVerified sets GetVerified.
func WithGetVerified[H Hash](f func() []Transaction[H]) func(config *Config[H]) {
	return func(cfg *Config[H]) {
		cfg.GetVerified = f
	}
}

// WithVerifyBlock sets VerifyBlock.
func WithVerifyBlock[H Hash](f func(b Block[H]) bool) func(config *Config[H]) {
	return func(cfg *Config[H]) {
		cfg.VerifyBlock = f
	}
}

// WithBroadcast sets Broadcast.
func WithBroadcast[H Hash](f func(m ConsensusPayload[H])) func(config *Config[H]) {
	return func(cfg *Config[H]) {
		cfg.Broadcast = f
	}
}

// WithProcessBlock sets ProcessBlock.
func WithProcessBlock[H Hash](f func(b Block[H])) func(config *Config[H]) {
	return func(cfg *Config[H]) {
		cfg.ProcessBlock = f
	}
}

// WithGetBlock sets GetBlock.
func WithGetBlock[H Hash](f func(h H) Block[H]) func(config *Config[H]) {
	return func(cfg *Config[H]) {
		cfg.GetBlock = f
	}
}

// WithWatchOnly sets WatchOnly.
func WithWatchOnly[H Hash](f func() bool) func(config *Config[H]) {
	return func(cfg *Config[H]) {
		cfg.WatchOnly = f
	}
}

// WithCurrentHeight sets CurrentHeight.
func WithCurrentHeight[H Hash](f func() uint32) func(config *Config[H]) {
	return func(cfg *Config[H]) {
		cfg.CurrentHeight = f
	}
}

// WithCurrentBlockHash sets CurrentBlockHash.
func WithCurrentBlockHash[H Hash](f func() H) func(config *Config[H]) {
	return func(cfg *Config[H]) {
		cfg.CurrentBlockHash = f
	}
}

// WithGetValidators sets GetValidators.
func WithGetValidators[H Hash](f func(txs ...Transaction[H]) []PublicKey) func(config *Config[H]) {
	return func(cfg *Config[H]) {
		cfg.GetValidators = f
	}
}

// WithNewConsensusPayload sets NewConsensusPayload.
func WithNewConsensusPayload[H Hash](f func(ctx *Context[H], typ MessageType, msg any) ConsensusPayload[H]) func(config *Config[H]) {
	return func(cfg *Config[H]) {
		cfg.NewConsensusPayload = f
	}
}

// WithNewPrepareRequest sets NewPrepareRequest.
func WithNewPrepareRequest[H Hash](f func(ts uint64, nonce uint64, transactionsHashes []H) PrepareRequest[H]) func(config *Config[H]) {
	return func(cfg *Config[H]) {
		cfg.NewPrepareRequest = f
	}
}

// WithNewPrepareResponse sets NewPrepareResponse.
func WithNewPrepareResponse[H Hash](f func(preparationHash H) PrepareResponse[H]) func(config *Config[H]) {
	return func(cfg *Config[H]) {
		cfg.NewPrepareResponse = f
	}
}

// WithNewChangeView sets NewChangeView.
func WithNewChangeView[H Hash](f func(newViewNumber byte, reason ChangeViewReason, ts uint64) ChangeView) func(config *Config[H]) {
	return func(cfg *Config[H]) {
		cfg.NewChangeView = f
	}
}

// WithNewCommit sets NewCommit.
func WithNewCommit[H Hash](f func(signature []byte) Commit) func(config *Config[H]) {
	return func(cfg *Config[H]) {
		cfg.NewCommit = f
	}
}

// WithNewRecoveryRequest sets NewRecoveryRequest.
func WithNewRecoveryRequest[H Hash](f func(ts uint64) RecoveryRequest) func(config *Config[H]) {
	return func(cfg *Config[H]) {
		cfg.NewRecoveryRequest = f
	}
}

// WithNewRecoveryMessage sets NewRecoveryMessage.
func WithNewRecoveryMessage[H Hash](f func() RecoveryMessage[H]) func(config *Config[H]) {
	return func(cfg *Config[H]) {
		cfg.NewRecoveryMessage = f
	}
}

// WithVerifyPrepareRequest sets VerifyPrepareRequest.
func WithVerifyPrepareRequest[H Hash](f func(prepareReq ConsensusPayload[H]) error) func(config *Config[H]) {
	return func(cfg *Config[H]) {
		cfg.VerifyPrepareRequest = f
	}
}

// WithVerifyPrepareResponse sets VerifyPrepareResponse.
func WithVerifyPrepareResponse[H Hash](f func(prepareResp ConsensusPayload[H]) error) func(config *Config[H]) {
	return func(cfg *Config[H]) {
		cfg.VerifyPrepareResponse = f
	}
}
