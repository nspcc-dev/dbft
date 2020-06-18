package dbft

import (
	"bytes"
	"errors"
	"time"

	"github.com/nspcc-dev/dbft/block"
	"github.com/nspcc-dev/dbft/crypto"
	"github.com/nspcc-dev/dbft/payload"
	"github.com/nspcc-dev/dbft/timer"
	"github.com/nspcc-dev/neo-go/pkg/util"
	"go.uber.org/zap"
)

// Config contains initialization and working parameters for dBFT.
type Config struct {
	// Logger
	Logger *zap.Logger
	// Timer
	Timer timer.Timer
	// SecondsPerBlock is the number of seconds that
	// need to pass before another block will be accepted.
	SecondsPerBlock time.Duration
	// GetKeyPair returns an index of the node in the list of validators
	// together with it's key pair.
	GetKeyPair func([]crypto.PublicKey) (int, crypto.PrivateKey, crypto.PublicKey)
	// NewBlockFromContext should allocate, fill from Context and return new block.Block.
	NewBlockFromContext func(ctx *Context) block.Block
	// RequestTx is a callback which is called when transaction contained
	// in current block can't be found in memory pool.
	RequestTx func(h ...util.Uint256)
	// GetTx returns a transaction from memory pool.
	GetTx func(h util.Uint256) block.Transaction
	// GetVerified returns a slice of verified transactions
	// to be proposed in a new block.
	GetVerified func() []block.Transaction
	// VerifyBlock verifies if block is valid.
	VerifyBlock func(b block.Block) bool
	// Broadcast should broadcast payload m to the consensus nodes.
	Broadcast func(m payload.ConsensusPayload)
	// ProcessBlock is called every time new block is accepted.
	ProcessBlock func(b block.Block)
	// GetBlock should return block with hash.
	GetBlock func(h util.Uint256) block.Block
	// WatchOnly tells if a node should only watch.
	WatchOnly func() bool
	// CurrentHeight returns index of the last accepted block.
	CurrentHeight func() uint32
	// CurrentBlockHash returns hash of the last accepted block.
	CurrentBlockHash func() util.Uint256
	// GetValidators returns list of the validators.
	// When called with a transaction list it must return
	// list of the validators of the next block.
	// If this function ever returns 0-length slice, dbft will panic.
	GetValidators func(...block.Transaction) []crypto.PublicKey
	// GetConsensusAddress returns hash of the validator list.
	GetConsensusAddress func(...crypto.PublicKey) util.Uint160
	// NewConsensusPayload is a constructor for payload.ConsensusPayload.
	NewConsensusPayload func() payload.ConsensusPayload
	// NewPrepareRequest is a constructor for payload.PrepareRequest.
	NewPrepareRequest func() payload.PrepareRequest
	// NewPrepareResponse is a constructor for payload.PrepareResponse.
	NewPrepareResponse func() payload.PrepareResponse
	// NewChangeView is a constructor for payload.ChangeView.
	NewChangeView func() payload.ChangeView
	// NewCommit is a constructor for payload.Commit.
	NewCommit func() payload.Commit
	// NewRecoveryRequest is a constructor for payload.RecoveryRequest.
	NewRecoveryRequest func() payload.RecoveryRequest
	// NewRecoveryMessage is a constructor for payload.RecoveryMessage.
	NewRecoveryMessage func() payload.RecoveryMessage
	// VerifyPrepareRequest can perform external payload verification and returns true iff it was successful.
	VerifyPrepareRequest func(p payload.ConsensusPayload) error
}

const defaultSecondsPerBlock = time.Second * 15

// Option is a generic options type. It can modify config in any way it wants.
type Option = func(*Config)

func defaultConfig() *Config {
	// fields which are set to nil must be provided from client
	return &Config{
		Logger:              zap.NewNop(),
		Timer:               timer.New(),
		SecondsPerBlock:     defaultSecondsPerBlock,
		GetKeyPair:          nil,
		NewBlockFromContext: NewBlockFromContext,
		RequestTx:           func(h ...util.Uint256) {},
		GetTx:               func(h util.Uint256) block.Transaction { return nil },
		GetVerified:         func() []block.Transaction { return make([]block.Transaction, 0) },
		VerifyBlock:         func(b block.Block) bool { return true },
		Broadcast:           func(m payload.ConsensusPayload) {},
		ProcessBlock:        func(b block.Block) {},
		GetBlock:            func(h util.Uint256) block.Block { return nil },
		WatchOnly:           func() bool { return false },
		CurrentHeight:       nil,
		CurrentBlockHash:    nil,
		GetValidators:       nil,
		GetConsensusAddress: func(...crypto.PublicKey) util.Uint160 { return util.Uint160{} },
		NewConsensusPayload: payload.NewConsensusPayload,
		NewPrepareRequest:   payload.NewPrepareRequest,
		NewPrepareResponse:  payload.NewPrepareResponse,
		NewChangeView:       payload.NewChangeView,
		NewCommit:           payload.NewCommit,
		NewRecoveryRequest:  payload.NewRecoveryRequest,
		NewRecoveryMessage:  payload.NewRecoveryMessage,

		VerifyPrepareRequest: func(payload.ConsensusPayload) error { return nil },
	}
}

func checkConfig(cfg *Config) error {
	if cfg.GetKeyPair == nil {
		return errors.New("private key is nil")
	} else if cfg.CurrentHeight == nil {
		return errors.New("CurrentHeight is nil")
	} else if cfg.CurrentBlockHash == nil {
		return errors.New("CurrentBlockHash is nil")
	} else if cfg.GetValidators == nil {
		return errors.New("GetValidators is nil")
	}

	return nil
}

