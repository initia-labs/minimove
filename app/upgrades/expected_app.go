package upgrades

import (
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"

	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"

	movekeeper "github.com/initia-labs/initia/x/move/keeper"

	forwardingkeeper "github.com/noble-assets/forwarding/v2/keeper"
)

type MinitiaApp interface {
	GetAccountKeeper() *authkeeper.AccountKeeper
	GetMoveKeeper() *movekeeper.Keeper
	GetUpgradeKeeper() *upgradekeeper.Keeper
	GetForwardingKeeper() *forwardingkeeper.Keeper

	GetConfigurator() module.Configurator
	GetModuleManager() *module.Manager
}
