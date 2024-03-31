package main

import (
	"context"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/types"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	indexer "github.com/initia-labs/kvindexer"
	indexercfg "github.com/initia-labs/kvindexer/config"
	minitiaapp "github.com/initia-labs/minimove/app"
)

func addIndexFlag(cmd *cobra.Command) {
	indexercfg.AddIndexerFlag(cmd)
}

func preSetupIndexer(svrCtx *server.Context, clientCtx client.Context, ctx context.Context, g *errgroup.Group, _app types.Application) error {
	app := _app.(*minitiaapp.MinitiaApp)

	// listen all keys
	keysToListen := []storetypes.StoreKey{}

	// TODO: if it downgrades performacne, have to set only keys for registered submodules and crons
	keys := app.GetKeys()
	for _, key := range keys {
		keysToListen = append(keysToListen, key)
	}
	app.CommitMultiStore().AddListeners(keysToListen)

	indexer, err := indexer.NewIndexer(app.GetBaseApp().Logger(), app.GetIndexerKeeper())
	// if err is not nil, it means there is an error regardless of indexer is nil or not.
	// else if indexer is nil, it means indexer is disabled and the returned err is nil.
	if err != nil || indexer == nil {
		return err
	}

	if err = indexer.Validate(); err != nil {
		return err
	}

	if err = indexer.Prepare(nil); err != nil {
		return err
	}
	if err = app.GetIndexerKeeper().Seal(); err != nil {
		return err
	}

	if err = indexer.Start(nil); err != nil {
		return err
	}

	streamingManager := storetypes.StreamingManager{
		ABCIListeners: []storetypes.ABCIListener{indexer},
		StopNodeOnErr: true,
	}
	app.SetStreamingManager(streamingManager)

	return nil
}

var startCmdOptions = server.StartCmdOptions{
	DBOpener: nil,
	PreSetup: preSetupIndexer,
	AddFlags: addIndexFlag,
}
