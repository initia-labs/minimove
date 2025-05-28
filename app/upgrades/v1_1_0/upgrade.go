package v1_1_0

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/initia-labs/minimove/app/upgrades"

	movetypes "github.com/initia-labs/initia/x/move/types"

	vmtypes "github.com/initia-labs/movevm/types"
)

const upgradeName = "v1.1.0"

// RegisterUpgradeHandlers returns upgrade handlers
func RegisterUpgradeHandlers(app upgrades.MinitiaApp) {
	app.GetUpgradeKeeper().SetUpgradeHandler(
		upgradeName,
		func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			err := PublishModuleBundle(ctx, app)
			if err != nil {
				return nil, err
			}

			return vm, nil
		},
	)
}

// PublishModuleBundle publishes the module bundle to the movevm expose for testing
func PublishModuleBundle(ctx context.Context, app upgrades.MinitiaApp) error {
	moduleBytesArray, err := GetModuleBytes()
	if err != nil {
		return err
	}

	var modules []vmtypes.Module
	for _, module := range moduleBytesArray {
		modules = append(modules, vmtypes.NewModule(module))
	}

	err = app.GetMoveKeeper().PublishModuleBundle(ctx, vmtypes.StdAddress, vmtypes.NewModuleBundle(modules...), movetypes.UpgradePolicy_COMPATIBLE)
	if err != nil {
		return err
	}

	return nil
}
