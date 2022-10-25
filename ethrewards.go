package ethrewards

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gobitfly/eth-rewards/elrewards"
	"github.com/gobitfly/eth-rewards/transition"
	"github.com/gobitfly/eth-rewards/types"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prysmaticlabs/prysm/v3/api/client/beacon"
	"github.com/prysmaticlabs/prysm/v3/encoding/ssz/detect"
	"github.com/sirupsen/logrus"

	ptypes "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
)

func GetRewardsForEpoch(epoch int, client *beacon.Client, elClient *rpc.Client) (map[uint64]*types.ValidatorEpochIncome, error) {
	ctx := context.Background()

	startSlot := (epoch - 1) * 32
	endSlot := startSlot + 32

	g := new(errgroup.Group)
	g.SetLimit(10)

	logrus.Infof("retrieving data for epoch %d (using slots %d - %d)", epoch, startSlot, endSlot)
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
		logrus.Info(i / 32)
		b := blocks[i]

		start := time.Now()
		// Execute epoch transition.
		s, err = transition.ProcessSlots(ctx, s, ptypes.Slot(i), rewards)
		if err != nil {
			return nil, errors.Wrap(err, "could not process slots")
		}

		processSlotsDuration := time.Since(start)
		start = time.Now()
		if b == nil || b.CLBlock.IsNil() || b.CLBlock.Block().IsNil() || i/32 != epoch-1 {
			logrus.WithFields(logrus.Fields{
				"ProcessSlot": processSlotsDuration,
				"Total":       processSlotsDuration,
			}).Infof("processed state transition for slot %v", i)
			continue
		}

		// Execute per block transition.
		_, s, err = transition.ProcessBlockNoVerifyAnySig(ctx, s, b.CLBlock, rewards)
		if err != nil {
			return nil, errors.Wrap(err, "could not process block")
		}
		processBlockNoVerifyAnySigDuration := time.Since(start)
		start = time.Now()

		// State root validation.
		postStateRoot, err := s.HashTreeRoot(ctx)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(postStateRoot[:], b.CLBlock.Block().StateRoot()) {
			return nil, fmt.Errorf("could not validate state root, wanted: %#x, received: %#x",
				postStateRoot[:], b.CLBlock.Block().StateRoot())
		}

		stateRootValidationDuration := time.Since(start)

		rewards[uint64(b.CLBlock.Block().ProposerIndex())].TxFeeRewardWei = b.TxFeeRewardWei.Bytes()

		logrus.WithFields(logrus.Fields{
			"ProcessSlot":                processSlotsDuration,
			"ProcessBlockNoVerifyAnySig": processBlockNoVerifyAnySigDuration,
			"StateRootValidation":        stateRootValidationDuration,
			"Total":                      processSlotsDuration + processBlockNoVerifyAnySigDuration + stateRootValidationDuration,
		}).Infof("processed state transition for slot %v", i)
	}

	return rewards, nil
}
