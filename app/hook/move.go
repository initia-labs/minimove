package hook

import (
	"context"
	"encoding/json"
	"strings"

	"cosmossdk.io/core/address"
	sdk "github.com/cosmos/cosmos-sdk/types"

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
	isJSON, msgBytes, err := checkIsJSON(msgBytes)
	if err != nil {
		return err
	}

	if isJSON {
		return mbh.handleMsgExecuteJSON(ctx, sender, msgBytes)
	} else {
		return mbh.handleMsgExecute(ctx, sender, msgBytes)
	}
}

func checkIsJSON(msgBytes []byte) (bool, []byte, error) {
	var jsonObj map[string]interface{}
	err := json.Unmarshal(msgBytes, &jsonObj)
	if err != nil {
		return false, nil, err
	}

	isJSON := false
	if val, ok := jsonObj["is_json"]; ok && val == true {
		isJSON = true
	}

	// remove is_json field from json object for decoding
	delete(jsonObj, "is_json")
	bz, err := json.Marshal(jsonObj)
	if err != nil {
		return false, nil, err
	}

	return isJSON, bz, nil
}

func (mbh MoveBridgeHook) handleMsgExecute(ctx context.Context, sender sdk.AccAddress, msgBytes []byte) error {
	var msg movetypes.MsgExecute
	decoder := json.NewDecoder(strings.NewReader(string(msgBytes)))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&msg)
	if err != nil {
		return err
	}

	// overwrite sender with the actual sender
	msg.Sender, err = mbh.ac.BytesToString(sender)
	if err != nil {
		return err
	}

	ms := movekeeper.NewMsgServerImpl(mbh.moveKeeper)
	_, err = ms.Execute(ctx, &msg)

	return err
}

func (mbh MoveBridgeHook) handleMsgExecuteJSON(ctx context.Context, sender sdk.AccAddress, msgBytes []byte) error {
	var msg movetypes.MsgExecuteJSON
	decoder := json.NewDecoder(strings.NewReader(string(msgBytes)))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&msg)
	if err != nil {
		return err
	}

	// overwrite sender with the actual sender
	msg.Sender, err = mbh.ac.BytesToString(sender)
	if err != nil {
		return err
	}

	ms := movekeeper.NewMsgServerImpl(mbh.moveKeeper)
	_, err = ms.ExecuteJSON(ctx, &msg)

	return err
}
