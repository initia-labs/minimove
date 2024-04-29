package steps

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestInitializeGenesis(t *testing.T) {
	home, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}

	os.MkdirAll(home+"/config", os.ModePerm)
	os.MkdirAll(home+"/data", os.ModePerm)

	defer os.RemoveAll(home)

	assert.NoError(t, InitializeGenesis(launchtools.TestInput())(launchtools.NewLauncherForTesting(home)))

	_, err = os.Stat(home + "/config/genesis.json")
	assert.NoError(t, err)
}
