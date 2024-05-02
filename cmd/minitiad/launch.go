package main

import (
	"github.com/initia-labs/OPinit/contrib/launchtools"
	"github.com/initia-labs/OPinit/contrib/launchtools/steps"
)

// DefaultLaunchStepFactories is a list of default launch step factories.
var DefaultLaunchStepFactories = []launchtools.LauncherStepFuncFactory[launchtools.Input]{
	steps.InitializeConfig,
	steps.InitializeGenesis,
	steps.InitializeKeyring,
	steps.RunApp,
	steps.EstablishIBCChannels,
	steps.InitializeOpBridge,
}
