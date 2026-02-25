package app

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	cosmosante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	opchildante "github.com/initia-labs/OPinit/x/opchild/ante"
	"github.com/initia-labs/initia/abcipp"
	initiaante "github.com/initia-labs/initia/app/ante"

	appante "github.com/initia-labs/minimove/app/ante"
)

func (app *MinitiaApp) setupABCIPP(mempoolMaxTxs int) (
	sdkmempool.Mempool,
	sdk.AnteHandler,
	sdk.PrepareProposalHandler,
	sdk.ProcessProposalHandler,
	abcipp.CheckTx,
	error,
) {

	feeChecker := opchildante.NewMempoolFeeChecker(app.OPChildKeeper).CheckTxFeeWithMinGasPrices
	feeCheckerWrapper := func(ctx sdk.Context, tx sdk.Tx) (sdk.Coins, int64, error) {
		freeFeeChecker := func() bool {
			feeTx, ok := tx.(sdk.FeeTx)
			if !ok {
				return false
			}

			whitelist, err := app.OPChildKeeper.FeeWhitelist(ctx)
			if err != nil {
				return false
			}

			payer, err := app.ac.BytesToString(feeTx.FeePayer())
			if err != nil {
				return false
			}

			var granter string
			if feeTx.FeeGranter() != nil {
				granter, err = app.ac.BytesToString(feeTx.FeeGranter())
				if err != nil {
					return false
				}
			}

			for _, addr := range whitelist {
				if addr == payer || addr == granter {
					return true
				}
			}

			return false
		}

		if !freeFeeChecker() {
			return feeChecker(ctx, tx)
		}

		// return fee without fee check
		feeTx, ok := tx.(sdk.FeeTx)
		if !ok {
			return nil, 0, errors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
		}

		return feeTx.GetFee(), 1 /* FIFO */, nil
	}

	handlerOpts := appante.HandlerOptions{
		HandlerOptions: cosmosante.HandlerOptions{
			AccountKeeper:   app.AccountKeeper,
			BankKeeper:      app.BankKeeper,
			FeegrantKeeper:  app.FeeGrantKeeper,
			SignModeHandler: app.txConfig.SignModeHandler(),
			TxFeeChecker:    feeCheckerWrapper,
		},
		IBCkeeper:     app.IBCKeeper,
		Codec:         app.appCodec,
		OPChildKeeper: app.OPChildKeeper,
	}

	fullHandler, err := appante.NewAnteHandler(handlerOpts)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	minimalHandler, err := appante.NewMinimalAnteHandler(handlerOpts)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	anteHandler := initiaante.NewDualAnteHandler(minimalHandler, fullHandler)

	mempool := abcipp.NewPriorityMempool(
		abcipp.PriorityMempoolConfig{
			MaxTx:       mempoolMaxTxs,
			AnteHandler: fullHandler, // cleaning worker uses full handler
			Tiers:       []abcipp.Tier{},
		}, app.txConfig.TxEncoder(),
	)
	mempool.SetAccountKeeper(app.AccountKeeper)

	// start mempool cleaning worker
	mempool.StartCleaningWorker(app.BaseApp, app.AccountKeeper, abcipp.DefaultMempoolCleaningInterval)

	proposalHandler := abcipp.NewProposalHandler(
		app.Logger(),
		app.txConfig.TxDecoder(),
		app.txConfig.TxEncoder(),
		mempool,
		fullHandler, // proposal handler uses full handler
	)

	checkTxHandler := abcipp.NewCheckTxHandler(
		app.Logger(),
		app.BaseApp,
		mempool,
		app.txConfig.TxDecoder(),
		app.BaseApp.CheckTx,
		feeCheckerWrapper,
	)

	return mempool, anteHandler, proposalHandler.PrepareProposalHandler(), proposalHandler.ProcessProposalHandler(), checkTxHandler.CheckTx, nil
}
