package hook

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"

	movekeeper "github.com/initia-labs/initia/x/move/keeper"
	movetypes "github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/initiavm/types"
)

// bridge hook implementation for move
type MoveBridgeHook struct {
	moveKeeper *movekeeper.Keeper
}

func NewMoveBridgeHook(moveKeeper *movekeeper.Keeper) MoveBridgeHook {
	return MoveBridgeHook{moveKeeper}
}

func (mbh MoveBridgeHook) Hook(ctx sdk.Context, sender sdk.AccAddress, msgBytes []byte) error {
	msg := movetypes.MsgExecute{}
	err := json.Unmarshal(msgBytes, &msg)
	if err != nil {
		return err
	}

	senderAddr, err := vmtypes.NewAccountAddressFromBytes(sender)
	if err != nil {
		return err
	}

	moduleAddress, err := movetypes.AccAddressFromString(msg.ModuleAddress)
	if err != nil {
		return err
	}

	typeArgs, err := movetypes.TypeTagsFromTypeArgs(msg.TypeArgs)
	if err != nil {
		return err
	}

	err = mbh.moveKeeper.ExecuteEntryFunction(
		ctx,
		senderAddr,
		moduleAddress,
		msg.ModuleName,
		msg.FunctionName,
		typeArgs,
		msg.Args,
	)

	return err
}
