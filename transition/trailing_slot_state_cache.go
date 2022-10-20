package transition

import (
	"bytes"
	"context"
	"sync"

	itypes "github.com/gobitfly/eth-rewards/types"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
)

type nextSlotCache struct {
	sync.RWMutex
	root  []byte
	state state.BeaconState
}

var (
	nsc nextSlotCache
)

// NextSlotState returns the saved state if the input root matches the root in `nextSlotCache`. Returns nil otherwise.
// This is useful to check before processing slots. With a cache hit, it will return last processed state with slot plus
// one advancement.
func NextSlotState(_ context.Context, root []byte) (state.BeaconState, error) {
	nsc.RLock()
	defer nsc.RUnlock()
	if !bytes.Equal(root, nsc.root) || bytes.Equal(root, []byte{}) {
		return nil, nil
	}
	// Returning copied state.
	return nsc.state.Copy(), nil
}

// UpdateNextSlotCache updates the `nextSlotCache`. It saves the input state after advancing the state slot by 1
// by calling `ProcessSlots`, it also saves the input root for later look up.
// This is useful to call after successfully processing a block.
func UpdateNextSlotCache(ctx context.Context, root []byte, state state.BeaconState,
	income map[uint64]*itypes.ValidatorEpochIncome) error {
	// Advancing one slot by using a copied state.
	copied := state.Copy()
	copied, err := ProcessSlots(ctx, copied, copied.Slot()+1, income)
	if err != nil {
		return err
	}

	nsc.Lock()
	defer nsc.Unlock()

	nsc.root = root
	nsc.state = copied
	return nil
}
