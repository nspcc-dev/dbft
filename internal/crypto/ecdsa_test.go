package crypto

import (
	"crypto/rand"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestECDSA_MarshalUnmarshal(t *testing.T) {
	priv, pub := generateECDSA(rand.Reader)
	require.NotNil(t, priv)
	require.NotNil(t, pub)

	data, err := pub.(*ECDSAPub).MarshalBinary()
	require.NoError(t, err)

	pub1 := new(ECDSAPub)
	require.NoError(t, pub1.UnmarshalBinary(data))
	require.Equal(t, pub, pub1)

	require.Error(t, pub1.UnmarshalBinary([]byte{0, 1, 2, 3}))
}

// Do not generate keys with not enough entropy.
func TestECDSA_Generate(t *testing.T) {
	rd := &errorReader{}
	priv, pub := GenerateWith(SuiteECDSA, rd)
	require.Nil(t, priv)
	require.Nil(t, pub)
}

type errorReader struct{}

func (r *errorReader) Read(_ []byte) (int, error) { return 0, errors.New("error on read") }
