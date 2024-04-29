package launchtools

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"os"
	"time"
)

type Input struct {
	L2Config        L2Config             `json:"l2_config"`
	L1Config        L1Config             `json:"l1_config"`
	OpBridge        OpBridge             `json:"op_bridge"`
	SystemKeys      SystemKeys           `json:"system_keys"`
	UserKeys        []UserKeys           `json:"user_keys"`
	GenesisAccounts []AccountWithBalance `json:"genesis_accounts"`
}

func (input Input) FromFile(path string) (Manifest, error) {
	bz, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to read file: %s", path))
	}

	ret := new(Input)
	if err := json.Unmarshal(bz, ret); err != nil {
		return nil, err
	}
	return ret, nil
}

type L2Config struct {
	ChainID       string `json:"chain_id"`
	Denom         string `json:"denom"`
	Moniker       string `json:"moniker"`
	AccountPrefix string `json:"account_prefix"`
	GasPrices     string `json:"gas_prices"`
}

type OpBridge struct {
	SubmissionStartTime time.Time     `json:"submission_start_time"`
	SubmitTarget        string        `json:"submit_target"`
	SubmissionInterval  time.Duration `json:"submission_interval"`
	FinalizationPeriod  time.Duration `json:"finalization_period"`
}

type L1Config struct {
	ChainID       string `json:"chain_id"`
	FaucetURL     string `json:"faucet_url"`
	RPCURL        string `json:"rpc_url"`
	RestURL       string `json:"rest_url"`
	GrpcURL       string `json:"grpc_url"`
	WsURL         string `json:"ws_url"`
	AccountPrefix string `json:"account_prefix"`
	GasPrices     string `json:"gas_prices"`
}

type Account struct {
	Address  string `json:"address"`
	Mnemonic string `json:"mnemonic"`
}

type AccountWithBalance struct {
	Account
	Coins string `json:"coins"`
}

type SystemKeys struct {
	Validator  Account `json:"validator"`
	Executor   Account `json:"executor"`
	Output     Account `json:"output"`
	Challenger Account `json:"challenger"`
	Submitter  Account `json:"submitter"`
	Relayer    Account `json:"relayer"`
}

type UserKeys struct {
	Name     string `json:"name"`
	Address  string `json:"address"`
	Mnemonic string `json:"mnemonic"`
}

func (i Input) Validate() error {
	return nil
}

func TestInput() Input {
	var testInput = `
{
  "l2_config": {
    "chain_id": "newmetric-test-minimove-100",
    "denom": "umin",
    "moniker": "sequencer",
    "account_prefix": "init",
    "gas_prices": "0.025umin"
  },
  "op_bridge": {
    "submission_start_time": "2024-04-03T18:26:05.713Z"
  },
  "l1_config": {
    "chain_id": "mahalo-2",
    "faucet_url": "https://faucet-api.mahalo-2.initia.xyz/claim-with-ip",
    "rpc_url": "http://34.87.121.251:26657",
    "rest_url": "http://34.87.121.251:1317",
    "grpc_url": "34.87.121.251:9090",
    "ws_url": "ws://34.87.121.251:26657/websocket",
    "account_prefix": "init",
    "gas_prices": "0.025uinit"
  },
  "system_keys": {
    "validator": {
      "address": "init15n9xp8vapc48jrq85gklla9e9d74w47fdx9hh7",
      "mnemonic": "lady artwork observe slab labor depth response autumn debate humor image ugly method awful lyrics bomb accident able quiz junk raccoon limit poet weird"
    },
    "executor": {
      "address": "init199d062ns49qagtct5gd9r7rjqpwvmqap0mhmlh",
      "mnemonic": "question cricket photo cloud just impose cherry session desk hurry over rude describe merit scatter favorite few avocado flush wrap habit kiss wrist firm"
    },
    "output": {
      "address": "init1cc2l0jdghgk54fcm5lhxx0r3apggdr5kryadja",
      "mnemonic": "sunny genre clean glue recycle grocery coil dog float two vacuum reunion bar chest service success april invite crumble wisdom secret love issue vague"
    },
    "challenger": {
      "address": "init1pp3r738z47cchyhz95h6wzct073u62hwz4mp3a",
      "mnemonic": "anchor kite safe artist olympic candy letter stage cricket artwork possible gap insect piano energy shoe riot original worry item claim cook bulb garment"
    },
    "submitter": {
      "address": "init10kwuyc9zeuj735m8f8lwg7q4gtvdp4jcn33hj8",
      "mnemonic": "section proof short great custom affair benefit arrest topic year damage lake citizen maximum term script maze axis image kiwi exotic brick trend shine"
    },
    "relayer": {
      "address": "init1wv7uvp5gnrj72zd7gwhvyxck6tkdtjp6wvvsey",
      "mnemonic": "slim police bamboo task icon mimic tenant cancel inform clutch raise reform cigar believe gossip vendor east liar slush confirm before lava profit rally"
    }
  },
  "user_keys": [
    {
      "name": "user1",
      "address": "init1uex2ecg2knu95y4d0zgz2thn99ac4ql0ststd3",
      "mnemonic": "ocean weekend wheel unfair blossom energy valley kitchen wood right solar chef crucial pact correct amused select stereo kick live pond top creek gossip"
    },
    {
      "name": "user2",
      "address": "init17ay45t0dvp5cj2uh3k0xzug7lup3f394xru6kx",
      "mnemonic": "wide hundred stem glare face pink melt right sniff trouble smoke mutual lizard act repeat ivory special rude gloom control opera vintage empower primary"
    }
  ],
  "genesis_accounts": [
    {
      "address": "init15n9xp8vapc48jrq85gklla9e9d74w47fdx9hh7",
      "amount": "190000000000",
      "denom": "umin"
    },
    {
      "address": "init199d062ns49qagtct5gd9r7rjqpwvmqap0mhmlh",
      "amount": "190000000000",
      "denom": "umin"
    },
    {
      "address": "init1cc2l0jdghgk54fcm5lhxx0r3apggdr5kryadja",
      "amount": "190000000000",
      "denom": "umin"
    },
    {
      "address": "init1pp3r738z47cchyhz95h6wzct073u62hwz4mp3a",
      "amount": "190000000",
      "denom": "umin"
    },
    {
      "address": "init1wv7uvp5gnrj72zd7gwhvyxck6tkdtjp6wvvsey",
      "amount": "190000000000",
      "denom": "umin"
    },
    {
      "address": "init1uex2ecg2knu95y4d0zgz2thn99ac4ql0ststd3",
      "amount": "190000000000",
      "denom": "umin"
    },
    {
      "address": "init17ay45t0dvp5cj2uh3k0xzug7lup3f394xru6kx",
      "amount": "190000000000",
      "denom": "umin"
    }
  ]
}`

	input := Input{}
	if err := json.Unmarshal([]byte(testInput), &input); err != nil {
		panic(err)
	}

	return input
}
