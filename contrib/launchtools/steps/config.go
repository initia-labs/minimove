package steps

import "github.com/initia-labs/minimove/contrib/launchtools"

// InitializeConfig sets the config for the server context.
func InitializeConfig(manifest launchtools.Input) launchtools.LauncherStepFunc {
	return func(ctx *launchtools.LauncherContext) error {
		// set config
		config := ctx.ServerCtx.Config
		config.SetRoot(ctx.ClientContext.HomeDir)
		config.Moniker = manifest.L2Config.Moniker

		return nil
	}
}
