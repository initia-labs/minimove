package steps

import (
	"errors"
	launchertypes "github.com/initia-labs/minimove/contrib/launchtools"
)

func StopApp(_ launchertypes.Input) launchertypes.LauncherStepFunc {
	return func(ctx *launchertypes.LauncherContext) error {
		if !ctx.IsAppInitialized() {
			return errors.New("app is not initialized")
		}

		log := ctx.ServerCtx.Logger
		log.Info("cleanup")

		return ctx.Cleanup()
	}
}
