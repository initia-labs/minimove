package hook

import (
	"context"
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"

	movekeeper "github.com/initia-labs/initia/x/move/keeper"
	movetypes "github.com/initia-labs/initia/x/move/types"
)

// bridge hook implementation for move
type MoveBridgeHook struct {
	moveKeeper *movekeeper.Keeper
}

func NewMoveBridgeHook(moveKeeper *movekeeper.Keeper) MoveBridgeHook {
	return MoveBridgeHook{moveKeeper}
}

func (mbh MoveBridgeHook) Hook(ctx context.Context, sender sdk.AccAddress, msgBytes []byte) error {
	msg := movetypes.MsgExecute{}
	err := json.Unmarshal(msgBytes, &msg)
	if err != nil {
		return err
	}

	ms := movekeeper.NewMsgServerImpl(*mbh.moveKeeper)
	_, err = ms.Execute(ctx, &msg)

	return err
}
