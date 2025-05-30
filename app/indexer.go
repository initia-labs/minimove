package app

import (
	storetypes "cosmossdk.io/store/types"

	dbm "github.com/cosmos/cosmos-db"

	servertypes "github.com/cosmos/cosmos-sdk/server/types"

	// kvindexer
	"github.com/initia-labs/initia/app/params"
	kvindexer "github.com/initia-labs/kvindexer"
	kvindexerconfig "github.com/initia-labs/kvindexer/config"
	blocksubmodule "github.com/initia-labs/kvindexer/submodules/block"
	nft "github.com/initia-labs/kvindexer/submodules/move-nft"
	"github.com/initia-labs/kvindexer/submodules/pair"
	tx "github.com/initia-labs/kvindexer/submodules/tx"
	kvindexermodule "github.com/initia-labs/kvindexer/x/kvindexer"
	kvindexerkeeper "github.com/initia-labs/kvindexer/x/kvindexer/keeper"
)

func setupIndexer(
	app *MinitiaApp,
	appOpts servertypes.AppOptions,
	encodingConfig params.EncodingConfig,
	kvindexerDB dbm.DB,
) (*kvindexerkeeper.Keeper, *kvindexermodule.AppModuleBasic, *storetypes.StreamingManager, error) {
	// initialize the indexer keeper
	kvindexerConfig, err := kvindexerconfig.NewConfig(appOpts)
	if err != nil {
		return nil, nil, nil, err
	}
	kvIndexerKeeper := kvindexerkeeper.NewKeeper(
		app.appCodec,
		"move",
		kvindexerDB,
		kvindexerConfig,
		app.ac,
		app.vc,
	)

	smBlock, err := blocksubmodule.NewBlockSubmodule(encodingConfig, kvIndexerKeeper, app.OPChildKeeper)
	if err != nil {
		return nil, nil, nil, err
	}
	smTx, err := tx.NewTxSubmodule(encodingConfig, kvIndexerKeeper)
	if err != nil {
		return nil, nil, nil, err
	}
	smPair, err := pair.NewPairSubmodule(encodingConfig, kvIndexerKeeper, app.IBCKeeper.ChannelKeeper, app.TransferKeeper)
	if err != nil {
		return nil, nil, nil, err
	}
	smNft, err := nft.NewMoveNftSubmodule(app.ac, encodingConfig, kvIndexerKeeper, app.MoveKeeper, smPair)
	if err != nil {
		return nil, nil, nil, err
	}

	// order of registration matters: smPair must be registered before smNft since smNft depends on smPair
	err = kvIndexerKeeper.RegisterSubmodules(smBlock, smTx, smPair, smNft)
	if err != nil {
		return nil, nil, nil, err
	}

	// Add your implementation here

	kvIndexer, err := kvindexer.NewIndexer(app.GetBaseApp().Logger(), kvIndexerKeeper)
	if err != nil {
		return nil, nil, nil, err
	} else if kvIndexer == nil {
		return nil, nil, nil, nil
	}

	if err = kvIndexer.Validate(); err != nil {
		return nil, nil, nil, err
	}

	if err = kvIndexer.Prepare(nil); err != nil {
		return nil, nil, nil, err
	}

	if err = kvIndexerKeeper.Seal(); err != nil {
		return nil, nil, nil, err
	}

	if err = kvIndexer.Start(nil); err != nil {
		return nil, nil, nil, err
	}

	kvIndexerModule := kvindexermodule.NewAppModuleBasic(kvIndexerKeeper)
	streamingManager := storetypes.StreamingManager{
		ABCIListeners: []storetypes.ABCIListener{kvIndexer},
		StopNodeOnErr: true,
	}

	return kvIndexerKeeper, &kvIndexerModule, &streamingManager, nil
}
