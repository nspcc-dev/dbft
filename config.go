package dbft

import (
	"bytes"
	"errors"
	"time"

	"github.com/nspcc-dev/dbft/block"
	"github.com/nspcc-dev/dbft/crypto"
	"github.com/nspcc-dev/dbft/payload"
	"github.com/nspcc-dev/dbft/timer"
	"go.uber.org/zap"
)

// Config contains initialization and working parameters for dBFT.
type Config[H crypto.Hash, A crypto.Address] struct {
	// Logger
	Logger *zap.Logger
	// Timer
	Timer timer.Timer
	// SecondsPerBlock is the number of seconds that
	// need to pass before another block will be accepted.
	SecondsPerBlock time.Duration
	// TimestampIncrement increment is the amount of units to add to timestamp
	// if current time is less than that of previous context.
	// By default use millisecond precision.
	TimestampIncrement uint64
	// GetKeyPair returns an index of the node in the list of validators
	// together with it's key pair.
	GetKeyPair func([]crypto.PublicKey) (int, crypto.PrivateKey, crypto.PublicKey)
	// NewBlockFromContext should allocate, fill from Context and return new block.Block.
	NewBlockFromContext func(ctx *Context[H, A]) block.Block[H, A]
	// RequestTx is a callback which is called when transaction contained
	// in current block can't be found in memory pool.
	RequestTx func(h ...H)
	// StopTxFlow is a callback which is called when the process no longer needs
	// any transactions.
	StopTxFlow func()
	// GetTx returns a transaction from memory pool.
	GetTx func(h H) block.Transaction[H]
	// GetVerified returns a slice of verified transactions
	// to be proposed in a new block.
	GetVerified func() []block.Transaction[H]
	// VerifyBlock verifies if block is valid.
	VerifyBlock func(b block.Block[H, A]) bool
	// Broadcast should broadcast payload m to the consensus nodes.
	Broadcast func(m payload.ConsensusPayload[H, A])
	// ProcessBlock is called every time new block is accepted.
	ProcessBlock func(b block.Block[H, A])
	// GetBlock should return block with hash.
	GetBlock func(h H) block.Block[H, A]
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
	GetValidators func(...block.Transaction[H]) []crypto.PublicKey
	// GetConsensusAddress returns hash of the validator list.
	GetConsensusAddress func(...crypto.PublicKey) A
	// NewConsensusPayload is a constructor for payload.ConsensusPayload.
	NewConsensusPayload func(*Context[H, A], payload.MessageType, any) payload.ConsensusPayload[H, A]
	// NewPrepareRequest is a constructor for payload.PrepareRequest.
	NewPrepareRequest func() payload.PrepareRequest[H, A]
	// NewPrepareResponse is a constructor for payload.PrepareResponse.
	NewPrepareResponse func() payload.PrepareResponse[H]
	// NewChangeView is a constructor for payload.ChangeView.
	NewChangeView func() payload.ChangeView
	// NewCommit is a constructor for payload.Commit.
	NewCommit func() payload.Commit
	// NewRecoveryRequest is a constructor for payload.RecoveryRequest.
	NewRecoveryRequest func() payload.RecoveryRequest
	// NewRecoveryMessage is a constructor for payload.RecoveryMessage.
	NewRecoveryMessage func() payload.RecoveryMessage[H, A]
	// VerifyPrepareRequest can perform external payload verification and returns true iff it was successful.
	VerifyPrepareRequest func(p payload.ConsensusPayload[H, A]) error
	// VerifyPrepareResponse performs external PrepareResponse verification and returns nil if it's successful.
	VerifyPrepareResponse func(p payload.ConsensusPayload[H, A]) error
}

const defaultSecondsPerBlock = time.Second * 15

const defaultTimestampIncrement = uint64(time.Millisecond / time.Nanosecond)

func defaultConfig[H crypto.Hash, A crypto.Address]() *Config[H, A] {
	// fields which are set to nil must be provided from client
	return &Config[H, A]{
		Logger:             zap.NewNop(),
		Timer:              timer.New(),
		SecondsPerBlock:    defaultSecondsPerBlock,
		TimestampIncrement: defaultTimestampIncrement,
		GetKeyPair:         nil,
		RequestTx:          func(...H) {},
		StopTxFlow:         func() {},
		GetTx:              func(H) block.Transaction[H] { return nil },
		GetVerified:        func() []block.Transaction[H] { return make([]block.Transaction[H], 0) },
		VerifyBlock:        func(block.Block[H, A]) bool { return true },
		Broadcast:          func(payload.ConsensusPayload[H, A]) {},
		ProcessBlock:       func(block.Block[H, A]) {},
		GetBlock:           func(H) block.Block[H, A] { return nil },
		WatchOnly:          func() bool { return false },
		CurrentHeight:      nil,
		CurrentBlockHash:   nil,
		GetValidators:      nil,

		VerifyPrepareRequest:  func(payload.ConsensusPayload[H, A]) error { return nil },
		VerifyPrepareResponse: func(payload.ConsensusPayload[H, A]) error { return nil },
	}
}

