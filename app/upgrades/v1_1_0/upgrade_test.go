package v1_1_0_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	minimoveapp "github.com/initia-labs/minimove/app"
	"github.com/initia-labs/minimove/app/upgrades/v1_1_0"

	vmtypes "github.com/initia-labs/movevm/types"
)

func TestPublishModuleBundle(t *testing.T) {
	app := minimoveapp.SetupWithGenesisAccounts(nil, nil)

	ctx, err := app.CreateQueryContext(app.LastBlockHeight(), false)
	require.NoError(t, err)

	err = v1_1_0.PublishModuleBundle(ctx, app)
	require.NoError(t, err)

	moduleBytes, err := v1_1_0.GetModuleWithName("account.mv")
	require.NoError(t, err)

	module, err := app.MoveKeeper.GetModule(ctx, vmtypes.StdAddress, "account")
	require.NoError(t, err)
	require.Equal(t, moduleBytes, module.RawBytes)

	// check supply_without_sanity_check exists
	_, _, err = app.MoveKeeper.ExecuteViewFunctionJSON(ctx, vmtypes.StdAddress, "fungible_asset", "supply_without_sanity_check", []vmtypes.TypeTag{}, []string{})
	require.Error(t, err, "NUMBER_OF_TYPE_ARGUMENTS_MISMATCH")

	// check balance_without_sanity_check exists
	_, _, err = app.MoveKeeper.ExecuteViewFunctionJSON(ctx, vmtypes.StdAddress, "fungible_asset", "balance_without_sanity_check", []vmtypes.TypeTag{}, []string{})
	require.ErrorContains(t, err, "NUMBER_OF_TYPE_ARGUMENTS_MISMATCH")
}
