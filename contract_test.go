package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContractSource(t *testing.T) {
	require.Equal(t, "1.17.8", GetReference().Version)
}
