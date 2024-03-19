package hook

import (
	"context"
	"encoding/json"
	"strings"

	"cosmossdk.io/core/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	movekeeper "github.com/initia-labs/initia/x/move/keeper"
	movetypes "github.com/initia-labs/initia/x/move/types"
)

// bridge hook implementation for move
type MoveBridgeHook struct {
	ac         address.Codec
	moveKeeper *movekeeper.Keeper
}

func NewMoveBridgeHook(ac address.Codec, moveKeeper *movekeeper.Keeper) MoveBridgeHook {
	return MoveBridgeHook{ac, moveKeeper}
}

func (mbh MoveBridgeHook) Hook(ctx context.Context, sender sdk.AccAddress, msgBytes []byte) error {
	var msg movetypes.MsgExecute
	decoder := json.NewDecoder(strings.NewReader(string(msgBytes)))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&msg)
	if err != nil {
		return err
	}

	senderAddr, err := mbh.ac.StringToBytes(msg.Sender)
	if err != nil {
		return err
	} else if !sender.Equals(sdk.AccAddress(senderAddr)) {
		return sdkerrors.ErrUnauthorized
	}

	ms := movekeeper.NewMsgServerImpl(mbh.moveKeeper)
	_, err = ms.Execute(ctx, &msg)

	return err
}
