// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root.

package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLimitsSource(t *testing.T) {
	input, err := validateInput(nil)
	require.NoError(t, err)
	require.Equal(t, []byte("{}"), input)
}
