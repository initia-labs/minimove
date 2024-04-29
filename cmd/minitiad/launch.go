package main

import (
	"cosmossdk.io/errors"
	"fmt"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/initia-labs/initia/app/params"
	launchtools "github.com/initia-labs/minimove/contrib/launchtools"
	launchsteps "github.com/initia-labs/minimove/contrib/launchtools/steps"
	"github.com/spf13/cobra"
	"reflect"
)

var DefaultSteps = []launchtools.LauncherStepFuncFactory[launchtools.Input]{
	launchsteps.InitializeConfig,
	launchsteps.InitializeGenesis,
	launchsteps.InitializeKeyring,
	launchsteps.RequestFaucetOnL1,
	launchsteps.RunApp,

	// MINIWASM??
	// initializeWasms; use instantiate2 to predict addresses?

	launchsteps.EstablishIBCChannels,
	// TODO: establish nft-transfer channel

	// HOW??
	//launchsteps.InitializeOpBridge,
}

var CleanupSteps = []launchtools.LauncherStepFuncFactory[launchtools.Input]{
	launchsteps.StopApp,
}

// LauncherCmd spawns a in-binary initializer for a Minitia chain.
// It takes a path to a manifest.json (whose type is defined in contrib/launcher/manifest.go) file as an argument.
//
// The launcher will go through a series of steps to initialize the chain, as defined in DefaultSteps.
// Each step is a factory function that takes an input (defined in contrib/launcher/types.go) and returns a LauncherStepFunc.
// This way, each step can validate the input **before** any step is run. This is useful for ensuring that the input is correct
// before running any steps that might depend on it.
//
// As such, the only function of this Command is to create minimum viable environment for an in-process chain to run,
// namely contexts pertaining to CosmosSDK application and CometBFT server.
// Main heavy lifting(s) are done in the DefaultSteps.
func LauncherCmd(encodingConfig params.EncodingConfig, mbm module.BasicManager) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "launch [path to manifest.json]",
		Short: "Launch a Minitia chain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			manifestPath := args[0]

			// create client and server context from this cmd
			// Note: these contexts are **heavily** modified in NewLauncher.
			// When in doubt, check the implementation of NewLauncher.
			clientCtx := client.GetClientContextFromCmd(cmd)
			serverCtx := server.GetServerContextFromCmd(cmd)

			// use server-provided logger
			log := serverCtx.Logger

			// create launcher context
			launcherCtx := launchtools.NewLauncher(
				clientCtx.HomeDir,
				&clientCtx,
				serverCtx,
				mbm,
				appCreator{encodingConfig}.newApp,
				encodingConfig,
				cmd,
			)

			// read manifest from the path, turn it into an allocated struct
			manifest, err := launchtools.NewManifestFromFile[launchtools.Input](manifestPath)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("failed to unmarshal manifest: %s", manifestPath))
			}

			// run factory functions on the DefaultSteps.
			// After this line, we end up with a list of LauncherStepFuncs -- callable functions
			// that take a LauncherContext.
			stepFns := make([]launchtools.LauncherStepFunc, 0)
			for _, step := range DefaultSteps {
				log.Info("registering step function",
					"step-fn", reflect.ValueOf(step).Type().Name(),
				)
				stepFns = append(stepFns, step(*manifest))
			}

			for _, cleanupStep := range CleanupSteps {
				log.Info("registering step cleanup function",
					"step-fn", reflect.ValueOf(cleanupStep),
				)
				stepFns = append(stepFns, cleanupStep(*manifest))
			}

			// run steps in series
			for i, step := range stepFns {
				if err := step(launcherCtx); err != nil {
					return errors.Wrapf(err, "failed to run step %d", i+1)
				}
			}

			// return nil if everything went well
			return nil
		},
	}

	return cmd
}
