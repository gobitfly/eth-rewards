package ethrewards

import (
	"fmt"
	"sync"

	"github.com/gobitfly/eth-rewards/beacon"
	"github.com/gobitfly/eth-rewards/elrewards"
	"github.com/gobitfly/eth-rewards/types"
	"golang.org/x/sync/errgroup"

	"github.com/sirupsen/logrus"
)

func GetRewardsForEpoch(epoch uint64, client *beacon.Client, elEndpoint string) (map[uint64]*types.ValidatorEpochIncome, error) {
	proposerAssignments, err := client.ProposerAssignments(epoch)
	if err != nil {
		return nil, err
	}

	slotsPerEpoch := uint64(len(proposerAssignments.Data))

	startSlot := epoch * slotsPerEpoch
	endSlot := startSlot + slotsPerEpoch - 1

	g := new(errgroup.Group)
	g.SetLimit(32)

	slotsToProposerIndex := make(map[uint64]uint64)
	for _, pa := range proposerAssignments.Data {
		slotsToProposerIndex[uint64(pa.Slot)] = uint64(pa.ValidatorIndex)
	}

	rewardsMux := &sync.Mutex{}

	rewards := make(map[uint64]*types.ValidatorEpochIncome)

	for i := startSlot + 1; i <= endSlot; i++ {
		i := i

		g.Go(func() error {
			proposer, found := slotsToProposerIndex[i]
			if !found {
				return fmt.Errorf("assigned proposer for slot %v not found", i)
			}

			execBlockNumber, err := client.ExecutionBlockNumber(i)
			rewardsMux.Lock()
			if rewards[i] == nil {
				rewards[proposer] = &types.ValidatorEpochIncome{}
			}
			rewardsMux.Unlock()
			if err != nil {
				if err == types.ErrBlockNotFound {
					rewardsMux.Lock()
					rewards[proposer].ProposalsMissed += 1
					rewardsMux.Unlock()
					return nil
				} else if err != types.ErrSlotPreMerge { // ignore
					logrus.Errorf("error retrieving execution block number for slot %v: %v", i, err)
					return err
				}
			} else {
				txFeeIncome, err := elrewards.GetELRewardForBlock(execBlockNumber, elEndpoint)
				if err != nil {
					return err
				}

				rewardsMux.Lock()
				rewards[proposer].TxFeeRewardWei = txFeeIncome.Bytes()
				rewardsMux.Unlock()
			}

			syncRewards, err := client.SyncCommitteeRewards(i)
			if err != nil {
				return err
			}

			blockRewards, err := client.BlockRewards(i)
			if err != nil {
				return err
			}

			rewardsMux.Lock()
			for _, sr := range syncRewards.Data {
				if rewards[sr.ValidatorIndex] == nil {
					rewards[sr.ValidatorIndex] = &types.ValidatorEpochIncome{}
				}

				if sr.Reward > 0 {
					rewards[sr.ValidatorIndex].SyncCommitteeReward = uint64(sr.Reward)
				} else {
					rewards[sr.ValidatorIndex].SyncCommitteePenalty = uint64(sr.Reward * -1)
				}
			}

			if rewards[blockRewards.Data.ProposerIndex] == nil {
				rewards[blockRewards.Data.ProposerIndex] = &types.ValidatorEpochIncome{}
			}
			rewards[blockRewards.Data.ProposerIndex].ProposerAttestationInclusionReward = blockRewards.Data.Attestations
			rewards[blockRewards.Data.ProposerIndex].ProposerSlashingInclusionReward = blockRewards.Data.AttesterSlashings + blockRewards.Data.ProposerSlashings
			rewards[blockRewards.Data.ProposerIndex].ProposerSyncInclusionReward = blockRewards.Data.SyncAggregate
			rewardsMux.Unlock()
			return nil
		})
	}

	g.Go(func() error {
		ar, err := client.AttestationRewards(epoch)
		if err != nil {
			return err
		}
		rewardsMux.Lock()
		for _, ar := range ar.Data.TotalRewards {
			if rewards[ar.ValidatorIndex] == nil {
				rewards[ar.ValidatorIndex] = &types.ValidatorEpochIncome{}
			}

			if ar.Head > 0 {
				rewards[ar.ValidatorIndex].AttestationHeadReward = uint64(ar.Head)
			}

			if ar.Source > 0 {
				rewards[ar.ValidatorIndex].AttestationSourceReward = uint64(ar.Source)
			} else {
				rewards[ar.ValidatorIndex].AttestationSourcePenalty = uint64(ar.Source * -1)
			}

			if ar.Source > 0 {
				rewards[ar.ValidatorIndex].AttestationTargetReward = uint64(ar.Target)
			} else {
				rewards[ar.ValidatorIndex].AttestationTargetPenalty = uint64(ar.Target * -1)
			}

			if ar.InclusionDelay < 0 {
				rewards[ar.ValidatorIndex].FinalityDelayPenalty = uint64(ar.InclusionDelay * -1)
			}
		}
		rewardsMux.Unlock()

		return nil
	})

	err = g.Wait()
	if err != nil {
		return nil, err
	}

	return rewards, nil
}
