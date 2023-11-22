package lanes

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"

	"github.com/skip-mev/block-sdk/block/base"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
)

type bankKeeper interface {
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
}

type opChildKeeper interface {
	GetParams(ctx sdk.Context) (params opchildtypes.Params)
}

// FreeLaneMatchHandler returns the default match handler for the free lane. The
// default implementation matches transactions that are ibc related. In particular,
// any transaction that is a MsgTimeout, MsgAcknowledgement.
func FreeLaneMatchHandler(bk bankKeeper, opchildKeeper opChildKeeper) base.MatchHandler {
	return func(ctx sdk.Context, tx sdk.Tx) bool {
		// allow ibc messages
		for _, msg := range tx.GetMsgs() {
			switch msg.(type) {
			case *channeltypes.MsgTimeout:
				return true
			case *channeltypes.MsgAcknowledgement:
				return true
			}
		}

		feeTx, ok := tx.(sdk.FeeTx)
		if !ok {
			return false
		}

		feePayer := feeTx.FeePayer()
		params := opchildKeeper.GetParams(ctx)
		for _, minGasPrice := range params.MinGasPrices {
			denom := minGasPrice.Denom
			requiredAmount := minGasPrice.Amount.MulInt64(1_000_000_000).TruncateInt()

			payerBalance := bk.GetBalance(ctx, feePayer, denom)
			if payerBalance.Amount.GTE(requiredAmount) {
				return true
			}
		}

		return false
	}
}
