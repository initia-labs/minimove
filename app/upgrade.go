package app

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/types/module"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
)

const upgradeName = "0.7.1"

// RegisterUpgradeHandlers returns upgrade handlers
func (app *MinitiaApp) RegisterUpgradeHandlers(cfg module.Configurator) {
	app.UpgradeKeeper.SetUpgradeHandler(
		upgradeName,
		func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			params, err := app.OPChildKeeper.GetParams(ctx)
			if err != nil {
				return nil, err
			}

			// set non-zero default values for new params
			if params.HookMaxGas == 0 {
				params.HookMaxGas = opchildtypes.DefaultHookMaxGas
				err = app.OPChildKeeper.SetParams(ctx, params)
				if err != nil {
					return nil, err
				}
			}

			return app.ModuleManager.RunMigrations(ctx, cfg, vm)
		},
	)
}
