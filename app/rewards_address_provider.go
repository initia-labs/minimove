package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	buildertypes "github.com/skip-mev/pob/x/builder/types"
)

var _ buildertypes.RewardsAddressProvider = (*RewardsAddressProvider)(nil)

// NewRewardsAddressProvider returns a new RewardsAddressProvider from a staking + distribution keeper
func NewRewardsAddressProvider(feeCollectorName string) *RewardsAddressProvider {
	return &RewardsAddressProvider{
		feeCollectorName: feeCollectorName,
	}
}

// RewardsAddressProvider implements the x/builder's RewardsAddressProvider interface. It is used
// to determine the address to which the rewards from the most recent block's auction are sent.
type RewardsAddressProvider struct {
	feeCollectorName string
}

// GetRewardsAddress returns the address of the proposer of the previous block
func (rap *RewardsAddressProvider) GetRewardsAddress(ctx sdk.Context) (sdk.AccAddress, error) {
	return authtypes.NewModuleAddress(rap.feeCollectorName), nil
}
