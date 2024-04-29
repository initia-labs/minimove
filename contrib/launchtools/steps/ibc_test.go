package steps

import (
	launchertypes "github.com/initia-labs/minimove/contrib/launcher"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEstablishIBCChannels(t *testing.T) {
	testInput := launchertypes.TestInput()
	launcher := launchertypes.NewLauncherForTesting(t.TempDir())

	// IBC depends on keyring, so we need to initialize it first
	assert.NoError(t, InitializeKeyring(testInput)(launcher))

	// then test IBC
	assert.NoError(t, EstablishIBCChannels(testInput)(launcher))
}
