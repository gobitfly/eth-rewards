package ethrewards

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gobitfly/eth-rewards/elrewards"
	"github.com/gobitfly/eth-rewards/transition"
	"github.com/gobitfly/eth-rewards/types"
	"golang.org/x/sync/errgroup"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prysmaticlabs/prysm/v3/api/client/beacon"
	"github.com/prysmaticlabs/prysm/v3/encoding/ssz/detect"
	"github.com/sirupsen/logrus"
)

func GetRewardsForEpoch(epoch int, client *beacon.Client, elClient *rpc.Client) (map[uint64]*types.ValidatorEpochIncome, error) {
	ctx := context.Background()

	startSlot := epoch * 32
	endSlot := startSlot + 32

	g := new(errgroup.Group)
	g.SetLimit(10)

	logrus.Infof("retrieving data for epoch %d (slot %d - %d)", epoch, startSlot, endSlot)
	start := time.Now()

	stateData, err := client.GetState(ctx, beacon.StateOrBlockId(fmt.Sprintf("%d", startSlot)))
	if err != nil {
		return nil, err
	}

	vu, err := detect.FromState(stateData)
	if err != nil {
		return nil, err
	}

	s, err := vu.UnmarshalBeaconState(stateData)
	if err != nil {
		return nil, err
	}

	blocks := make(map[int]*types.BlockData)
	blocksMux := &sync.Mutex{}

	for i := startSlot + 1; i <= endSlot; i++ {
		i := i

		g.Go(func() error {
			data, err := client.GetBlock(ctx, beacon.StateOrBlockId(fmt.Sprintf("%d", i)))
			if err != nil {
				if strings.Contains(err.Error(), "NOT_FOUND") {
					return nil
				}
				return err
			}
			b, err := vu.UnmarshalBeaconBlock(data)

			if err != nil {
				return err
			}

			txFeeIncome, err := elrewards.GetELRewardForBlock(b, elClient)
			if err != nil {
				return err
			}
			blocksMux.Lock()
			blocks[i] = &types.BlockData{
				CLBlock:        b,
				TxFeeRewardWei: txFeeIncome,
			}
			blocksMux.Unlock()

			return nil
		})
	}

	err = g.Wait()
	if err != nil {
		return nil, err
	}

	logrus.Infof("retrieved epoch data in %v", time.Since(start))

	rewards := make(map[uint64]*types.ValidatorEpochIncome)
	for i := range s.Validators() {
		rewards[uint64(i)] = &types.ValidatorEpochIncome{Index: uint64(i)}
	}

	for i := startSlot + 1; i <= endSlot; i++ {

		b := blocks[i]
		if b == nil {
			continue
		}
		_, s, err = transition.ExecuteStateTransitionNoVerifyAnySig(ctx, s, b.CLBlock, rewards)
		if err != nil {
			return nil, err
		}

		rewards[uint64(b.CLBlock.Block().ProposerIndex())].TxFeeRewardWei = b.TxFeeRewardWei.Bytes()
	}

	return rewards, nil
}
