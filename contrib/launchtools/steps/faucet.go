package steps

import (
	"bytes"
	"encoding/json"
	launchtools "github.com/initia-labs/minimove/contrib/launchtools"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"strings"
	"time"
)

const L1Denom = "uinit"
const HeaderFaucetRequest = "Bearer bWFoYWxvOm9wZW5zZXNhbWk="
const IntervalFaucetWait = 10 * time.Second
const IntervalFaucetTimeout = 10 * time.Second

// RequestFaucetOnL1 requests l1 tokens for certain system keys.
// This is to allow system keys (i.e. executor) to have enough funds to post transactions on L1.
func RequestFaucetOnL1(input launchtools.Input) launchtools.LauncherStepFunc {
	if input.OpBridge.SubmitTarget == "" {
		panic("missing input.OpBridge.SubmitTarget")
	}

	if input.L1Config.FaucetURL == "" {
		panic("missing input.L1Config.FaucetURL")
	}

	if input.OpBridge.SubmitTarget == "l1" && input.SystemKeys.Submitter.Address == "" {
		panic("missing input.SystemKeys.Submitter.Address")
	}

	return func(ctx *launchtools.LauncherContext) error {
		log := ctx.ServerCtx.Logger

		// request faucet for these addresses
		faucetReceivers := []string{
			input.SystemKeys.Executor.Address,
			input.SystemKeys.Output.Address,
			input.SystemKeys.Relayer.Address,
		}

		// If submit target is L1, request faucet for submitter
		if input.OpBridge.SubmitTarget == "l1" {
			faucetReceivers = append(faucetReceivers, input.SystemKeys.Submitter.Address)
		}

		log.Info("requesting faucet for system keys",
			"addresses", strings.Join(faucetReceivers, ","),
		)

		// create http client
		httpClient := http.Client{
			Timeout: IntervalFaucetTimeout,
		}
		for _, address := range faucetReceivers {
			log.Info("requesting faucet for address",
				"address", address,
			)
			resp, err := requestFaucet(&httpClient, input.L1Config.FaucetURL, address)
			if err != nil {
				return errors.Wrapf(err, "failed to request faucet for address: %v", address)
			}

			if resp.StatusCode != http.StatusOK {
				return errors.Errorf("failed to request faucet for address: %v, status code: %v", address, resp.StatusCode)
			}

			bz, err := io.ReadAll(resp.Body)
			log.Info("faucet response",
				"response", string(bz),
			)
		}

		time.Sleep(IntervalFaucetWait)

		return nil
	}
}

func requestFaucet(client *http.Client, faucetURL string, address string) (*http.Response, error) {
	bz, err := json.Marshal(struct {
		Address string `json:"address"`
		Denom   string `json:"denom"`
	}{
		Address: address,
		Denom:   L1Denom,
	})

	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal request body for address: %v", address)
	}

	req, err := http.NewRequest(http.MethodPost, faucetURL, bytes.NewReader(bz))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create http request")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", HeaderFaucetRequest)

	return client.Do(req)
}
