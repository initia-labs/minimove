package v1_2_0

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/types/module"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	"github.com/initia-labs/minimove/app/upgrades"

	movetypes "github.com/initia-labs/initia/x/move/types"

	vmprecom "github.com/initia-labs/movevm/precompile"
	vmtypes "github.com/initia-labs/movevm/types"
)

const upgradeName = "v1.2.0"

// RegisterUpgradeHandlers returns upgrade handlers
func RegisterUpgradeHandlers(app upgrades.MinitiaApp) {
	// apply store upgrade only if this upgrade is scheduled at a height
	if upgradeInfo, err := app.GetUpgradeKeeper().ReadUpgradeInfoFromDisk(); err == nil {
		if upgradeInfo.Name == upgradeName && !app.GetUpgradeKeeper().IsSkipHeight(upgradeInfo.Height) {
			storeUpgrades := storetypes.StoreUpgrades{
				Deleted: []string{"auction"},
			}

			app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
		}
	}

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

			// bind the opinit IBC port for opchild module
			bound, err := app.GetOPChildKeeper().IsBound(ctx, opchildtypes.PortID)
			if err != nil {
				return nil, err
			}
			if !bound {
				if err := app.GetOPChildKeeper().BindPort(ctx, opchildtypes.PortID); err != nil {
					return nil, err
				}
			}

			return vm, nil
		},
	)
}
