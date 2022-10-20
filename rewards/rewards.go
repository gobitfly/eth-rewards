package rewards

import (
	"context"
	"fmt"
	"strings"

	"github.com/gobitfly/eth-rewards/elrewards"
	"github.com/gobitfly/eth-rewards/transition"
	"github.com/gobitfly/eth-rewards/types"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prysmaticlabs/prysm/v3/api/client/beacon"
	"github.com/prysmaticlabs/prysm/v3/encoding/ssz/detect"
	"github.com/sirupsen/logrus"
)

func GetRewardsForEpoch(epoch int, client *beacon.Client, elClient *rpc.Client) (map[uint64]*types.ValidatorEpochIncome, error) {
	ctx := context.Background()

	startSlot := epoch * 32
	endSlot := startSlot + 32
	logrus.Infof("retrieving data for epoch %d (slot %d - %d)", epoch, startSlot, endSlot)
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

	rewards := make(map[uint64]*types.ValidatorEpochIncome)
	for i := range s.Validators() {
		rewards[uint64(i)] = &types.ValidatorEpochIncome{Index: uint64(i)}
	}

	for i := startSlot + 1; i <= endSlot; i++ {
		blockData, err := client.GetBlock(ctx, beacon.StateOrBlockId(fmt.Sprintf("%d", i)))
		if err != nil {
			if strings.Contains(err.Error(), "NOT_FOUND") {
				continue
			}
			return nil, err
		}
		b, err := vu.UnmarshalBeaconBlock(blockData)

		if err != nil {
			return nil, err
		}

		_, s, err = transition.ExecuteStateTransitionNoVerifyAnySig(ctx, s, b, rewards)
		if err != nil {
			return nil, err
		}

		txFeeIncome, err := elrewards.GetELRewardForBlock(b, elClient)
		if err != nil {
			return nil, err
		}

		rewards[uint64(b.Block().ProposerIndex())].TxFeeRewardWei = txFeeIncome.Bytes()
	}

	return rewards, nil
}