// WithKeyPair sets GetKeyPair to a function returning default key pair
// if it is present in a list of validators.
func WithKeyPair(priv crypto.PrivateKey, pub crypto.PublicKey) Option {
	myPub, err := pub.MarshalBinary()
	if err != nil {
		return nil
	}

	return func(cfg *Config) {
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
func WithGetKeyPair(f func([]crypto.PublicKey) (int, crypto.PrivateKey, crypto.PublicKey)) Option {
	return func(cfg *Config) {
		cfg.GetKeyPair = f
	}
}

// WithLogger sets Logger.
func WithLogger(log *zap.Logger) Option {
	return func(cfg *Config) {
		cfg.Logger = log
	}
}

// WithTimer sets Timer.
func WithTimer(t timer.Timer) Option {
	return func(cfg *Config) {
		cfg.Timer = t
	}
}

// WithSecondsPerBlock sets SecondsPerBlock.
func WithSecondsPerBlock(d time.Duration) Option {
	return func(cfg *Config) {
		cfg.SecondsPerBlock = d
	}
}

// WithNewBlockFromContext sets NewBlockFromContext.
func WithNewBlockFromContext(f func(ctx *Context) block.Block) Option {
	return func(cfg *Config) {
		cfg.NewBlockFromContext = f
	}
}

// WithRequestTx sets RequestTx.
func WithRequestTx(f func(h ...util.Uint256)) Option {
	return func(cfg *Config) {
		cfg.RequestTx = f
	}
}

// WithGetTx sets GetTx.
func WithGetTx(f func(h util.Uint256) block.Transaction) Option {
	return func(cfg *Config) {
		cfg.GetTx = f
	}
}

// WithGetVerified sets GetVerified.
func WithGetVerified(f func() []block.Transaction) Option {
	return func(cfg *Config) {
		cfg.GetVerified = f
	}
}

// WithVerifyBlock sets VerifyBlock.
func WithVerifyBlock(f func(b block.Block) bool) Option {
	return func(cfg *Config) {
		cfg.VerifyBlock = f
	}
}

// WithBroadcast sets Broadcast.
func WithBroadcast(f func(m payload.ConsensusPayload)) Option {
	return func(cfg *Config) {
		cfg.Broadcast = f
	}
}

// WithProcessBlock sets ProcessBlock.
func WithProcessBlock(f func(b block.Block)) Option {
	return func(cfg *Config) {
		cfg.ProcessBlock = f
	}
}

// WithGetBlock sets GetBlock.
func WithGetBlock(f func(h util.Uint256) block.Block) Option {
	return func(cfg *Config) {
		cfg.GetBlock = f
	}
}

// WithWatchOnly sets WatchOnly.
func WithWatchOnly(f func() bool) Option {
	return func(cfg *Config) {
		cfg.WatchOnly = f
	}
}

// WithCurrentHeight sets CurrentHeight.
func WithCurrentHeight(f func() uint32) Option {
	return func(cfg *Config) {
		cfg.CurrentHeight = f
	}
}

// WithCurrentBlockHash sets CurrentBlockHash.
func WithCurrentBlockHash(f func() util.Uint256) Option {
	return func(cfg *Config) {
		cfg.CurrentBlockHash = f
	}
}

// WithGetValidators sets GetValidators.
func WithGetValidators(f func(...block.Transaction) []crypto.PublicKey) Option {
	return func(cfg *Config) {
		cfg.GetValidators = f
	}
}

// WithGetConsensusAddress sets GetConsensusAddress.
func WithGetConsensusAddress(f func(keys ...crypto.PublicKey) util.Uint160) Option {
	return func(cfg *Config) {
		cfg.GetConsensusAddress = f
	}
}

// WithNewConsensusPayload sets NewConsensusPayload.
func WithNewConsensusPayload(f func() payload.ConsensusPayload) Option {
	return func(cfg *Config) {
		cfg.NewConsensusPayload = f
	}
}

// WithNewPrepareRequest sets NewPrepareRequest.
func WithNewPrepareRequest(f func() payload.PrepareRequest) Option {
	return func(cfg *Config) {
		cfg.NewPrepareRequest = f
	}
}

// WithNewPrepareResponse sets NewPrepareResponse.
func WithNewPrepareResponse(f func() payload.PrepareResponse) Option {
	return func(cfg *Config) {
		cfg.NewPrepareResponse = f
	}
}

// WithNewChangeView sets NewChangeView.
func WithNewChangeView(f func() payload.ChangeView) Option {
	return func(cfg *Config) {
		cfg.NewChangeView = f
	}
}

// WithNewCommit sets NewCommit.
func WithNewCommit(f func() payload.Commit) Option {
	return func(cfg *Config) {
		cfg.NewCommit = f
	}
}

// WithNewRecoveryRequest sets NewRecoveryRequest.
func WithNewRecoveryRequest(f func() payload.RecoveryRequest) Option {
	return func(cfg *Config) {
		cfg.NewRecoveryRequest = f
	}
}

// WithNewRecoveryMessage sets NewRecoveryMessage.
func WithNewRecoveryMessage(f func() payload.RecoveryMessage) Option {
	return func(cfg *Config) {
		cfg.NewRecoveryMessage = f
	}
}

// WithVerifyPrepareRequest sets VerifyPrepareRequest.
func WithVerifyPrepareRequest(f func(payload.ConsensusPayload) error) Option {
	return func(cfg *Config) {
		cfg.VerifyPrepareRequest = f
	}
}
