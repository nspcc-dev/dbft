package crypto

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVerifySignature(t *testing.T) {
	const dataSize = 1000

	priv, pub := Generate(rand.Reader)
	data := make([]byte, dataSize)
	_, err := rand.Reader.Read(data)
	require.NoError(t, err)

	sign, err := priv.Sign(data)
	require.NoError(t, err)
	require.Equal(t, 64, len(sign))

	err = pub.(*ECDSAPub).Verify(data, sign)
	require.NoError(t, err)
}

func TestGenerateWith(t *testing.T) {
	priv, pub := GenerateWith(defaultSuite, rand.Reader)
	require.NotNil(t, priv)
	require.NotNil(t, pub)

	priv, pub = GenerateWith(suiteType(0xFF), rand.Reader)
	require.Nil(t, priv)
	require.Nil(t, pub)
}
