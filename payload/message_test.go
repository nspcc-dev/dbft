package payload

import (
	"crypto/rand"
	"testing"

	"github.com/CityOfZion/neo-go/pkg/io"
	"github.com/CityOfZion/neo-go/pkg/util"
	"github.com/stretchr/testify/require"
)

func TestPayload_EncodeDecode(t *testing.T) {
	m := NewConsensusPayload().(*consensusPayload)
	m.SetValidatorIndex(10)
	m.SetHeight(77)
	m.SetPrevHash(util.Uint256{1})
	m.SetVersion(8)
	m.SetViewNumber(3)

	t.Run("PrepareRequest", func(t *testing.T) {
		m.SetType(PrepareRequestType)
		m.SetPayload(&prepareRequest{
			nonce:     123,
			timestamp: 345,
			transactionHashes: []util.Uint256{
				{1, 2, 3},
				{5, 6, 7},
			},
		})

		testEncodeDecode(t, m, new(consensusPayload))
	})

	t.Run("PrepareResponse", func(t *testing.T) {
		m.SetType(PrepareResponseType)
		m.SetPayload(&prepareResponse{
			preparationHash: util.Uint256{3},
		})

		testEncodeDecode(t, m, new(consensusPayload))
	})

	t.Run("Commit", func(t *testing.T) {
		m.SetType(CommitType)
		var cc commit
		fillRandom(t, cc.signature[:])
		m.SetPayload(&cc)

		testEncodeDecode(t, m, new(consensusPayload))
	})

	t.Run("ChangeView", func(t *testing.T) {
		m.SetType(ChangeViewType)
		m.SetPayload(&changeView{
			timestamp:     12345,
			newViewNumber: 4,
		})

		testEncodeDecode(t, m, new(consensusPayload))
	})

	t.Run("RecoveryMessage", func(t *testing.T) {
		m.SetType(RecoveryMessageType)
		m.SetPayload(&recoveryMessage{
			changeViewPayloads: []changeViewCompact{
				{
					timestamp:          123,
					validatorIndex:     1,
					originalViewNumber: 3,
				},
			},
			commitPayloads: []commitCompact{},
			preparationPayloads: []preparationCompact{
				1: {validatorIndex: 1},
				3: {validatorIndex: 3},
				4: {validatorIndex: 4},
			},
			prepareRequest: &prepareRequest{
				nonce:     123,
				timestamp: 345,
				transactionHashes: []util.Uint256{
					{1, 2, 3},
					{5, 6, 7},
				},
			},
		})

		testEncodeDecode(t, m, new(consensusPayload))
	})

	t.Run("RecoveryRequest", func(t *testing.T) {
		m.SetType(RecoveryRequestType)
		m.SetPayload(&recoveryRequest{
			timestamp: 17334,
		})

		testEncodeDecode(t, m, new(consensusPayload))
	})
}

func TestCompact_EncodeDecode(t *testing.T) {
	t.Run("ChangeView", func(t *testing.T) {
		p := &changeViewCompact{
			validatorIndex:     10,
			originalViewNumber: 31,
			timestamp:          98765,
		}

		testEncodeDecode(t, p, new(changeViewCompact))
	})

	t.Run("Preparation", func(t *testing.T) {
		p := &preparationCompact{
			validatorIndex: 10,
		}

		testEncodeDecode(t, p, new(preparationCompact))
	})

	t.Run("Commit", func(t *testing.T) {
		p := &commitCompact{
			validatorIndex: 10,
			viewNumber:     77,
		}
		fillRandom(t, p.signature[:])

		testEncodeDecode(t, p, new(commitCompact))
	})
}

func testEncodeDecode(t *testing.T, expected, actual io.Serializable) {
	w := io.NewBufBinWriter()
	expected.EncodeBinary(w.BinWriter)
	require.NoError(t, w.Err)

	buf := w.Bytes()
	r := io.NewBinReaderFromBuf(buf)

	actual.DecodeBinary(r)
	require.NoError(t, r.Err)
	require.Equal(t, expected, actual)
}

func fillRandom(t *testing.T, arr []byte) {
	_, err := rand.Read(arr)
	require.NoError(t, err)
}