func checkConfig[H crypto.Hash, A crypto.Address](cfg *Config[H, A]) error {
	if cfg.GetKeyPair == nil {
		return errors.New("private key is nil")
	} else if cfg.CurrentHeight == nil {
		return errors.New("CurrentHeight is nil")
	} else if cfg.CurrentBlockHash == nil {
		return errors.New("CurrentBlockHash is nil")
	} else if cfg.GetValidators == nil {
		return errors.New("GetValidators is nil")
	} else if cfg.NewBlockFromContext == nil {
		return errors.New("NewBlockFromContext is nil")
	} else if cfg.GetConsensusAddress == nil {
		return errors.New("GetConsensusAddress is nil")
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

// WithKeyPair sets GetKeyPair to a function returning default key pair
// if it is present in a list of validators.
func WithKeyPair[H crypto.Hash, A crypto.Address](priv crypto.PrivateKey, pub crypto.PublicKey) func(config *Config[H, A]) {
	myPub, err := pub.MarshalBinary()
	if err != nil {
		return nil
	}

	return func(cfg *Config[H, A]) {
		cfg.GetKeyPair = func(ps []crypto.PublicKey) (int, crypto.PrivateKey, crypto.PublicKey) {
			for i := range ps {
				pi, err := ps[i].MarshalBinary()
				if err != nil {
					continue
				} else if bytes.Equal(myPub, pi) {
					return i, priv, pub
				}
			}

			return -1, nil, nil
		}
	}
}

// WithGetKeyPair sets GetKeyPair.
func WithGetKeyPair[H crypto.Hash, A crypto.Address](f func([]crypto.PublicKey) (int, crypto.PrivateKey, crypto.PublicKey)) func(config *Config[H, A]) {
	return func(cfg *Config[H, A]) {
		cfg.GetKeyPair = f
	}
}

// WithLogger sets Logger.
func WithLogger[H crypto.Hash, A crypto.Address](log *zap.Logger) func(config *Config[H, A]) {
	return func(cfg *Config[H, A]) {
		cfg.Logger = log
	}
}

// WithTimer sets Timer.
func WithTimer[H crypto.Hash, A crypto.Address](t timer.Timer) func(config *Config[H, A]) {
	return func(cfg *Config[H, A]) {
		cfg.Timer = t
	}
}

// WithSecondsPerBlock sets SecondsPerBlock.
func WithSecondsPerBlock[H crypto.Hash, A crypto.Address](d time.Duration) func(config *Config[H, A]) {
	return func(cfg *Config[H, A]) {
		cfg.SecondsPerBlock = d
	}
}

// WithTimestampIncrement sets TimestampIncrement.
func WithTimestampIncrement[H crypto.Hash, A crypto.Address](u uint64) func(config *Config[H, A]) {
	return func(cfg *Config[H, A]) {
		cfg.TimestampIncrement = u
	}
}

// WithNewBlockFromContext sets NewBlockFromContext.
func WithNewBlockFromContext[H crypto.Hash, A crypto.Address](f func(ctx *Context[H, A]) block.Block[H, A]) func(config *Config[H, A]) {
	return func(cfg *Config[H, A]) {
		cfg.NewBlockFromContext = f
	}
}

// WithRequestTx sets RequestTx.
func WithRequestTx[H crypto.Hash, A crypto.Address](f func(h ...H)) func(config *Config[H, A]) {
	return func(cfg *Config[H, A]) {
		cfg.RequestTx = f
	}
}

// WithStopTxFlow sets StopTxFlow.
func WithStopTxFlow[H crypto.Hash, A crypto.Address](f func()) func(config *Config[H, A]) {
	return func(cfg *Config[H, A]) {
		cfg.StopTxFlow = f
	}
}

// WithGetTx sets GetTx.
func WithGetTx[H crypto.Hash, A crypto.Address](f func(h H) block.Transaction[H]) func(config *Config[H, A]) {
	return func(cfg *Config[H, A]) {
		cfg.GetTx = f
	}
}

// WithGetVerified sets GetVerified.
func WithGetVerified[H crypto.Hash, A crypto.Address](f func() []block.Transaction[H]) func(config *Config[H, A]) {
	return func(cfg *Config[H, A]) {
		cfg.GetVerified = f
	}
}

// WithVerifyBlock sets VerifyBlock.
func WithVerifyBlock[H crypto.Hash, A crypto.Address](f func(b block.Block[H, A]) bool) func(config *Config[H, A]) {
	return func(cfg *Config[H, A]) {
		cfg.VerifyBlock = f
	}
}

// WithBroadcast sets Broadcast.
func WithBroadcast[H crypto.Hash, A crypto.Address](f func(m payload.ConsensusPayload[H, A])) func(config *Config[H, A]) {
	return func(cfg *Config[H, A]) {
		cfg.Broadcast = f
	}
}

// WithProcessBlock sets ProcessBlock.
func WithProcessBlock[H crypto.Hash, A crypto.Address](f func(b block.Block[H, A])) func(config *Config[H, A]) {
	return func(cfg *Config[H, A]) {
		cfg.ProcessBlock = f
	}
}

// WithGetBlock sets GetBlock.
func WithGetBlock[H crypto.Hash, A crypto.Address](f func(h H) block.Block[H, A]) func(config *Config[H, A]) {
	return func(cfg *Config[H, A]) {
		cfg.GetBlock = f
	}
}

// WithWatchOnly sets WatchOnly.
func WithWatchOnly[H crypto.Hash, A crypto.Address](f func() bool) func(config *Config[H, A]) {
	return func(cfg *Config[H, A]) {
		cfg.WatchOnly = f
	}
}

// WithCurrentHeight sets CurrentHeight.
func WithCurrentHeight[H crypto.Hash, A crypto.Address](f func() uint32) func(config *Config[H, A]) {
	return func(cfg *Config[H, A]) {
		cfg.CurrentHeight = f
	}
}

// WithCurrentBlockHash sets CurrentBlockHash.
func WithCurrentBlockHash[H crypto.Hash, A crypto.Address](f func() H) func(config *Config[H, A]) {
	return func(cfg *Config[H, A]) {
		cfg.CurrentBlockHash = f
	}
}

// WithGetValidators sets GetValidators.
func WithGetValidators[H crypto.Hash, A crypto.Address](f func(...block.Transaction[H]) []crypto.PublicKey) func(config *Config[H, A]) {
	return func(cfg *Config[H, A]) {
		cfg.GetValidators = f
	}
}

// WithGetConsensusAddress sets GetConsensusAddress.
func WithGetConsensusAddress[H crypto.Hash, A crypto.Address](f func(keys ...crypto.PublicKey) A) func(config *Config[H, A]) {
	return func(cfg *Config[H, A]) {
		cfg.GetConsensusAddress = f
	}
}

// WithNewConsensusPayload sets NewConsensusPayload.
func WithNewConsensusPayload[H crypto.Hash, A crypto.Address](f func(*Context[H, A], payload.MessageType, any) payload.ConsensusPayload[H, A]) func(config *Config[H, A]) {
	return func(cfg *Config[H, A]) {
		cfg.NewConsensusPayload = f
	}
}

// WithNewPrepareRequest sets NewPrepareRequest.
func WithNewPrepareRequest[H crypto.Hash, A crypto.Address](f func() payload.PrepareRequest[H, A]) func(config *Config[H, A]) {
	return func(cfg *Config[H, A]) {
		cfg.NewPrepareRequest = f
	}
}

// WithNewPrepareResponse sets NewPrepareResponse.
func WithNewPrepareResponse[H crypto.Hash, A crypto.Address](f func() payload.PrepareResponse[H]) func(config *Config[H, A]) {
	return func(cfg *Config[H, A]) {
		cfg.NewPrepareResponse = f
	}
}

// WithNewChangeView sets NewChangeView.
func WithNewChangeView[H crypto.Hash, A crypto.Address](f func() payload.ChangeView) func(config *Config[H, A]) {
	return func(cfg *Config[H, A]) {
		cfg.NewChangeView = f
	}
}

// WithNewCommit sets NewCommit.
func WithNewCommit[H crypto.Hash, A crypto.Address](f func() payload.Commit) func(config *Config[H, A]) {
	return func(cfg *Config[H, A]) {
		cfg.NewCommit = f
	}
}

// WithNewRecoveryRequest sets NewRecoveryRequest.
func WithNewRecoveryRequest[H crypto.Hash, A crypto.Address](f func() payload.RecoveryRequest) func(config *Config[H, A]) {
	return func(cfg *Config[H, A]) {
		cfg.NewRecoveryRequest = f
	}
}

// WithNewRecoveryMessage sets NewRecoveryMessage.
func WithNewRecoveryMessage[H crypto.Hash, A crypto.Address](f func() payload.RecoveryMessage[H, A]) func(config *Config[H, A]) {
	return func(cfg *Config[H, A]) {
		cfg.NewRecoveryMessage = f
	}
}

// WithVerifyPrepareRequest sets VerifyPrepareRequest.
func WithVerifyPrepareRequest[H crypto.Hash, A crypto.Address](f func(payload.ConsensusPayload[H, A]) error) func(config *Config[H, A]) {
	return func(cfg *Config[H, A]) {
		cfg.VerifyPrepareRequest = f
	}
}

// WithVerifyPrepareResponse sets VerifyPrepareResponse.
func WithVerifyPrepareResponse[H crypto.Hash, A crypto.Address](f func(payload.ConsensusPayload[H, A]) error) func(config *Config[H, A]) {
	return func(cfg *Config[H, A]) {
		cfg.VerifyPrepareResponse = f
	}
}
