package steps

import (
	"context"
	"encoding/json"
	ibctypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	initiahook "github.com/initia-labs/initia/app/hook"
	minitiaapp "github.com/initia-labs/minimove/app"
	"github.com/initia-labs/minimove/contrib/launchtools"
	"github.com/pkg/errors"
)

func InitializeOpBridge(input launchtools.Input) launchtools.LauncherStepFunc {
	return func(ctx *launchtools.LauncherContext) error {
		minitiaApp, ok := ctx.GetApp().(*minitiaapp.MinitiaApp)
		if !ok {
			return errors.New("App is not MinitiaApp")
		}

		resp, err := minitiaApp.IBCKeeper.QueryServer.Channels(
			context.Background(),
			&ibctypes.QueryChannelsRequest{},
		)
		if err != nil {
			return errors.Wrap(err, "failed to query client states")
		}

		// generate initiahook.PermsMetadata
		// assume that all channels in IBC keeper need to be permitted on OPChild
		// [transfer, nft-transfer, ...]
		permChannels := make([]initiahook.PortChannelID, 0)
		for _, channel := range resp.Channels {
			permChannels = append(permChannels, initiahook.PortChannelID{
				PortID:    channel.PortId,
				ChannelID: channel.ChannelId,
			})
		}

		permsMetadata := initiahook.PermsMetadata{PermChannels: permChannels}
		permsMetadataJSON, err := json.Marshal(permsMetadata)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal perms metadata")
		}

		// create OpBridgeMessage
		createOpBridgeMessage := ophosttypes.NewMsgCreateBridge(
			input.SystemKeys.Executor.Address,
			ophosttypes.BridgeConfig{
				Challenger: input.SystemKeys.Challenger.Address,
				Proposer:   input.SystemKeys.Output.Address,
				BatchInfo: ophosttypes.BatchInfo{
					Submitter: input.SystemKeys.Submitter.Address,
					Chain:     input.OpBridge.SubmitTarget,
				},
				SubmissionInterval:  input.OpBridge.SubmissionInterval,
				FinalizationPeriod:  input.OpBridge.FinalizationPeriod,
				SubmissionStartTime: input.OpBridge.SubmissionStartTime,
				Metadata:            permsMetadataJSON, // ???
			},
		)

		// somehow send createOpBridgeMessage to OPChild
		ctx.ServerCtx.Logger.Error(
			"unimplemeted: send createOpBridgeMessage to OPChild",
			"msg", createOpBridgeMessage,
		)

		return nil
	}
}
