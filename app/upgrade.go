package app

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	movetypes "github.com/initia-labs/initia/x/move/types"
	vmapi "github.com/initia-labs/movevm/api"
	vmtypes "github.com/initia-labs/movevm/types"
)

const upgradeName = "0.2.2"

// RegisterUpgradeHandlers returns upgrade handlers
func (app *MinitiaApp) RegisterUpgradeHandlers(cfg module.Configurator) {
	app.UpgradeKeeper.SetUpgradeHandler(
		upgradeName,
		func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			codeMetadataStore, err := vmapi.ParseStructTag("0x1::code::MetadataStore")
			if err != nil {
				return nil, err
			}

			metadataStore, err := app.MoveKeeper.GetResourceBytes(ctx, vmtypes.StdAddress, codeMetadataStore)
			if err != nil {
				return nil, err
			}

			tableHandle, err := movetypes.ReadTableHandleFromTable(metadataStore)
			if err != nil {
				return nil, err
			}

			bz, err := vmtypes.SerializeString("0x1::oracle")
			if err != nil {
				return nil, err
			}

			tableEntry, err := app.MoveKeeper.GetTableEntryBytes(ctx, tableHandle, bz)
			if err != nil {
				return nil, err
			}

			tableEntry.ValueBytes[0] = 0
			err = app.MoveKeeper.SetTableEntry(ctx, tableEntry)
			if err != nil {
				return nil, err
			}

			return app.ModuleManager.RunMigrations(ctx, app.configurator, vm)
		},
	)
}
