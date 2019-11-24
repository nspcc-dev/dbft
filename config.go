package dbft

import (
	"context"
	"errors"
	"time"

	"bitbucket.org/nspcc-dev/dbft/block"
	"bitbucket.org/nspcc-dev/dbft/crypto"
	"bitbucket.org/nspcc-dev/dbft/payload"
	"bitbucket.org/nspcc-dev/dbft/timer"
	"github.com/CityOfZion/neo-go/pkg/util"
	"go.uber.org/zap"
)

// Config contains initialization and working parameters for dBFT.
type Config struct {
	// TxPerBlock is the maximum size of a block.
	TxPerBlock int
	// Logger
	Logger *zap.Logger
	// Timer
	Timer timer.Timer
	// SecondsPerBlock is the number of seconds that
	// need to pass before another block will be accepted.
	SecondsPerBlock time.Duration
	// Priv is node private key.
	Priv crypto.PrivateKey
	// Pub is node public key.
	Pub crypto.PublicKey
	// NewBlock should allocate and return new block.Block.
	NewBlock func() block.Block
	// RequestTx is a callback which is called when transaction contained
	// in current block can't be found in memory pool.
	RequestTx func(h ...util.Uint256)
	// GetTx returns a transaction from memory pool.
	GetTx func(h util.Uint256) block.Transaction
	// GetVerified returns a slice of verified transactions
	// to be proposed in a new block.
	GetVerified func(count int) []block.Transaction
	// VerifyBlock verifies if block is valid.
	VerifyBlock func(b block.Block) bool
	// Broadcast should broadcast payload m to the consensus nodes.
	Broadcast func(ctx context.Context, m payload.ConsensusPayload)
	// ProcessBlock is called every time new block is accepted.
	ProcessBlock func(ctx context.Context, b block.Block)
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
}

const (
	defaultMaxBlockSize    = 1000
	defaultSecondsPerBlock = time.Second * 15
)

// Option is a generic options type. It can modify config in any way it wants.
type Option = func(*Config)

func defaultConfig() *Config {
	// fields which are set to nil must be provided from client
	cfg := &Config{
		TxPerBlock:          defaultMaxBlockSize,
		Logger:              zap.NewNop(),
		Timer:               timer.New(),
		SecondsPerBlock:     defaultSecondsPerBlock,
		Priv:                nil,
		Pub:                 nil,
		NewBlock:            block.NewBlock,
		RequestTx:           func(h ...util.Uint256) {},
		GetTx:               func(h util.Uint256) block.Transaction { return nil },
		GetVerified:         func(count int) []block.Transaction { return make([]block.Transaction, 0) },
		VerifyBlock:         func(b block.Block) bool { return true },
		Broadcast:           func(ctx context.Context, m payload.ConsensusPayload) {},
		ProcessBlock:        func(ctx context.Context, b block.Block) {},
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
	}

	return cfg
}

func checkConfig(cfg *Config) error {
	if cfg.Priv == nil {
		return errors.New("private key is nil")
	} else if cfg.Timer == nil {
		return errors.New("timer is nil")
	}

	return nil
}

func WithKeyPair(priv crypto.PrivateKey, pub crypto.PublicKey) Option {
	return func(cfg *Config) {
		cfg.Priv = priv
		cfg.Pub = pub
	}
}

func WithTxPerBlock(n int) Option {
	return func(cfg *Config) {
		cfg.TxPerBlock = n
	}
}

func WithLogger(log *zap.Logger) Option {
	return func(cfg *Config) {
		cfg.Logger = log
	}
}

func WithTimer(t timer.Timer) Option {
	return func(cfg *Config) {
		cfg.Timer = t
	}
}

func WithSecondsPerBlock(d time.Duration) Option {
	return func(cfg *Config) {
		cfg.SecondsPerBlock = d
	}
}

func WithNewBlock(f func() block.Block) Option {
	return func(cfg *Config) {
		cfg.NewBlock = f
	}
}

func WithRequestTx(f func(h ...util.Uint256)) Option {
	return func(cfg *Config) {
		cfg.RequestTx = f
	}
}

func WithGetTx(f func(h util.Uint256) block.Transaction) Option {
	return func(cfg *Config) {
		cfg.GetTx = f
	}
}

func WithGetVerified(f func(count int) []block.Transaction) Option {
	return func(cfg *Config) {
		cfg.GetVerified = f
	}
}

func WithVerifyBlock(f func(b block.Block) bool) Option {
	return func(cfg *Config) {
		cfg.VerifyBlock = f
	}
}

func WithBroadcast(f func(ctx context.Context, m payload.ConsensusPayload)) Option {
	return func(cfg *Config) {
		cfg.Broadcast = f
	}
}

func WithProcessBlock(f func(ctx context.Context, b block.Block)) Option {
	return func(cfg *Config) {
		cfg.ProcessBlock = f
	}
}

func WithGetBlock(f func(h util.Uint256) block.Block) Option {
	return func(cfg *Config) {
		cfg.GetBlock = f
	}
}

func WithWatchOnly(f func() bool) Option {
	return func(cfg *Config) {
		cfg.WatchOnly = f
	}
}

func WithCurrentHeight(f func() uint32) Option {
	return func(cfg *Config) {
		cfg.CurrentHeight = f
	}
}

func WithCurrentBlockHash(f func() util.Uint256) Option {
	return func(cfg *Config) {
		cfg.CurrentBlockHash = f
	}
}

func WithGetValidators(f func(...block.Transaction) []crypto.PublicKey) Option {
	return func(cfg *Config) {
		cfg.GetValidators = f
	}
}

func WithGetConsensusAddress(f func(keys ...crypto.PublicKey) util.Uint160) Option {
	return func(cfg *Config) {
		cfg.GetConsensusAddress = f
	}
}

func WithNewConsensusPayload(f func() payload.ConsensusPayload) Option {
	return func(cfg *Config) {
		cfg.NewConsensusPayload = f
	}
}

func WithNewPrepareRequest(f func() payload.PrepareRequest) Option {
	return func(cfg *Config) {
		cfg.NewPrepareRequest = f
	}
}

func WithNewPrepareResponse(f func() payload.PrepareResponse) Option {
	return func(cfg *Config) {
		cfg.NewPrepareResponse = f
	}
}

func WithNewChangeView(f func() payload.ChangeView) Option {
	return func(cfg *Config) {
		cfg.NewChangeView = f
	}
}

func WithNewCommit(f func() payload.Commit) Option {
	return func(cfg *Config) {
		cfg.NewCommit = f
	}
}

func WithNewRecoveryRequest(f func() payload.RecoveryRequest) Option {
	return func(cfg *Config) {
		cfg.NewRecoveryRequest = f
	}
}

func WithNewRecoveryMessage(f func() payload.RecoveryMessage) Option {
	return func(cfg *Config) {
		cfg.NewRecoveryMessage = f
	}
}
