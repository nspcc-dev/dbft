package crypto

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// Do not generate keys with not enough entropy.
func TestECDSA_Generate(t *testing.T) {
	rd := &errorReader{}
	priv, pub := GenerateWith(SuiteECDSA, rd)
	require.Nil(t, priv)
	require.Nil(t, pub)
}

type errorReader struct{}

func (r *errorReader) Read(_ []byte) (int, error) { return 0, errors.New("error on read") }
