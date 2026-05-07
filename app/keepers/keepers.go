package keepers

import (
	"context"
	"os"

	tmos "github.com/cometbft/cometbft/libs/os"

	"cosmossdk.io/core/address"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/feegrant"
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	consensusparamkeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	"github.com/cosmos/cosmos-sdk/x/group"
	groupkeeper "github.com/cosmos/cosmos-sdk/x/group/keeper"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v10/packetforward"
	packetforwardkeeper "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v10/packetforward/keeper"
	packetforwardtypes "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v10/packetforward/types"
	ratelimit "github.com/cosmos/ibc-apps/modules/rate-limiting/v10"
	ratelimitkeeper "github.com/cosmos/ibc-apps/modules/rate-limiting/v10/keeper"
	ratelimittypes "github.com/cosmos/ibc-apps/modules/rate-limiting/v10/types"
	icacontroller "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller"
	icacontrollerkeeper "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/keeper"
	icacontrollertypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"
	icahost "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host"
	icahostkeeper "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/types"
	ibctransfer "github.com/cosmos/ibc-go/v10/modules/apps/transfer"
	ibctransferkeeper "github.com/cosmos/ibc-go/v10/modules/apps/transfer/keeper"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v10/modules/core/keeper"
	solomachine "github.com/cosmos/ibc-go/v10/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"

	appheaderinfo "github.com/initia-labs/initia/app/header_info"
	"github.com/initia-labs/initia/app/ibc/legacyfeeack"
	ibchooks "github.com/initia-labs/initia/x/ibc-hooks"
	ibchookskeeper "github.com/initia-labs/initia/x/ibc-hooks/keeper"
	ibcmovehooks "github.com/initia-labs/initia/x/ibc-hooks/move-hooks"
	ibchookstypes "github.com/initia-labs/initia/x/ibc-hooks/types"
	ibcnfttransfer "github.com/initia-labs/initia/x/ibc/nft-transfer"
	ibcnfttransferkeeper "github.com/initia-labs/initia/x/ibc/nft-transfer/keeper"
	ibcnfttransfertypes "github.com/initia-labs/initia/x/ibc/nft-transfer/types"
	icaauth "github.com/initia-labs/initia/x/intertx"
	icaauthkeeper "github.com/initia-labs/initia/x/intertx/keeper"
	icaauthtypes "github.com/initia-labs/initia/x/intertx/types"

	bankkeeper "github.com/initia-labs/initia/x/bank/keeper"
	moveconfig "github.com/initia-labs/initia/x/move/config"
	movekeeper "github.com/initia-labs/initia/x/move/keeper"
	movetypes "github.com/initia-labs/initia/x/move/types"

	"github.com/initia-labs/OPinit/x/opchild"
	opchildkeeper "github.com/initia-labs/OPinit/x/opchild/keeper"
	opchildmigration "github.com/initia-labs/OPinit/x/opchild/middleware/migration"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"

	marketmapkeeper "github.com/skip-mev/connect/v2/x/marketmap/keeper"
	marketmaptypes "github.com/skip-mev/connect/v2/x/marketmap/types"
	oraclekeeper "github.com/skip-mev/connect/v2/x/oracle/keeper"
	oracletypes "github.com/skip-mev/connect/v2/x/oracle/types"

	"github.com/initia-labs/minimove/app/ante"

	// noble forwarding keeper
	forwarding "github.com/noble-assets/forwarding/v2"
	forwardingkeeper "github.com/noble-assets/forwarding/v2/keeper"
	forwardingtypes "github.com/noble-assets/forwarding/v2/types"
)

