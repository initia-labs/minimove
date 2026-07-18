package v1_1_13

import (
	"context"
	"encoding/binary"
	"fmt"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/initia-labs/minimove/app/upgrades"

	vmtypes "github.com/initia-labs/movevm/types"
)

const upgradeName = "v1.1.13"

// strat-1 h2626106 stale-read incident (iavl #1142). The ghost merge materialized modify
// on a non-existent key, adding a physical entry to the strat `positions` table  without
// passing through table.add, so the length counter in PositionStore undercounts the
// physical entries by exactly one. No Move-level operation can close that gap
// (add/remove move counter and entries in lockstep), hence this node-side migration.
// It bumps the stored counter by one (+1, never an absolute value as the counter keeps
// moving as positions open and close, only the offset is constant).
const (
	stratChainID = "strat-1"
	stratModule  = "0xd49da8a8c29c1294b98fcb119ae3bdc1cf697ac2b42d63caed608b07941ce111"

	// expected table handles inside PositionStore, asserted before writing so the
	// migration aborts (and the upgrade halts) if the resource is not byte-shaped
	// exactly as analyzed
	positionIDMapHandle = "0xf44b3d6f36d59efeff36922a43d6b48109ba2cd6c754dbbd3348e68b962684ff"
	positionsHandle     = "0x8ebf15a319d93ea00a671be70f6cfeb2fb50388fdefb0f0309eafcb9f2c89d07"

	// PositionStore BCS layout: last_position_id u64 | id_map {handle addr32, length u64}
	// | positions {handle addr32, length u64}
	positionStoreLen  = 8 + (32 + 8) + (32 + 8)
	idMapHandleOff    = 8
	positionsHandleOf = 8 + 32 + 8
	positionsLenOff   = 8 + 32 + 8 + 32
)

// RegisterUpgradeHandlers returns upgrade handlers
func RegisterUpgradeHandlers(app upgrades.MinitiaApp) {
	app.GetUpgradeKeeper().SetUpgradeHandler(
		upgradeName,
		func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			sdkCtx := sdk.UnwrapSDKContext(ctx)
			if sdkCtx.ChainID() != stratChainID {
				// the binary ships to every rollup; the migration is strat-1 only
				return vm, nil
			}

			if err := fixStratPositionsTableLength(ctx, app); err != nil {
				return nil, err
			}

			return vm, nil
		},
	)
}

func fixStratPositionsTableLength(ctx context.Context, app upgrades.MinitiaApp) error {
	stratAddr, err := vmtypes.NewAccountAddress(stratModule)
	if err != nil {
		return err
	}

	structTag := vmtypes.StructTag{
		Address: stratAddr,
		Module:  "position",
		Name:    "PositionStore",
	}

	moveKeeper := app.GetMoveKeeper()
	bz, err := moveKeeper.GetResourceBytes(ctx, stratAddr, structTag)
	if err != nil {
		return fmt.Errorf("read PositionStore: %w", err)
	}

	patched, err := patchPositionsLength(bz)
	if err != nil {
		return err
	}

	return moveKeeper.SetResource(ctx, stratAddr, structTag, patched)
}

// patchPositionsLength does the actual byte surgery; split out and exported to be
// testable against a byte-copy of the real mainnet resource.
func patchPositionsLength(bz []byte) ([]byte, error) {
	if len(bz) != positionStoreLen {
		return nil, fmt.Errorf("unexpected PositionStore size: got %d, want %d", len(bz), positionStoreLen)
	}

	idMapAddr, err := vmtypes.NewAccountAddress(positionIDMapHandle)
	if err != nil {
		return nil, err
	}
	positionsAddr, err := vmtypes.NewAccountAddress(positionsHandle)
	if err != nil {
		return nil, err
	}

	if gotIDMap := vmtypes.AccountAddress(bz[idMapHandleOff : idMapHandleOff+32]); gotIDMap != idMapAddr {
		return nil, fmt.Errorf("id_map handle mismatch: got %s", gotIDMap)
	}
	if gotPositions := vmtypes.AccountAddress(bz[positionsHandleOf : positionsHandleOf+32]); gotPositions != positionsAddr {
		return nil, fmt.Errorf("positions handle mismatch: got %s", gotPositions)
	}

	patched := make([]byte, len(bz))
	copy(patched, bz)

	length := binary.LittleEndian.Uint64(patched[positionsLenOff:])
	binary.LittleEndian.PutUint64(patched[positionsLenOff:], length+1)

	return patched, nil
}
