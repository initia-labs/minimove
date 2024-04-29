package steps

import (
	"context"
	"cosmossdk.io/log"
	"encoding/json"
	"fmt"
	relayercmd "github.com/cosmos/relayer/v2/cmd"
	launchtools "github.com/initia-labs/minimove/contrib/launchtools"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"os"
	"path"
	"reflect"
)

func EstablishIBCChannels(input launchtools.Input) launchtools.LauncherStepFunc {
	relayerPath, err := os.MkdirTemp("", RelayerPathTemp)
	if err != nil {
		panic(err)
	}

	runLifecycle := lifecycle(
		initializeConfig,
		initializeChains(input, relayerPath),
		initializePaths(input, relayerPath),
		initializeRelayerKeyring(input, relayerPath),
		link,
	)

	return func(ctx *launchtools.LauncherContext) error {
		if !ctx.IsAppInitialized() {
			return errors.New("app is not initialized")
		}

		// Step 0. initialize relayer config
		return runLifecycle(NewRelayer(ctx.Cmd.Context(), relayerPath, ctx.ServerCtx.Logger))
	}
}

// -------------------------------
func initializeConfig(r *Relayer) error {
	return r.run([]string{"config", "init"})
}

func initializeChains(input launchtools.Input, basePath string) func(*Relayer) error {
	// ChainConfig is a struct that represents the configuration of a chain
	// cosmos/relayer specific
	type ChainConfigValue struct {
		Key            string  `json:"key"`
		ChainID        string  `json:"chain-id"`
		RPCAddr        string  `json:"rpc-addr"`
		GrpcAddr       string  `json:"grpc-addr"`
		AccountPrefix  string  `json:"account-prefix"`
		KeyringBackend string  `json:"keyring-backend"`
		GasAdjustment  float64 `json:"gas-adjustment"`
		GasPrices      string  `json:"gas-prices"`
		Debug          bool    `json:"debug"`
		Timeout        string  `json:"timeout"`
		OutputFormat   string  `json:"output-format"`
		SignMode       string  `json:"sign-mode"`
	}

	type ChainConfig struct {
		Type  string           `json:"type"`
		Value ChainConfigValue `json:"value"`
	}

	var chainConfigs = [2]ChainConfig{
		{
			Type: "cosmos",
			Value: ChainConfigValue{
				Key:            RelayerKeyName,
				ChainID:        input.L1Config.ChainID,
				RPCAddr:        input.L1Config.RPCURL,
				GrpcAddr:       input.L1Config.GrpcURL,
				AccountPrefix:  input.L1Config.AccountPrefix,
				KeyringBackend: KeyringBackend,
				GasAdjustment:  1.5,
				GasPrices:      input.L1Config.GasPrices,
				Debug:          true,
				Timeout:        "160s",
				OutputFormat:   "json",
				SignMode:       "direct",
			},
		},
		{
			Type: "cosmos",
			Value: ChainConfigValue{
				Key:            RelayerKeyName,
				ChainID:        input.L2Config.ChainID,
				RPCAddr:        "http://localhost:26657",
				GrpcAddr:       "localhost:9090",
				AccountPrefix:  input.L2Config.AccountPrefix,
				KeyringBackend: KeyringBackend,
				GasAdjustment:  1.5,
				GasPrices:      input.L2Config.GasPrices,
				Debug:          true,
				Timeout:        "160s",
				OutputFormat:   "json",
				SignMode:       "direct",
			},
		},
	}

	for i, chainConfig := range chainConfigs {
		bz, err := json.MarshalIndent(chainConfig, "", " ")
		if err != nil {
			panic(errors.New("failed to create chain config"))
		}

		pathName := fmt.Sprintf("chain%d", i)
		fileName := fmt.Sprintf("%s/%s.json", basePath, pathName)

		if err := os.WriteFile(fileName, bz, 0644); err != nil {
			panic(errors.New("failed to write chain config"))
		}
	}

	return func(r *Relayer) error {
		r.logger.Info("initializing chains for relayer...",
			"chains-len", len(chainConfigs),
			"chain-0", chainConfigs[0].Value.ChainID,
			"chain-1", chainConfigs[1].Value.ChainID,
		)

		for i, chainConfig := range chainConfigs {
			if err := r.run([]string{
				"chains",
				"add",
				"--file",
				path.Join(basePath, fmt.Sprintf("chain%d.json", i)),
				chainConfig.Value.ChainID,
			}); err != nil {
				return err
			}
		}

		return nil
	}
}