type AppKeepers struct {
	// keys to access the substores
	keys    map[string]*storetypes.KVStoreKey
	tkeys   map[string]*storetypes.TransientStoreKey
	memKeys map[string]*storetypes.MemoryStoreKey

	// keepers
	AccountKeeper         *authkeeper.AccountKeeper
	BankKeeper            *bankkeeper.BaseKeeper
	UpgradeKeeper         *upgradekeeper.Keeper
	GroupKeeper           *groupkeeper.Keeper
	ConsensusParamsKeeper *consensusparamkeeper.Keeper
	IBCKeeper             *ibckeeper.Keeper // IBC Keeper must be a pointer in the app, so we can SetRouter on it correctly
	TransferKeeper        *ibctransferkeeper.Keeper
	NftTransferKeeper     *ibcnfttransferkeeper.Keeper
	AuthzKeeper           *authzkeeper.Keeper
	FeeGrantKeeper        *feegrantkeeper.Keeper
	ICAHostKeeper         *icahostkeeper.Keeper
	ICAControllerKeeper   *icacontrollerkeeper.Keeper
	ICAAuthKeeper         *icaauthkeeper.Keeper
	MoveKeeper            *movekeeper.Keeper
	OPChildKeeper         *opchildkeeper.Keeper
	IBCHooksKeeper        *ibchookskeeper.Keeper
	PacketForwardKeeper   *packetforwardkeeper.Keeper
	OracleKeeper          *oraclekeeper.Keeper // x/oracle keeper used for the connect oracle
	MarketMapKeeper       *marketmapkeeper.Keeper
	ForwardingKeeper      *forwardingkeeper.Keeper
	RatelimitKeeper       *ratelimitkeeper.Keeper

	// light client modules
	TMLightClientModule *ibctm.LightClientModule
	SMLightClientModule *solomachine.LightClientModule
}

