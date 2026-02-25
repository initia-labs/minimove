package ante

import (
	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	ibcante "github.com/cosmos/ibc-go/v8/modules/core/ante"

	opchildante "github.com/initia-labs/OPinit/x/opchild/ante"
	initiaante "github.com/initia-labs/initia/app/ante"
	"github.com/initia-labs/initia/app/ante/sigverify"
	moveante "github.com/initia-labs/initia/x/move/ante"
)

// NewMinimalAnteHandler returns a reduced AnteHandler chain for CheckTx mode.
// It validates signatures, format, gas limits, and fees (for priority) but
// does not deduct fees or increment sequences; those are handled by the
// full handler during PrepareProposal/FinalizeBlock.
func NewMinimalAnteHandler(options HandlerOptions) (sdk.AnteHandler, error) {
	if options.AccountKeeper == nil {
		return nil, errors.Wrap(sdkerrors.ErrLogic, "account keeper is required for minimal ante handler")
	}
	if options.SignModeHandler == nil {
		return nil, errors.Wrap(sdkerrors.ErrLogic, "sign mode handler is required for minimal ante handler")
	}
	if options.OPChildKeeper == nil {
		return nil, errors.Wrap(sdkerrors.ErrLogic, "opchild keeper is required for minimal ante handler")
	}
	if options.IBCkeeper == nil {
		return nil, errors.Wrap(sdkerrors.ErrLogic, "IBC keeper is required for minimal ante handler")
	}

	sigGasConsumer := options.SigGasConsumer
	if sigGasConsumer == nil {
		sigGasConsumer = sigverify.DefaultSigVerificationGasConsumer
	}

	txFeeChecker := options.TxFeeChecker
	if txFeeChecker == nil {
		return nil, errors.Wrap(sdkerrors.ErrLogic, "tx fee checker is required for minimal ante handler")
	}

	anteDecorators := []sdk.AnteDecorator{
		ante.NewSetUpContextDecorator(),
		ante.NewExtensionOptionsDecorator(options.ExtensionOptionChecker),
		moveante.NewGasPricesDecorator(),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(options.AccountKeeper),
		ante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
		initiaante.NewCheckFeeDecorator(txFeeChecker), // validate fee + set priority, no deduction
		ante.NewSetPubKeyDecorator(options.AccountKeeper),
		ante.NewValidateSigCountDecorator(options.AccountKeeper),
		ante.NewSigGasConsumeDecorator(options.AccountKeeper, sigGasConsumer),
		sigverify.NewSigVerificationDecorator(options.AccountKeeper, options.SignModeHandler),
		// no IncrementSequenceDecorator here since mempool tracks nonces
		ibcante.NewRedundantRelayDecorator(options.IBCkeeper),
		opchildante.NewRedundantBridgeDecorator(options.OPChildKeeper),
	}

	return sdk.ChainAnteDecorators(anteDecorators...), nil
}