func initializePaths(input launchtools.Input, basePath string) func(*Relayer) error {
	var pathConfig struct {
		Src struct {
			ChainID string `json:"chain-id"`
		} `json:"src"`
		Dst struct {
			ChainID string `json:"chain-id"`
		} `json:"dst"`
		SrcChannelFilter struct {
			Rule        string   `json:"rule"`
			ChannelList []string `json:"channel-list"`
		} `json:"src-channel-filter"`
	}

	pathConfig.Src.ChainID = input.L2Config.ChainID
	pathConfig.Dst.ChainID = input.L1Config.ChainID

	pathConfigJSON, err := json.Marshal(pathConfig)
	if err != nil {
		panic(errors.New("failed to create path config"))
	}

	if err := os.WriteFile(fmt.Sprintf("%s/paths.json", basePath), pathConfigJSON, 0644); err != nil {
		panic(errors.New("failed to write path config"))
	}

	return func(r *Relayer) error {
		r.logger.Info("initializing paths for relayer...",
			"src-chain", pathConfig.Src.ChainID,
			"dst-chain", pathConfig.Dst.ChainID,
		)

		return r.run([]string{
			"paths",
			"add",
			input.L2Config.ChainID,
			input.L1Config.ChainID,
			RelayerPathName,
			"-f",
			fmt.Sprintf("%s/paths.json", basePath),
		})
	}
}

func initializeRelayerKeyring(input launchtools.Input, basePath string) func(*Relayer) error {
	relayerKeyFromInput := reflect.ValueOf(input.SystemKeys).FieldByName(RelayerKeyName)
	if !relayerKeyFromInput.IsValid() {
		panic(errors.New("relayer key not found in input"))
	}

	relayerKey := relayerKeyFromInput.Interface().(launchtools.Account)

	return func(r *Relayer) error {
		r.logger.Info("initializing keyring for relayer...",
			"key-name", RelayerKeyName,
		)

		for _, chainName := range []string{
			input.L2Config.ChainID,
			input.L1Config.ChainID,
		} {
			if err := r.run([]string{
				"keys",
				"restore",
				chainName,
				RelayerKeyName,
				relayerKey.Mnemonic,
			}); err != nil {
				return err
			}
		}

		return nil
	}
}

func link(r *Relayer) error {
	r.logger.Info("linking chains for relayer...")
	return r.run([]string{
		"tx",
		"link",
		RelayerPathName,
	})
}

func establishClients(r *Relayer) error {
	r.logger.Info("establishing clients for relayer...")
	return r.run([]string{
		"tx",
		"clients",
		RelayerPathName,
	})
}

func establishConnections(r *Relayer) error {
	r.logger.Info("establishing connections for relayer...")
	return r.run([]string{
		"tx",
		"connections",
		RelayerPathName,
	})
}

func establishChannels(r *Relayer) error {
	r.logger.Info("establishing channels for relayer...")
	return r.run([]string{
		"tx",
		"channels",
		RelayerPathName,
	})
}

// -------------------------------
// lifecycle manager
func lifecycle(lfc ...func(*Relayer) error) func(*Relayer) error {
	return func(rly *Relayer) error {
		for i, lf := range lfc {
			if err := lf(rly); err != nil {
				return errors.Wrapf(err, "failed to run lifecycle during ibc step %d", i+1)
			}
		}

		return nil
	}
}

// Relayer cmd proxy caller
type Relayer struct {
	// home is Relayer home directory
	home   string
	zap    *zap.Logger
	logger log.Logger
	ctx    context.Context
}

func NewRelayer(
	ctx context.Context,
	home string,
	logger log.Logger,
) *Relayer {
	return &Relayer{
		home:   home,
		zap:    zap.L(),
		logger: logger,
		ctx:    ctx,
	}
}

func (r *Relayer) run(args []string) error {
	cmd := relayercmd.NewRootCmd(nil)
	cmd.SilenceUsage = true

	cmd.SetArgs(append(args, []string{"--home", r.home, "--debug"}...))
	return cmd.ExecuteContext(context.Background())
}
