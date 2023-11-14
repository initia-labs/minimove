package main

import (
	"encoding/json"
	"fmt"

	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/version"
	cosmosgenutil "github.com/cosmos/cosmos-sdk/x/genutil"
	cosmostypes "github.com/cosmos/cosmos-sdk/x/genutil/types"

	"github.com/initia-labs/OPinit/x/opchild/client/cli"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
)

// AddGenesisValidatorCmd builds the application's gentx command.
func AddGenesisValidatorCmd(mbm module.BasicManager, txEncCfg client.TxEncodingConfig, genBalIterator cosmostypes.GenesisBalancesIterator, defaultNodeHome string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-genesis-validator [key_name]",
		Short: "Add a genesis validator",
		Args:  cobra.ExactArgs(1),
		Long: fmt.Sprintf(`Add a genesis validator with the key in the Keyring referenced by a given name.
		A Bech32 consensus pubkey may optionally be provided.

Example:
$ %s add-genesis-validator my-key-name --home=/path/to/home/dir --keyring-backend=os --chain-id=test-chain-1
`, version.AppName,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			serverCtx := server.GetServerContextFromCmd(cmd)
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			cdc := clientCtx.Codec

			config := serverCtx.Config
			config.SetRoot(clientCtx.HomeDir)

			genDoc, err := tmtypes.GenesisDocFromFile(config.GenesisFile())
			if err != nil {
				return errors.Wrapf(err, "failed to read genesis doc file %s", config.GenesisFile())
			}

			var genesisState map[string]json.RawMessage
			if err = json.Unmarshal(genDoc.AppState, &genesisState); err != nil {
				return errors.Wrap(err, "failed to unmarshal genesis state")
			}

			_ /*nodeId*/, valPubKey, err := cosmosgenutil.InitializeNodeValidatorFiles(serverCtx.Config)
			if err != nil {
				return errors.Wrap(err, "failed to initialize node validator files")
			}

			// read --pubkey, if empty take it from priv_validator.json
			if pkStr, _ := cmd.Flags().GetString(cli.FlagPubKey); pkStr != "" {
				if err := clientCtx.Codec.UnmarshalInterfaceJSON([]byte(pkStr), &valPubKey); err != nil {
					return errors.Wrap(err, "failed to unmarshal validator public key")
				}
			}

			name := args[0]
			key, err := clientCtx.Keyring.Key(name)
			if err != nil {
				return errors.Wrapf(err, "failed to fetch '%s' from the keyring", name)
			}

			moniker := config.Moniker
			if m, _ := cmd.Flags().GetString(cli.FlagMoniker); m != "" {
				moniker = m
			}

			addr, err := key.GetAddress()
			if err != nil {
				return err
			}
			valAddr := sdk.ValAddress(addr)

			validator, err := opchildtypes.NewValidator(valAddr, valPubKey, moniker)
			if err != nil {
				return err
			}

			opchildState := opchildtypes.GetGenesisStateFromAppState(cdc, genesisState)
			opchildState.Validators = append((*opchildState).Validators, validator)
			if opchildState.Params.BridgeExecutor == "" {
				opchildState.Params.BridgeExecutor = addr.String()
			}

			genesisState[opchildtypes.ModuleName] = cdc.MustMarshalJSON(opchildState)

			if err = mbm.ValidateGenesis(cdc, txEncCfg, genesisState); err != nil {
				return errors.Wrap(err, "failed to validate genesis state")
			}

			genDoc.AppState, err = json.MarshalIndent(genesisState, "", "  ")
			if err != nil {
				return errors.Wrap(err, "failed to marshal genesis state")
			}

			if err = cosmosgenutil.ExportGenesisFile(genDoc, config.GenesisFile()); err != nil {
				return errors.Wrap(err, "Failed to export genesis file")
			}

			return nil
		},
	}

	cmd.Flags().String(flags.FlagHome, defaultNodeHome, "The application home directory")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
