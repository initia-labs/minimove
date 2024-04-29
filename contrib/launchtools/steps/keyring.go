package steps

import (
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/initia-labs/minimove/contrib/launchtools"
	"github.com/pkg/errors"
	"reflect"
)

// InitializeKeyring adds all system keys to the keyring
func InitializeKeyring(input launchtools.Input) launchtools.LauncherStepFunc {
	// keyAdders is a slice of functions that add keys to the keyring
	var keyAdders = make([]func(keyring.Keyring, log.Logger) error, 0)

	// use reflect to iterate over all system keys, since it's a struct
	// also this is future-proof in case any new system key is added
	systemKeys := reflect.ValueOf(input.SystemKeys)
	for i := 0; i < systemKeys.NumField(); i++ {
		fieldName := systemKeys.Type().Field(i).Name
		k, ok := systemKeys.Field(i).Interface().(launchtools.Account)
		if !ok {
			panic(errors.New("systemKeys must be of type launcher.Account"))
		}

		keyAdders = append(keyAdders, func(kr keyring.Keyring, l log.Logger) error {
			l.Info("adding system key",
				"key-name", fieldName,
				"address", k.Address,
			)

			accountRecord, err := kr.NewAccount(
				fieldName,
				k.Mnemonic,
				keyring.DefaultBIP39Passphrase,
				sdk.GetConfig().GetFullBIP44Path(),
				hd.Secp256k1,
			)

			// keyring addition must be successful
			if err != nil {
				return errors.Wrapf(err, "failed to add key %s to keyring", fieldName)
			}

			// check if added key is the same as the one supplied in the input
			addr, addrErr := accountRecord.GetAddress()
			if addrErr != nil {
				return errors.Wrapf(addrErr, "failed to get address for key %s", fieldName)
			}

			if addr.String() != k.Address {
				return errors.Errorf("address mismatch for key %s, keyring=%s, input=%s", fieldName, addr.String(), k.Address)
			}

			return nil
		})
	}

	return func(ctx *launchtools.LauncherContext) error {
		ctx.ServerCtx.Logger.Info("adding system keys to keyring...",
			"keyring-backend", ctx.ClientContext.Keyring.Backend(),
		)

		for i, keyAdder := range keyAdders {
			if err := keyAdder(ctx.ClientContext.Keyring, ctx.ServerCtx.Logger); err != nil {
				return errors.Wrapf(err, "failed to add key %d", i)
			}
		}

		return nil
	}
}
