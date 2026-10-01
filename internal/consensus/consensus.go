package consensus

import (
	"time"

	"github.com/nspcc-dev/dbft"
	"github.com/nspcc-dev/dbft/internal/crypto"
	"github.com/nspcc-dev/dbft/timer"
	"go.uber.org/zap"
)

func New(logger *zap.Logger, key dbft.PrivateKey, pub dbft.PublicKey,
	getVerified func() []*Tx64,
	broadcast func(dbft.ConsensusPayload[crypto.Uint256, *Tx64]),
	processBlock func(dbft.Block[crypto.Uint256, *Tx64]) error,
	currentHeight func() uint32,
	currentBlockHash func() crypto.Uint256,
	getValidators func(...*Tx64) []dbft.PublicKey,
	verifyPayload func(consensusPayload dbft.ConsensusPayload[crypto.Uint256, *Tx64]) error) (*dbft.DBFT[crypto.Uint256, *Tx64], error) {
	return dbft.New[crypto.Uint256](
		dbft.WithTimer[crypto.Uint256, *Tx64](timer.New()),
		dbft.WithLogger[crypto.Uint256, *Tx64](logger),
		dbft.WithTimePerBlock[crypto.Uint256, *Tx64](func() time.Duration {
			return time.Second * 5
		}),
		dbft.WithGetKeyPair[crypto.Uint256, *Tx64](func(pubs []dbft.PublicKey) (int, dbft.PrivateKey, dbft.PublicKey) {
			for i := range pubs {
				if pub.(*crypto.ECDSAPub).Equals(pubs[i]) {
					return i, key, pub
				}
			}

			return -1, nil, nil
		}),
		dbft.WithGetVerified[crypto.Uint256, *Tx64](getVerified),
		dbft.WithBroadcast[crypto.Uint256, *Tx64](broadcast),
		dbft.WithProcessBlock[crypto.Uint256, *Tx64](processBlock),
		dbft.WithCurrentHeight[crypto.Uint256, *Tx64](currentHeight),
		dbft.WithCurrentBlockHash[crypto.Uint256, *Tx64](currentBlockHash),
		dbft.WithGetValidators[crypto.Uint256, *Tx64](getValidators),
		dbft.WithVerifyPrepareRequest[crypto.Uint256, *Tx64](verifyPayload),
		dbft.WithVerifyPrepareResponse[crypto.Uint256, *Tx64](verifyPayload),
		dbft.WithVerifyCommit[crypto.Uint256, *Tx64](verifyPayload),

		dbft.WithNewBlockFromContext[crypto.Uint256, *Tx64](newBlockFromContext),
		dbft.WithNewConsensusPayload[crypto.Uint256, *Tx64](defaultNewConsensusPayload),
		dbft.WithNewPrepareRequest[crypto.Uint256, *Tx64](NewPrepareRequest),
		dbft.WithNewPrepareResponse[crypto.Uint256, *Tx64](NewPrepareResponse),
		dbft.WithNewChangeView[crypto.Uint256, *Tx64](NewChangeView),
		dbft.WithNewCommit[crypto.Uint256, *Tx64](NewCommit),
		dbft.WithNewRecoveryMessage[crypto.Uint256, *Tx64](func() dbft.RecoveryMessage[crypto.Uint256, *Tx64] {
			return NewRecoveryMessage(nil)
		}),
		dbft.WithNewRecoveryRequest[crypto.Uint256, *Tx64](NewRecoveryRequest),
	)
}

func newBlockFromContext(ctx *dbft.Context[crypto.Uint256, *Tx64]) dbft.Block[crypto.Uint256, *Tx64] {
	if ctx.Transactions == nil {
		return nil
	}
	block := NewBlock(ctx.Timestamp, ctx.BlockIndex, ctx.PrevHash, ctx.Nonce, ctx.Transactions)
	return block
}

// defaultNewConsensusPayload is default function for creating
// consensus payload of specific type.
func defaultNewConsensusPayload(c *dbft.Context[crypto.Uint256, *Tx64], t dbft.MessageType, msg any) dbft.ConsensusPayload[crypto.Uint256, *Tx64] {
	return NewConsensusPayload(t, c.BlockIndex, uint16(c.MyIndex), c.ViewNumber, msg)
}
