package app

import (
	"golang.org/x/exp/maps"

	// cosmos mod modules
	"cosmossdk.io/x/feegrant"
	feegrantmodule "cosmossdk.io/x/feegrant/module"
	"cosmossdk.io/x/upgrade"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	// cosmos modules
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/consensus"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	genutil "github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/group"
	groupmodule "github.com/cosmos/cosmos-sdk/x/group/module"

	// ibc modules
	packetforward "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v10/packetforward"
	packetforwardtypes "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v10/packetforward/types"
	ica "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	ibctransfer "github.com/cosmos/ibc-go/v10/modules/apps/transfer"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v10/modules/core"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v10/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"

	// initia ibc modules
	ibchooks "github.com/initia-labs/initia/x/ibc-hooks"
	ibchookstypes "github.com/initia-labs/initia/x/ibc-hooks/types"
	ibcnfttransfer "github.com/initia-labs/initia/x/ibc/nft-transfer"
	ibcnfttransfertypes "github.com/initia-labs/initia/x/ibc/nft-transfer/types"
	icaauth "github.com/initia-labs/initia/x/intertx"
	icaauthtypes "github.com/initia-labs/initia/x/intertx/types"

	// initia modules
	authzmodule "github.com/initia-labs/initia/x/authz/module"
	"github.com/initia-labs/initia/x/bank"
	"github.com/initia-labs/initia/x/move"
	movetypes "github.com/initia-labs/initia/x/move/types"

	// opinit modules
	opchild "github.com/initia-labs/OPinit/x/opchild"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"

	// skip-mev modules
	marketmap "github.com/skip-mev/connect/v2/x/marketmap"
	marketmaptypes "github.com/skip-mev/connect/v2/x/marketmap/types"
	"github.com/skip-mev/connect/v2/x/oracle"
	oracletypes "github.com/skip-mev/connect/v2/x/oracle/types"

	// noble forwarding keeper
	forwarding "github.com/noble-assets/forwarding/v2"
	forwardingtypes "github.com/noble-assets/forwarding/v2/types"
)

// module account permissions
var maccPerms = map[string][]string{
	authtypes.FeeCollectorName:      nil,
	icatypes.ModuleName:             nil,
	ibctransfertypes.ModuleName:     {authtypes.Minter, authtypes.Burner},
	movetypes.MoveStakingModuleName: nil,
	opchildtypes.ModuleName:         {authtypes.Minter, authtypes.Burner},

	// connect oracle permissions
	oracletypes.ModuleName: nil,

	// this is only for testing
	authtypes.Minter: {authtypes.Minter},
}

func appModules(
	app *MinitiaApp,
) []module.AppModule { //nolint:staticcheck
	return []module.AppModule{ //nolint:staticcheck
		auth.NewAppModule(app.appCodec, *app.AccountKeeper, nil, nil),
		bank.NewAppModule(app.appCodec, *app.BankKeeper, app.AccountKeeper),
		opchild.NewAppModule(app.appCodec, app.OPChildKeeper),
		feegrantmodule.NewAppModule(app.appCodec, app.AccountKeeper, app.BankKeeper, *app.FeeGrantKeeper, app.interfaceRegistry),
		upgrade.NewAppModule(app.UpgradeKeeper, app.ac),
		authzmodule.NewAppModule(app.appCodec, *app.AuthzKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),
		groupmodule.NewAppModule(app.appCodec, *app.GroupKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),
		consensus.NewAppModule(app.appCodec, *app.ConsensusParamsKeeper),
		move.NewAppModule(app.appCodec, *app.MoveKeeper, app.vc, maps.Keys(maccPerms)),
		// ibc modules
		ibc.NewAppModule(app.IBCKeeper),
		ibctransfer.NewAppModule(*app.TransferKeeper),
		ibcnfttransfer.NewAppModule(app.appCodec, *app.NftTransferKeeper),
		ica.NewAppModule(app.ICAControllerKeeper, app.ICAHostKeeper),
		icaauth.NewAppModule(app.appCodec, *app.ICAAuthKeeper),
		ibctm.NewAppModule(*app.TMLightClientModule),
		solomachine.NewAppModule(*app.SMLightClientModule),
		packetforward.NewAppModule(app.PacketForwardKeeper, nil),
		ibchooks.NewAppModule(app.appCodec, *app.IBCHooksKeeper),
		forwarding.NewAppModule(app.ForwardingKeeper),
		// connect modules
		oracle.NewAppModule(app.appCodec, *app.OracleKeeper),
		marketmap.NewAppModule(app.appCodec, app.MarketMapKeeper),
	}
}

// ModuleBasics defines the module BasicManager that is in charge of setting up basic,
// non-dependant module elements, such as codec registration
// and genesis verification.
func newBasicManagerFromManager(app *MinitiaApp) module.BasicManager {
	basicManager := module.NewBasicManagerFromManager(
		app.ModuleManager,
		map[string]module.AppModuleBasic{
			genutiltypes.ModuleName: genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
		})
	basicManager.RegisterLegacyAminoCodec(app.legacyAmino)
	basicManager.RegisterInterfaces(app.interfaceRegistry)
	return basicManager
}

/*
orderBeginBlockers tells the app's module manager how to set the order of
BeginBlockers, which are run at the beginning of every block.
*/
func orderBeginBlockers() []string {
	return []string{
		opchildtypes.ModuleName,
		authz.ModuleName,
		movetypes.ModuleName,
		ibcexported.ModuleName,
		oracletypes.ModuleName,
		marketmaptypes.ModuleName,
	}
}

func orderEndBlockers() []string {
	return []string{
		opchildtypes.ModuleName,
		authz.ModuleName,
		feegrant.ModuleName,
		group.ModuleName,
		oracletypes.ModuleName,
		marketmaptypes.ModuleName,
		forwardingtypes.ModuleName,
	}
}

/*
NOTE: The genutils module must occur after staking so that pools are
properly initialized with tokens from genesis accounts.
NOTE: The genutils module must also occur after auth so that it can access the params from auth.
*/
func orderInitBlockers() []string {
	return []string{
		authtypes.ModuleName, movetypes.ModuleName, banktypes.ModuleName,
		opchildtypes.ModuleName, genutiltypes.ModuleName, authz.ModuleName, group.ModuleName,
		upgradetypes.ModuleName, feegrant.ModuleName, consensusparamtypes.ModuleName, ibcexported.ModuleName,
		ibctransfertypes.ModuleName, ibcnfttransfertypes.ModuleName, icatypes.ModuleName, icaauthtypes.ModuleName,
		oracletypes.ModuleName,
		marketmaptypes.ModuleName, packetforwardtypes.ModuleName, ibchookstypes.ModuleName, forwardingtypes.ModuleName,
	}
}
