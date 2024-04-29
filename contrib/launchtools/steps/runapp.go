package steps

import (
	"context"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/initia-labs/minimove/contrib/launchtools"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

func RunApp(_ launchtools.Input) launchtools.LauncherStepFunc {
	return func(ctx *launchtools.LauncherContext) error {
		// temporarily allow creation of empty blocks
		// this should help creation of ibc channels.
		// NOTE: This part is ephemeral only in the context of the launcher.
		ctx.ServerCtx.Config.Consensus.CreateEmptyBlocks = true
		ctx.ServerCtx.Config.Consensus.CreateEmptyBlocksInterval = CreateEmptyBlocksInterval

		// create a channel to synchronise on app creation
		var syncDone = make(chan interface{})

		// create cobra command context
		startCmd := server.StartCmdWithOptions(
			ctx.AppCreator,
			ctx.ClientContext.HomeDir,

			// set up a post setup function to set the app in the context
			server.StartCmdOptions{
				PostSetup: func(svrCtx *server.Context, clientCtx client.Context, _ context.Context, _ *errgroup.Group, app servertypes.Application) (err error) {
					// Register the app in the context
					ctx.SetApp(app)

					// Add cleanup function to close the app
					ctx.AddCleanupFn(func() error {
						return app.Close()
					})

					// Signal that the app is created
					syncDone <- struct{}{}

					return nil
				},
			},
		)

		// set relevant context; this part is necessary to correctly set up the start command and their start-up flags
		startCmd.SetContext(ctx.Cmd.Context())

		// Run PreRunE from startCmd. This step is necessary to correctly set up start-up flags,
		// as it is done usually with cometbft start command.
		if err := startCmd.PreRunE(startCmd, nil); err != nil {
			return errors.Wrapf(err, "failed to prerun command")
		}

		// Run RunE command - this part fires up the actual chain
		// Note that the command is run in a separate goroutine, as it is blocking.
		// App should be later cleaned up in another launcher step
		go func() {
			if err := startCmd.RunE(startCmd, nil); err != nil {
				panic(errors.Wrapf(err, "failed to run command"))
			}
		}()

		<-syncDone

		return nil
	}
}
