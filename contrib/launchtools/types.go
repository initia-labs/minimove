package launchtools

import (
	"cosmossdk.io/log"
	cosmosdb "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/initia-labs/initia/app/params"
	moveconfig "github.com/initia-labs/initia/x/move/config"
	minitiaapp "github.com/initia-labs/minimove/app"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"strings"
	"sync"
)

type LauncherStepFuncFactory[Manifest any] func(Manifest) LauncherStepFunc
type LauncherStepFunc func(ctx *LauncherContext) error
type LauncherCleanupFunc func() error

type LauncherContext struct {
	mtx *sync.Mutex

	app        servertypes.Application
	cleanupFns []LauncherCleanupFunc

	Dry           bool
	AppCreator    servertypes.AppCreator
	Home          string
	BasicManager  module.BasicManager
	ClientContext *client.Context
	ServerCtx     *server.Context

	Cmd *cobra.Command
}

func NewLauncher(
	home string,
	clientCtx *client.Context,
	serverCtx *server.Context,
	basicManager module.BasicManager,
	appCreator servertypes.AppCreator,
	encodingConfig params.EncodingConfig,
	cmd *cobra.Command,
) *LauncherContext {

	kr, err := keyring.New("minitia", keyring.BackendTest, home, nil, encodingConfig.Codec)
	if err != nil {
		panic("failed to create keyring")
	}

	nextClientCtx := clientCtx.WithKeyring(kr)
	serverCtx.Config.SetRoot(home)

	return &LauncherContext{
		mtx:           new(sync.Mutex),
		Dry:           false,
		Home:          home,
		ClientContext: &nextClientCtx,
		ServerCtx:     serverCtx,
		BasicManager:  basicManager,
		AppCreator:    appCreator,
		Cmd:           cmd,
	}
}

func (l *LauncherContext) AddCleanupFn(fn LauncherCleanupFunc) {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	l.cleanupFns = append(l.cleanupFns, fn)
}

func (l *LauncherContext) Cleanup() error {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	var errs []string
	for _, fn := range l.cleanupFns {
		if err := fn(); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return errors.Errorf("cleanup finished with errors: %s", strings.Join(errs, "\n"))
	}

	return nil
}

func (l *LauncherContext) SetApp(app servertypes.Application) servertypes.Application {
	l.app = app
	return app
}

func (l *LauncherContext) GetApp() servertypes.Application {
	return l.app
}

func (l *LauncherContext) IsAppInitialized() bool {
	return l.app != nil
}

// NewLauncherForTesting creates a launcher context from default values for testing purposes.
func NewLauncherForTesting(dir string) *LauncherContext {
	sdkConfig := sdk.GetConfig()
	sdkConfig.SetCoinType(minitiaapp.CoinType)

	accountPubKeyPrefix := minitiaapp.AccountAddressPrefix + "pub"
	validatorAddressPrefix := minitiaapp.AccountAddressPrefix + "valoper"
	validatorPubKeyPrefix := minitiaapp.AccountAddressPrefix + "valoperpub"
	consNodeAddressPrefix := minitiaapp.AccountAddressPrefix + "valcons"
	consNodePubKeyPrefix := minitiaapp.AccountAddressPrefix + "valconspub"

	sdkConfig.SetBech32PrefixForAccount(minitiaapp.AccountAddressPrefix, accountPubKeyPrefix)
	sdkConfig.SetBech32PrefixForValidator(validatorAddressPrefix, validatorPubKeyPrefix)
	sdkConfig.SetBech32PrefixForConsensusNode(consNodeAddressPrefix, consNodePubKeyPrefix)
	sdkConfig.SetAddressVerifier(minitiaapp.VerifyAddressLen())
	//sdkConfig.Seal()

	encodingConfig := minitiaapp.MakeEncodingConfig()

	kr, err := keyring.New("minitia", keyring.BackendTest, dir, nil, encodingConfig.Codec)
	if err != nil {
		panic("failed to create keyring")
	}

	// Configure the viper instance
	initClientCtx := client.Context{}.
		WithCodec(encodingConfig.Codec).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		WithLegacyAmino(encodingConfig.Amino).
		WithInput(os.Stdin).
		WithAccountRetriever(types.AccountRetriever{}).
		WithHomeDir(dir).
		WithViper(minitiaapp.EnvPrefix).

		// use in-memory keyring
		WithKeyring(kr)

	// pre-configure app
	app := minitiaapp.NewMinitiaApp(
		log.NewNopLogger(),
		cosmosdb.NewMemDB(),
		nil,
		false,
		moveconfig.DefaultMoveConfig(),
		viper.New(), // use whatever
	)

	return &LauncherContext{
		app:           app,
		Cmd:           &cobra.Command{},
		Dry:           true,
		Home:          dir,
		ClientContext: &initClientCtx,
		ServerCtx:     server.NewDefaultContext(),
		BasicManager:  minitiaapp.BasicManager(),
	}
}
