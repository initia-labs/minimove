package v1_1_0

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/initia-labs/minimove/app/upgrades"

	movetypes "github.com/initia-labs/initia/x/move/types"

	vmprecom "github.com/initia-labs/movevm/precompile"
	vmtypes "github.com/initia-labs/movevm/types"
)

const upgradeName = "v1.1.0-alpha.0"

// RegisterUpgradeHandlers returns upgrade handlers
func RegisterUpgradeHandlers(app upgrades.MinitiaApp) {
	app.GetUpgradeKeeper().SetUpgradeHandler(
		upgradeName,
		func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			moduleBytesArray, err := vmprecom.ReadMinlib()
			if err != nil {
				return nil, err
			}

			var modules []vmtypes.Module
			for _, module := range moduleBytesArray {
				modules = append(modules, vmtypes.NewModule(module))
			}

			err = app.GetMoveKeeper().PublishModuleBundle(ctx, vmtypes.StdAddress, vmtypes.NewModuleBundle(modules...), movetypes.UpgradePolicy_COMPATIBLE)
			if err != nil {
				return nil, err
			}

			return vm, nil
		},
	)
}
