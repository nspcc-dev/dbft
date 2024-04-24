package consensus

import (
	"time"

	"github.com/nspcc-dev/dbft"
	"github.com/nspcc-dev/dbft/internal/crypto"
	"github.com/nspcc-dev/dbft/timer"
	"go.uber.org/zap"
)

func New(logger *zap.Logger, key dbft.PrivateKey, pub dbft.PublicKey,
	getTx func(uint256 crypto.Uint256) dbft.Transaction[crypto.Uint256],
	getVerified func() []dbft.Transaction[crypto.Uint256],
	broadcast func(dbft.ConsensusPayload[crypto.Uint256]),
	processBlock func(dbft.Block[crypto.Uint256]),
	currentHeight func() uint32,
	currentBlockHash func() crypto.Uint256,
	getValidators func(...dbft.Transaction[crypto.Uint256]) []dbft.PublicKey,
	verifyPayload func(consensusPayload dbft.ConsensusPayload[crypto.Uint256]) error) (*dbft.DBFT[crypto.Uint256], error) {
	return dbft.New[crypto.Uint256](
		dbft.WithTimer[crypto.Uint256](timer.New()),
		dbft.WithLogger[crypto.Uint256](logger),
		dbft.WithSecondsPerBlock[crypto.Uint256](time.Second*5),
		dbft.WithGetKeyPair[crypto.Uint256](func(pubs []dbft.PublicKey) (int, dbft.PrivateKey, dbft.PublicKey) {
			for i := range pubs {
				if pub.(*crypto.ECDSAPub).Equals(pubs[i]) {
					return i, key, pub
				}
			}

			return -1, nil, nil
		}),
		dbft.WithGetTx[crypto.Uint256](getTx),
		dbft.WithGetVerified[crypto.Uint256](getVerified),
		dbft.WithBroadcast[crypto.Uint256](broadcast),
		dbft.WithProcessBlock[crypto.Uint256](processBlock),
		dbft.WithCurrentHeight[crypto.Uint256](currentHeight),
		dbft.WithCurrentBlockHash[crypto.Uint256](currentBlockHash),
		dbft.WithGetValidators[crypto.Uint256](getValidators),
		dbft.WithVerifyPrepareRequest[crypto.Uint256](verifyPayload),
		dbft.WithVerifyPrepareResponse[crypto.Uint256](verifyPayload),

		dbft.WithNewBlockFromContext[crypto.Uint256](newBlockFromContext),
		dbft.WithNewConsensusPayload[crypto.Uint256](defaultNewConsensusPayload),
		dbft.WithNewPrepareRequest[crypto.Uint256](NewPrepareRequest),
		dbft.WithNewPrepareResponse[crypto.Uint256](NewPrepareResponse),
		dbft.WithNewChangeView[crypto.Uint256](NewChangeView),
		dbft.WithNewCommit[crypto.Uint256](NewCommit),
		dbft.WithNewRecoveryMessage[crypto.Uint256](func() dbft.RecoveryMessage[crypto.Uint256] {
			return NewRecoveryMessage(nil)
		}),
		dbft.WithNewRecoveryRequest[crypto.Uint256](NewRecoveryRequest),
	)
}

func newBlockFromContext(ctx *dbft.Context[crypto.Uint256]) dbft.Block[crypto.Uint256] {
	if ctx.TransactionHashes == nil {
		return nil
	}
	block := NewBlock(ctx.Timestamp, ctx.BlockIndex, ctx.PrevHash, ctx.Nonce, ctx.TransactionHashes)
	return block
}

// defaultNewConsensusPayload is default function for creating
// consensus payload of specific type.
func defaultNewConsensusPayload(c *dbft.Context[crypto.Uint256], t dbft.MessageType, msg any) dbft.ConsensusPayload[crypto.Uint256] {
	return NewConsensusPayload(t, c.BlockIndex, uint16(c.MyIndex), c.ViewNumber, msg)
}
