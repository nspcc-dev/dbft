package payload

import (
	"bytes"
	"crypto/rand"
	"encoding/gob"
	"testing"

	"github.com/nspcc-dev/dbft"
	"github.com/nspcc-dev/dbft/internal/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPayload_EncodeDecode(t *testing.T) {
	m := NewConsensusPayload().(*Payload)
	m.SetValidatorIndex(10)
	m.SetHeight(77)
	m.SetPrevHash(crypto.Uint256{1})
	m.SetVersion(8)
	m.SetViewNumber(3)

	t.Run("PrepareRequest", func(t *testing.T) {
		m.SetType(dbft.PrepareRequestType)
		m.SetPayload(&prepareRequest{
			nonce:     123,
			timestamp: 345,
			transactionHashes: []crypto.Uint256{
				{1, 2, 3},
				{5, 6, 7},
			},
		})

		testEncodeDecode(t, m, new(Payload))
		testMarshalUnmarshal(t, m, new(Payload))
	})

	t.Run("PrepareResponse", func(t *testing.T) {
		m.SetType(dbft.PrepareResponseType)
		m.SetPayload(&prepareResponse{
			preparationHash: crypto.Uint256{3},
		})

		testEncodeDecode(t, m, new(Payload))
		testMarshalUnmarshal(t, m, new(Payload))
	})

	t.Run("Commit", func(t *testing.T) {
		m.SetType(dbft.CommitType)
		var cc commit
		fillRandom(t, cc.signature[:])
		m.SetPayload(&cc)

		testEncodeDecode(t, m, new(Payload))
		testMarshalUnmarshal(t, m, new(Payload))
	})

	t.Run("ChangeView", func(t *testing.T) {
		m.SetType(dbft.ChangeViewType)
		m.SetPayload(&changeView{
			timestamp:     12345,
			newViewNumber: 4,
		})

		testEncodeDecode(t, m, new(Payload))
		testMarshalUnmarshal(t, m, new(Payload))
	})

	t.Run("RecoveryMessage", func(t *testing.T) {
		m.SetType(dbft.RecoveryMessageType)
		m.SetPayload(&recoveryMessage{
			changeViewPayloads: []changeViewCompact{
				{
					Timestamp:          123,
					ValidatorIndex:     1,
					OriginalViewNumber: 3,
				},
			},
			commitPayloads: []commitCompact{},
			preparationPayloads: []preparationCompact{
				1: {ValidatorIndex: 1},
				3: {ValidatorIndex: 3},
				4: {ValidatorIndex: 4},
			},
			prepareRequest: &prepareRequest{
				nonce:     123,
				timestamp: 345,
				transactionHashes: []crypto.Uint256{
					{1, 2, 3},
					{5, 6, 7},
				},
			},
		})

		testEncodeDecode(t, m, new(Payload))
		testMarshalUnmarshal(t, m, new(Payload))
	})

	t.Run("RecoveryRequest", func(t *testing.T) {
		m.SetType(dbft.RecoveryRequestType)
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
	m.SetPrevHash(crypto.Uint256{1})
	m.SetVersion(8)
	m.SetViewNumber(3)
	m.SetPayload(&recoveryMessage{})

	validators := make([]dbft.PublicKey, 1)
	_, validators[0] = crypto.Generate(rand.Reader)

	rec := m.GetRecoveryMessage()
	require.NotNil(t, rec)

	var p dbft.ConsensusPayload[crypto.Uint256, crypto.Uint160]
	require.NotPanics(t, func() { p = rec.GetPrepareRequest(p, validators, 0) })
	require.Nil(t, p)

	var ps []dbft.ConsensusPayload[crypto.Uint256, crypto.Uint160]
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
			ValidatorIndex:     10,
			OriginalViewNumber: 31,
			Timestamp:          98765,
		}

		testEncodeDecode(t, p, new(changeViewCompact))
	})

	t.Run("Preparation", func(t *testing.T) {
		p := &preparationCompact{
			ValidatorIndex: 10,
		}

		testEncodeDecode(t, p, new(preparationCompact))
	})

	t.Run("Commit", func(t *testing.T) {
		p := &commitCompact{
			ValidatorIndex: 10,
			ViewNumber:     77,
		}
		fillRandom(t, p.Signature[:])

		testEncodeDecode(t, p, new(commitCompact))
	})
}

func TestPayload_Setters(t *testing.T) {
	t.Run("ChangeView", func(t *testing.T) {
		cv := NewChangeView()

		cv.SetTimestamp(secToNanoSec(1234))
		assert.EqualValues(t, secToNanoSec(1234), cv.Timestamp())

		cv.SetNewViewNumber(4)
		assert.EqualValues(t, 4, cv.NewViewNumber())
	})

	t.Run("RecoveryRequest", func(t *testing.T) {
		r := NewRecoveryRequest()

		r.SetTimestamp(secToNanoSec(321))
		require.EqualValues(t, secToNanoSec(321), r.Timestamp())
	})

	t.Run("RecoveryMessage", func(t *testing.T) {
		r := NewRecoveryMessage()

		r.SetPreparationHash(&crypto.Uint256{1, 2, 3})
		require.Equal(t, &crypto.Uint256{1, 2, 3}, r.PreparationHash())
	})
}

func TestMessageType_String(t *testing.T) {
	require.Equal(t, "ChangeView", dbft.ChangeViewType.String())
	require.Equal(t, "PrepareRequest", dbft.PrepareRequestType.String())
	require.Equal(t, "PrepareResponse", dbft.PrepareResponseType.String())
	require.Equal(t, "Commit", dbft.CommitType.String())
	require.Equal(t, "RecoveryRequest", dbft.RecoveryRequestType.String())
	require.Equal(t, "RecoveryMessage", dbft.RecoveryMessageType.String())
}

func testEncodeDecode(t *testing.T, expected, actual Serializable) {
	var buf bytes.Buffer
	w := gob.NewEncoder(&buf)
	err := expected.EncodeBinary(w)
	require.NoError(t, err)

	b := buf.Bytes()
	r := gob.NewDecoder(bytes.NewReader(b))

	err = actual.DecodeBinary(r)
	require.NoError(t, err)
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