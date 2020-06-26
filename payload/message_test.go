package payload

import (
	"crypto/rand"
	"testing"

	"github.com/nspcc-dev/dbft/crypto"
	"github.com/nspcc-dev/neo-go/pkg/io"
	"github.com/nspcc-dev/neo-go/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPayload_EncodeDecode(t *testing.T) {
	m := NewConsensusPayload().(*Payload)
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

		testEncodeDecode(t, m, new(Payload))
		testMarshalUnmarshal(t, m, new(Payload))
	})

	t.Run("PrepareResponse", func(t *testing.T) {
		m.SetType(PrepareResponseType)
		m.SetPayload(&prepareResponse{
			preparationHash: util.Uint256{3},
		})

		testEncodeDecode(t, m, new(Payload))
		testMarshalUnmarshal(t, m, new(Payload))
	})

	t.Run("Commit", func(t *testing.T) {
		m.SetType(CommitType)
		var cc commit
		fillRandom(t, cc.signature[:])
		m.SetPayload(&cc)

		testEncodeDecode(t, m, new(Payload))
		testMarshalUnmarshal(t, m, new(Payload))
	})

	t.Run("ChangeView", func(t *testing.T) {
		m.SetType(ChangeViewType)
		m.SetPayload(&changeView{
			timestamp:     12345,
			newViewNumber: 4,
		})

		testEncodeDecode(t, m, new(Payload))
		testMarshalUnmarshal(t, m, new(Payload))
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

		testEncodeDecode(t, m, new(Payload))
		testMarshalUnmarshal(t, m, new(Payload))
	})

	t.Run("RecoveryRequest", func(t *testing.T) {
		m.SetType(RecoveryRequestType)
		m.SetPayload(&recoveryRequest{
			timestamp: 17334,
		})

		testEncodeDecode(t, m, new(Payload))
		testMarshalUnmarshal(t, m, new(Payload))
	})
}

func TestRecoveryMessage_NoPayloads(t *testing.T) {
	m := NewConsensusPayload().(*Payload)
	m.SetValidatorIndex(0)
	m.SetHeight(77)
	m.SetPrevHash(util.Uint256{1})
	m.SetVersion(8)
	m.SetViewNumber(3)
	m.SetPayload(&recoveryMessage{})

	validators := make([]crypto.PublicKey, 1)
	_, validators[0] = crypto.Generate(rand.Reader)

	rec := m.GetRecoveryMessage()
	require.NotNil(t, rec)

	var p ConsensusPayload
	require.NotPanics(t, func() { p = rec.GetPrepareRequest(p, validators, 0) })
	require.Nil(t, p)

	var ps []ConsensusPayload
	require.NotPanics(t, func() { ps = rec.GetPrepareResponses(p, validators) })
	require.Len(t, ps, 0)

	require.NotPanics(t, func() { ps = rec.GetCommits(p, validators) })
	require.Len(t, ps, 0)

	require.NotPanics(t, func() { ps = rec.GetChangeViews(p, validators) })
	require.Len(t, ps, 0)
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

func TestPayload_Setters(t *testing.T) {
	t.Run("ChangeView", func(t *testing.T) {
		cv := NewChangeView()

		cv.SetTimestamp(1234)
		assert.EqualValues(t, 1234, cv.Timestamp())

		cv.SetNewViewNumber(4)
		assert.EqualValues(t, 4, cv.NewViewNumber())
	})

	t.Run("RecoveryRequest", func(t *testing.T) {
		r := NewRecoveryRequest()

		r.SetTimestamp(321)
		require.EqualValues(t, 321, r.Timestamp())
	})

	t.Run("RecoveryMessage", func(t *testing.T) {
		r := NewRecoveryMessage()

		r.SetPreparationHash(&util.Uint256{1, 2, 3})
		require.Equal(t, &util.Uint256{1, 2, 3}, r.PreparationHash())
	})
}

func TestMessageType_String(t *testing.T) {
	require.Equal(t, "ChangeView", ChangeViewType.String())
	require.Equal(t, "PrepareRequest", PrepareRequestType.String())
	require.Equal(t, "PrepareResponse", PrepareResponseType.String())
	require.Equal(t, "Commit", CommitType.String())
	require.Equal(t, "RecoveryRequest", RecoveryRequestType.String())
	require.Equal(t, "RecoveryMessage", RecoveryMessageType.String())
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

func testMarshalUnmarshal(t *testing.T, expected, actual *Payload) {
	data := expected.MarshalUnsigned()
	require.NoError(t, actual.UnmarshalUnsigned(data))
	require.Equal(t, expected.Hash(), actual.Hash())
}

func fillRandom(t *testing.T, arr []byte) {
	_, err := rand.Read(arr)
	require.NoError(t, err)
}
