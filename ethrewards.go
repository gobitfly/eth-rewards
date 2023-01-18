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
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v3/encoding/ssz/detect"
	"github.com/sirupsen/logrus"

	ptypes "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
)

func GetRewardsForEpoch(epoch int, client *beacon.Client, elClient *rpc.Client, previousState state.BeaconState, network string) (map[uint64]*types.ValidatorEpochData, state.BeaconState, error) {
	ctx := context.Background()

	if network == "sepolia" {
		params.SetActive(params.SepoliaConfig().Copy())
	} else if network == "prater" {
		params.SetActive(params.PraterConfig().Copy())
	} else {
		params.SetActive(params.MainnetConfig().Copy())
	}

	startSlot := epoch * 32
	endSlot := startSlot + 32 - 1

	g := new(errgroup.Group)
	g.SetLimit(10)

	logrus.Infof("retrieving data for epoch %d (using slots %d - %d)", epoch, startSlot, endSlot)
	start := time.Now()

	var s state.BeaconState

	if previousState == nil {
		stateData, err := client.GetState(ctx, beacon.StateOrBlockId(fmt.Sprintf("%d", startSlot)))
		if err != nil {
			return nil, nil, err
		}

		vu, err := detect.FromState(stateData)
		if err != nil {
			return nil, nil, err
		}

		s, err = vu.UnmarshalBeaconState(stateData)
		if err != nil {
			return nil, nil, err
		}
	} else {
		s = previousState
	}

	vu, err := detect.FromForkVersion(bytesutil.ToBytes4(s.Fork().CurrentVersion))
	if err != nil {
		return nil, nil, err
	}

	data := make(map[uint64]*types.ValidatorEpochData, s.NumValidators())
	if err != nil {
		return nil, nil, err
	}

	for i := range s.Validators() {
		data[uint64(i)] = &types.ValidatorEpochData{
			IncomeDetails: &types.ValidatorEpochIncome{},
		}
	}

	_, proposerIndexToSlots, err := helpers.CommitteeAssignments(ctx, s.Copy(), ptypes.Epoch(epoch))
	if err != nil {
		return nil, nil, err
	}

	slotsToProposerIndex := make(map[uint64]uint64)
	for validator, slots := range proposerIndexToSlots {
		for _, slot := range slots {
			slotsToProposerIndex[uint64(slot)] = uint64(validator)
			if data[uint64(validator)].Proposals == nil {
				data[uint64(validator)].Proposals = make(map[uint64]bool, 1)
			}
			data[uint64(validator)].Proposals[uint64(slot)] = false
		}
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
		return nil, nil, err
	}

	logrus.Infof("retrieved epoch data in %v", time.Since(start))

	for i := startSlot + 1; i <= endSlot; i++ {
		logrus.Infof("processing slot %v", i)
		b := blocks[i]

		start := time.Now()
		// Execute epoch transition.
		if previousState == nil {
			s, err = transition.ProcessSlots(ctx, s, ptypes.Slot(i), data)
			if err != nil {
				return nil, nil, errors.Wrap(err, "could not process slots")
			}
		}

		processSlotsDuration := time.Since(start)
		start = time.Now()
		proposer, found := slotsToProposerIndex[uint64(i)]
		if !found {
			return nil, nil, fmt.Errorf("proposer for slot %v not found, this should never happen", i)
		}

		if b == nil || b.CLBlock.IsNil() || b.CLBlock.Block().IsNil() {
			logrus.Infof("validator %v missed slot %v", proposer, i)
			logrus.WithFields(logrus.Fields{
				"ProcessSlot": processSlotsDuration,
				"Total":       processSlotsDuration,
			}).Infof("processed state transition for slot %v", i)
			continue
		}

		// mark the block as proposed
		data[slotsToProposerIndex[uint64(i)]].Proposals[uint64(i)] = true

		// Execute per block transition.
		_, s, err = transition.ProcessBlockNoVerifyAnySig(ctx, s, b.CLBlock, data)
		if err != nil {
			return nil, nil, errors.Wrap(err, "could not process block")
		}
		processBlockNoVerifyAnySigDuration := time.Since(start)
		start = time.Now()

		// State root validation.
		postStateRoot, err := s.HashTreeRoot(ctx)
		if err != nil {
			return nil, nil, err
		}
		if !bytes.Equal(postStateRoot[:], b.CLBlock.Block().StateRoot()) {
			return nil, nil, fmt.Errorf("could not validate state root, wanted: %#x, received: %#x",
				postStateRoot[:], b.CLBlock.Block().StateRoot())
		}

		stateRootValidationDuration := time.Since(start)

		data[uint64(b.CLBlock.Block().ProposerIndex())].IncomeDetails.TxFeeRewardWei = b.TxFeeRewardWei.Bytes()

		logrus.WithFields(logrus.Fields{
			"ProcessSlot":                processSlotsDuration,
			"ProcessBlockNoVerifyAnySig": processBlockNoVerifyAnySigDuration,
			"StateRootValidation":        stateRootValidationDuration,
			"Total":                      processSlotsDuration + processBlockNoVerifyAnySigDuration + stateRootValidationDuration,
		}).Infof("processed state transition for slot %v", i)
	}

	// Execute epoch transition.
	s, err = transition.ProcessSlots(ctx, s, ptypes.Slot(endSlot+1), data)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not process slots")
	}

	return data, s, nil
}
