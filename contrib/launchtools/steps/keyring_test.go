package steps

import (
	launchertypes "github.com/initia-labs/minimove/contrib/launcher"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInitializeKeyring(t *testing.T) {
	launcher := launchertypes.NewLauncherForTesting(t.TempDir())
	assert.Equal(t, "memory", launcher.ClientContext.Keyring.Backend())

	assert.NoError(t, InitializeKeyring(launchertypes.TestInput())(launcher))

	accRecord, err := launcher.ClientContext.Keyring.Key("Relayer")
	assert.NoError(t, err)
	assert.NotNil(t, accRecord)

}
