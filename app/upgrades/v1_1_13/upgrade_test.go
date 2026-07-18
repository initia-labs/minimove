package v1_1_13

import (
	"encoding/binary"
	"encoding/hex"
	"testing"
)

// buildPositionStore assembles PositionStore resource bytes in BCS layout:
// last_position_id u64 | id_map {handle 32, length u64} | positions {handle 32, length u64}
func buildPositionStore(t *testing.T, lastID uint64, idMapHandle string, idMapLen uint64, posHandle string, posLen uint64) []byte {
	t.Helper()

	bz := make([]byte, 0, positionStoreLen)
	u64 := func(v uint64) []byte {
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, v)
		return b
	}
	addr := func(h string) []byte {
		b, err := hex.DecodeString(h)
		if err != nil || len(b) != 32 {
			t.Fatalf("bad handle hex: %s", h)
		}
		return b
	}

	bz = append(bz, u64(lastID)...)
	bz = append(bz, addr(idMapHandle)...)
	bz = append(bz, u64(idMapLen)...)
	bz = append(bz, addr(posHandle)...)
	bz = append(bz, u64(posLen)...)
	return bz
}

const (
	realIDMapHandle     = "f44b3d6f36d59efeff36922a43d6b48109ba2cd6c754dbbd3348e68b962684ff"
	realPositionsHandle = "8ebf15a319d93ea00a671be70f6cfeb2fb50388fdefb0f0309eafcb9f2c89d07"
)

func TestPatchPositionsLength(t *testing.T) {
	// counters as observed on strat-1 after the h2626106 incident: physical entries 18,
	// stored length 17; the absolute values will differ at upgrade height, only the +1 matters
	in := buildPositionStore(t, 931, realIDMapHandle, 18, realPositionsHandle, 17)

	out, err := patchPositionsLength(in)
	if err != nil {
		t.Fatalf("patch failed: %v", err)
	}

	if got := binary.LittleEndian.Uint64(out[positionsLenOff:]); got != 18 {
		t.Fatalf("positions length: got %d, want 18", got)
	}
	// everything before the last u64 must be untouched
	for i := range positionsLenOff {
		if out[i] != in[i] {
			t.Fatalf("byte %d modified: %x -> %x", i, in[i], out[i])
		}
	}
	// input must not be mutated
	if got := binary.LittleEndian.Uint64(in[positionsLenOff:]); got != 17 {
		t.Fatalf("input mutated: %d", got)
	}
}

func TestPatchPositionsLengthRejectsWrongShape(t *testing.T) {
	if _, err := patchPositionsLength(make([]byte, positionStoreLen-1)); err == nil {
		t.Fatal("accepted truncated resource")
	}

	wrongHandle := buildPositionStore(t, 931, realIDMapHandle, 18,
		"00000000000000000000000000000000000000000000000000000000000000aa", 17)
	if _, err := patchPositionsLength(wrongHandle); err == nil {
		t.Fatal("accepted wrong positions handle")
	}

	swapped := buildPositionStore(t, 931, realPositionsHandle, 18, realIDMapHandle, 17)
	if _, err := patchPositionsLength(swapped); err == nil {
		t.Fatal("accepted swapped table handles")
	}
}
