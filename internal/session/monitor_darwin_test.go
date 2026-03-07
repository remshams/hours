//go:build darwin

package session

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDarwinSessionLockState(t *testing.T) {
	t.Run("locked", func(t *testing.T) {
		locked, err := parseDarwinSessionLockState([]byte(`
<key>CGSSessionScreenIsLocked</key>
<true/>
`))

		require.NoError(t, err)
		assert.True(t, locked)
	})

	t.Run("unlocked", func(t *testing.T) {
		locked, err := parseDarwinSessionLockState([]byte(`
<key>CGSSessionScreenIsLocked</key>
<false/>
`))

		require.NoError(t, err)
		assert.False(t, locked)
	})

	t.Run("missing key", func(t *testing.T) {
		_, err := parseDarwinSessionLockState([]byte(`<key>OtherKey</key><true/>`))

		require.ErrorIs(t, err, errSessionLockStateNotFound)
	})
}