func NewAppKeeper(
	ac, vc, cc address.Codec,
	appCodec codec.Codec,
	txConfig client.TxConfig,
	bApp *baseapp.BaseApp,
	legacyAmino *codec.LegacyAmino,
	maccPerms map[string][]string,
	blockedAddress map[string]bool,
	skipUpgradeHeights map[int64]bool,
	homePath string,
	logger log.Logger,
	moveConfig moveconfig.MoveConfig,
	appOpts servertypes.AppOptions,
) AppKeepers {
	appKeepers := AppKeepers{}

	// Set keys KVStoreKey, TransientStoreKey, MemoryStoreKey
	appKeepers.GenerateKeys()

	// register streaming services
	if err := bApp.RegisterStreamingServices(appOpts, appKeepers.keys); err != nil {
		logger.Error("failed to load state streaming", "err", err)
		os.Exit(1)
	}

	authorityAccAddr := authtypes.NewModuleAddress(opchildtypes.ModuleName)
	authorityAddr, err := ac.BytesToString(authorityAccAddr)
	if err != nil {
		logger.Error("failed to retrieve authority address", "err", err)
		os.Exit(1)
	}

	// set the BaseApp's parameter store
	consensusParamsKeeper := consensusparamkeeper.NewKeeper(appCodec, runtime.NewKVStoreService(appKeepers.keys[consensusparamtypes.StoreKey]), authorityAddr, runtime.EventService{})
	appKeepers.ConsensusParamsKeeper = &consensusParamsKeeper
	bApp.SetParamStore(appKeepers.ConsensusParamsKeeper.ParamsStore)

	// add keepers
	appKeepers.MoveKeeper = &movekeeper.Keeper{}

	accountKeeper := authkeeper.NewAccountKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[authtypes.StoreKey]),
		authtypes.ProtoBaseAccount,
		maccPerms,
		ac,
		sdk.GetConfig().GetBech32AccountAddrPrefix(),
		authorityAddr,
	)
	appKeepers.AccountKeeper = &accountKeeper

	bankKeeper := bankkeeper.NewBaseKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[banktypes.StoreKey]),
		appKeepers.AccountKeeper,
		movekeeper.NewMoveBankKeeper(appKeepers.MoveKeeper),
		blockedAddress,
		authorityAddr,
	)
	appKeepers.BankKeeper = &bankKeeper

	/////////////////////////////////
	// OPChildKeeper Configuration //
	/////////////////////////////////

	// initialize oracle keeper
	marketMapKeeper := marketmapkeeper.NewKeeper(
		runtime.NewKVStoreService(appKeepers.keys[marketmaptypes.StoreKey]),
		appCodec,
		authorityAccAddr,
	)
	appKeepers.MarketMapKeeper = marketMapKeeper

	oracleKeeper := oraclekeeper.NewKeeper(
		runtime.NewKVStoreService(appKeepers.keys[oracletypes.StoreKey]),
		appCodec,
		marketMapKeeper,
		authorityAccAddr,
	)
	appKeepers.OracleKeeper = &oracleKeeper

	// Add the oracle keeper as a hook to market map keeper so new market map entries can be created
	// and propagated to the oracle keeper.
	appKeepers.MarketMapKeeper.SetHooks(appKeepers.OracleKeeper.Hooks())

	appKeepers.OPChildKeeper = opchildkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[opchildtypes.StoreKey]),
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		appKeepers.OracleKeeper,
		ante.CreateAnteHandlerForOPinit(appKeepers.AccountKeeper, txConfig.SignModeHandler()),
		txConfig.TxDecoder(),
		bApp.MsgServiceRouter(),
		authorityAddr,
		ac,
		vc,
		cc,
		authcodec.NewBech32Codec("init"),
		logger,
	)

	appKeepers.UpgradeKeeper = upgradekeeper.NewKeeper(
		skipUpgradeHeights,
		runtime.NewKVStoreService(appKeepers.keys[upgradetypes.StoreKey]),
		appCodec,
		homePath,
		bApp,
		authorityAddr,
	)

	feeGrantKeeper := feegrantkeeper.NewKeeper(appCodec, runtime.NewKVStoreService(appKeepers.keys[feegrant.StoreKey]), appKeepers.AccountKeeper)
	appKeepers.FeeGrantKeeper = &feeGrantKeeper

	authzKeeper := authzkeeper.NewKeeper(runtime.NewKVStoreService(appKeepers.keys[authzkeeper.StoreKey]), appCodec, bApp.MsgServiceRouter(), appKeepers.AccountKeeper)
	authzKeeper = authzKeeper.SetBankKeeper(appKeepers.BankKeeper)
	appKeepers.AuthzKeeper = &authzKeeper

	groupConfig := group.DefaultConfig()
	groupKeeper := groupkeeper.NewKeeper(
		appKeepers.keys[group.StoreKey],
		appCodec,
		bApp.MsgServiceRouter(),
		appKeepers.AccountKeeper,
		groupConfig,
	)
	appKeepers.GroupKeeper = &groupKeeper

	// Create IBC Keeper
	appKeepers.IBCKeeper = ibckeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[ibcexported.StoreKey]),
		nil, // we don't need migration
		appKeepers.UpgradeKeeper,
		authorityAddr,
	)

	if err := appKeepers.OPChildKeeper.SetIBCKeepers(appKeepers.IBCKeeper.ClientKeeper); err != nil {
		logger.Error("failed to setup IBCKeepers on OPChildKeeper", "error", err.Error())
		tmos.Exit(err.Error())
	}

	// Set IBC post handler to receive validator set updates
	appKeepers.IBCKeeper.ClientKeeper.SetPostUpdateHandler(
		appKeepers.OPChildKeeper.UpdateHostValidatorSet,
	)

	appKeepers.IBCHooksKeeper = ibchookskeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[ibchookstypes.StoreKey]),
		runtime.NewTransientStoreService(appKeepers.tkeys[ibchookstypes.TStoreKey]),
		authorityAddr,
		ac,
	)

	appKeepers.ForwardingKeeper = forwardingkeeper.NewKeeper(
		appCodec,
		logger,
		runtime.NewKVStoreService(appKeepers.keys[forwardingtypes.StoreKey]),
		runtime.NewTransientStoreService(appKeepers.tkeys[forwardingtypes.TransientStoreKey]),
		appheaderinfo.NewHeaderInfoService(),
		runtime.ProvideEventService(),
		authorityAddr,
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		appKeepers.IBCKeeper.ChannelKeeper,
		appKeepers.TransferKeeper,
	)
	appKeepers.BankKeeper.AppendSendRestriction(appKeepers.ForwardingKeeper.SendRestrictionFn)

	////////////////////////////
	// Transfer configuration //
	////////////////////////////
	// Send   : transfer -> packet forward -> rate limit -> ibchooks -> channel
	// Receive: channel  -> legacyfeeack   -> ibchooks(move) -> opchild-migration -> rate limit -> packet forward -> forwarding -> transfer

	var transferStack porttypes.IBCModule
	{
		packetForwardKeeper := &packetforwardkeeper.Keeper{}
		rateLimitKeeper := &ratelimitkeeper.Keeper{}
		ibcHooksICS4Wrapper := &ibchooks.ICS4Middleware{}

		// Create Transfer Keeper
		transferKeeper := ibctransferkeeper.NewKeeper(
			appCodec,
			runtime.NewKVStoreService(appKeepers.keys[ibctransfertypes.StoreKey]),
			nil, // we don't need migration
			// ics4wrapper: transfer -> packet forward
			packetForwardKeeper,
			appKeepers.IBCKeeper.ChannelKeeper,
			bApp.MsgServiceRouter(),
			appKeepers.AccountKeeper,
			appKeepers.BankKeeper,
			authorityAddr,
		)
		appKeepers.TransferKeeper = &transferKeeper
		transferStack = ibctransfer.NewIBCModule(*appKeepers.TransferKeeper)

		// forwarding middleware
		transferStack = forwarding.NewMiddleware(
			// receive: forwarding -> transfer
			transferStack,
			appKeepers.AccountKeeper,
			appKeepers.ForwardingKeeper,
		)

		// create packet forward middleware
		*packetForwardKeeper = *packetforwardkeeper.NewKeeper(
			appCodec,
			runtime.NewKVStoreService(appKeepers.keys[packetforwardtypes.StoreKey]),
			appKeepers.TransferKeeper,
			appKeepers.IBCKeeper.ChannelKeeper,
			appKeepers.BankKeeper,
			// ics4wrapper: transfer -> packet forward -> rate limit
			rateLimitKeeper,
			authorityAddr,
		)
		appKeepers.PacketForwardKeeper = packetForwardKeeper
		transferStack = packetforward.NewIBCMiddleware(
			// receive: packet forward -> forwarding -> transfer
			transferStack,
			appKeepers.PacketForwardKeeper,
			0,
			packetforwardkeeper.DefaultForwardTransferPacketTimeoutTimestamp,
		)

		// create the rate limit keeper
		*rateLimitKeeper = *ratelimitkeeper.NewKeeper(
			appCodec,
			runtime.NewKVStoreService(appKeepers.keys[ratelimittypes.StoreKey]),
			paramtypes.Subspace{}, // empty params
			authorityAddr,
			appKeepers.BankKeeper,
			appKeepers.IBCKeeper.ChannelKeeper,
			appKeepers.IBCKeeper.ClientKeeper,
			// ics4wrapper: transfer -> packet forward -> rate limit -> ibchooks
			ibcHooksICS4Wrapper,
		)
		appKeepers.RatelimitKeeper = rateLimitKeeper

		// rate limit middleware
		transferStack = ratelimit.NewIBCMiddleware(
			*appKeepers.RatelimitKeeper,
			// receive: rate limit -> packet forward -> forwarding -> transfer
			transferStack,
		)

		// opchild migration middleware (rollup-side denom migration)
		transferStack = opchildmigration.NewIBCMiddleware(
			ac,
			appCodec,
			// receive: migration -> rate limit -> packet forward -> forwarding -> transfer
			transferStack,
			nil, /* ics4wrapper: not used */
			appKeepers.BankKeeper,
			appKeepers.OPChildKeeper,
		)

		// create move ibc-hooks middleware for transfer
		*ibcHooksICS4Wrapper = *ibchooks.NewICS4Middleware(
			// ics4wrapper: ibchooks -> channel
			appKeepers.IBCKeeper.ChannelKeeper,
			appKeepers.IBCHooksKeeper,
			ibcmovehooks.NewMoveHooks(ac, appCodec, logger, appKeepers.MoveKeeper),
		)
		transferStack = ibchooks.NewIBCMiddleware(
			// receive: ibchooks(move) -> migration -> rate limit -> packet forward -> forwarding -> transfer
			transferStack,
			ibcHooksICS4Wrapper,
			appKeepers.IBCHooksKeeper,
		)

		// legacy 29-fee ack compatibility for pre-v10 channels
		// whose counterparties are still on ibc-go v8.
		transferStack = legacyfeeack.NewIBCMiddleware(transferStack)
	}

	////////////////////////////////
	// Nft Transfer configuration //
	////////////////////////////////

	var nftTransferStack porttypes.IBCModule
	{
		ibcHooksICS4Wrapper := &ibchooks.ICS4Middleware{}

		// Create NFT Transfer Keeper
		appKeepers.NftTransferKeeper = ibcnfttransferkeeper.NewKeeper(
			appCodec,
			runtime.NewKVStoreService(appKeepers.keys[ibcnfttransfertypes.StoreKey]),
			// ics4wrapper: nft transfer -> ibchooks
			ibcHooksICS4Wrapper,
			appKeepers.IBCKeeper.ChannelKeeper,
			appKeepers.AccountKeeper,
			movekeeper.NewNftKeeper(appKeepers.MoveKeeper),
			authorityAddr,
		)
		nftTransferIBCModule := ibcnfttransfer.NewIBCModule(*appKeepers.NftTransferKeeper)
		nftTransferStack = nftTransferIBCModule

		// create move ibc-hooks middleware for nft-transfer
		*ibcHooksICS4Wrapper = *ibchooks.NewICS4Middleware(
			// ics4wrapper: ibchooks -> channel
			appKeepers.IBCKeeper.ChannelKeeper,
			appKeepers.IBCHooksKeeper,
			ibcmovehooks.NewMoveHooks(ac, appCodec, logger, appKeepers.MoveKeeper),
		)
		nftTransferStack = ibchooks.NewIBCMiddleware(
			nftTransferStack,
			ibcHooksICS4Wrapper,
			appKeepers.IBCHooksKeeper,
		)

		// legacy 29-fee ack compatibility for pre-v10 channels
		// whose counterparties are still on ibc-go v8.
		nftTransferStack = legacyfeeack.NewIBCMiddleware(nftTransferStack)
	}

	///////////////////////////
	// OPChild configuration //
	///////////////////////////

	opchildStack := legacyfeeack.NewIBCMiddleware(opchild.NewIBCModule(*appKeepers.OPChildKeeper))

	///////////////////////
	// ICA configuration //
	///////////////////////

	var icaHostStack porttypes.IBCModule
	var icaControllerStack porttypes.IBCModule
	{
		icaHostKeeper := icahostkeeper.NewKeeper(
			appCodec,
			runtime.NewKVStoreService(appKeepers.keys[icahosttypes.StoreKey]),
			nil, // we don't need migration
			appKeepers.IBCKeeper.ChannelKeeper,
			appKeepers.IBCKeeper.ChannelKeeper, // ics4Wrapper
			appKeepers.AccountKeeper,
			bApp.MsgServiceRouter(),
			bApp.GRPCQueryRouter(),
			authorityAddr,
		)
		appKeepers.ICAHostKeeper = &icaHostKeeper

		icaControllerKeeper := icacontrollerkeeper.NewKeeper(
			appCodec,
			runtime.NewKVStoreService(appKeepers.keys[icacontrollertypes.StoreKey]),
			nil,                                // we don't need migration
			appKeepers.IBCKeeper.ChannelKeeper, // ics4Wrapper
			appKeepers.IBCKeeper.ChannelKeeper,
			bApp.MsgServiceRouter(),
			authorityAddr,
		)
		appKeepers.ICAControllerKeeper = &icaControllerKeeper

		icaAuthKeeper := icaauthkeeper.NewKeeper(
			appCodec,
			*appKeepers.ICAControllerKeeper,
			ac,
		)
		appKeepers.ICAAuthKeeper = &icaAuthKeeper

		icaAuthIBCModule := icaauth.NewIBCModule(*appKeepers.ICAAuthKeeper)
		icaHostIBCModule := icahost.NewIBCModule(*appKeepers.ICAHostKeeper)
		// legacyfeeack outermost on both ICA stacks for fee-wrapped channel ack compatibility.
		icaHostStack = legacyfeeack.NewIBCMiddleware(icaHostIBCModule)
		icaControllerStack = legacyfeeack.NewIBCMiddleware(
			icacontroller.NewIBCMiddlewareWithAuth(icaAuthIBCModule, *appKeepers.ICAControllerKeeper),
		)
	}

	//////////////////////////////
	// IBC router Configuration //
	//////////////////////////////

	// Create static IBC router, add transfer route, then set and seal it
	ibcRouter := porttypes.NewRouter()
	ibcRouter.AddRoute(ibctransfertypes.ModuleName, transferStack).
		AddRoute(icahosttypes.SubModuleName, icaHostStack).
		AddRoute(icacontrollertypes.SubModuleName, icaControllerStack).
		AddRoute(icaauthtypes.ModuleName, icaControllerStack).
		// new v10 PortKeeper.Route requires alphanumeric route keys but does a substring
		// fallback (over sorted Keys()) when exact match fails. PortID "nft-transfer"
		// has a hyphen so we register under "nft". sorts before "transfer", is a
		// substring of "nft-transfer", so the fallback resolves deterministically.
		AddRoute(ibcnfttransfertypes.IbcRouterKey, nftTransferStack).
		AddRoute(opchildtypes.ModuleName, opchildStack)

	appKeepers.IBCKeeper.SetRouter(ibcRouter)
	appKeepers.OPChildKeeper.
		WithTransferKeeper(appKeepers.TransferKeeper).
		WithChannelKeeper(appKeepers.IBCKeeper.ChannelKeeper)

	// register light client modules
	clientKeeper := appKeepers.IBCKeeper.ClientKeeper
	storeProvider := appKeepers.IBCKeeper.ClientKeeper.GetStoreProvider()

	tmLightClientModule := ibctm.NewLightClientModule(appCodec, storeProvider)
	clientKeeper.AddRoute(ibctm.ModuleName, &tmLightClientModule)
	appKeepers.TMLightClientModule = &tmLightClientModule

	smLightClientModule := solomachine.NewLightClientModule(appCodec, storeProvider)
	clientKeeper.AddRoute(solomachine.ModuleName, &smLightClientModule)
	appKeepers.SMLightClientModule = &smLightClientModule

	//////////////////////////////
	// MoveKeeper Configuration //
	//////////////////////////////

	queryWhitelist := movetypes.DefaultVMQueryWhiteList(ac)
	queryWhitelist.Custom["chain_id"] = func(ctx context.Context, _ []byte) ([]byte, error) {
		return []byte(sdk.UnwrapSDKContext(ctx).ChainID()), nil
	}
	queryWhitelist.Stargate["/connect.oracle.v2.Query/GetAllCurrencyPairs"] = movetypes.ProtoSet{
		Request:  &oracletypes.GetAllCurrencyPairsRequest{},
		Response: &oracletypes.GetAllCurrencyPairsResponse{},
	}
	queryWhitelist.Stargate["/connect.oracle.v2.Query/GetPrice"] = movetypes.ProtoSet{
		Request:  &oracletypes.GetPriceRequest{},
		Response: &oracletypes.GetPriceResponse{},
	}
	queryWhitelist.Stargate["/connect.oracle.v2.Query/GetPrices"] = movetypes.ProtoSet{
		Request:  &oracletypes.GetPricesRequest{},
		Response: &oracletypes.GetPricesResponse{},
	}

	*appKeepers.MoveKeeper = movekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[movetypes.StoreKey]),
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		appKeepers.OracleKeeper,
		bApp.MsgServiceRouter(),
		bApp.GRPCQueryRouter(),
		moveConfig,
		// staking feature
		nil, // placeholder for distribution keeper
		nil, // placeholder for staking keeper
		nil, // placeholder for reward keeper,
		newCommunityPoolKeeper(appKeepers.BankKeeper, authtypes.FeeCollectorName),
		authtypes.FeeCollectorName,
		authorityAddr,
		ac, vc,
	).WithVMQueryWhitelist(queryWhitelist)

	return appKeepers
}
