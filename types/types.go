package types

import (
	"encoding/json"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type TxReceipt struct {
	BlockHash         *common.Hash    `json:"blockHash"`
	BlockNumber       *hexutil.Big    `json:"blockNumber"`
	ContractAddress   *common.Address `json:"contractAddress,omitempty"`
	CumulativeGasUsed hexutil.Uint64  `json:"cumulativeGasUsed"`
	EffectiveGasPrice *hexutil.Big    `json:"effectiveGasPrice"`
	From              *common.Address `json:"from,omitempty"`
	GasUsed           hexutil.Uint64  `json:"gasUsed"`
	LogsBloom         hexutil.Bytes   `json:"logsBloom"`
	Status            hexutil.Uint64  `json:"status"`
	To                *common.Address `json:"to,omitempty"`
	TransactionHash   *common.Hash    `json:"transactionHash"`
	TransactionIndex  hexutil.Uint64  `json:"transactionIndex"`
	Type              hexutil.Uint64  `json:"type"`
}

func (income *ValidatorEpochIncome) TotalClRewards() int64 {
	rewards := income.AttestationSourceReward +
		income.AttestationTargetReward +
		income.AttestationHeadReward +
		income.ProposerSlashingInclusionReward +
		income.ProposerAttestationInclusionReward +
		income.ProposerSyncInclusionReward +
		income.SyncCommitteeReward +
		income.SlashingReward

	penalties := income.AttestationSourcePenalty +
		income.AttestationTargetPenalty +
		income.FinalityDelayPenalty +
		income.SyncCommitteePenalty +
		income.SlashingPenalty
	return int64(rewards) - int64(penalties)
}

type AttestationRewardsApiResponse struct {
	Data struct {
		IdealRewards []*IdealAttestationRewardContainer  `json:"ideal_rewards"`
		TotalRewards []*TotalAttestationRewardsContainer `json:"total_rewards"`
	} `json:"data"`
	ExecutionOptimistic bool `json:"execution_optimistic"`
}
type IdealAttestationRewardContainer struct {
	EffectiveBalance int64 `json:"effective_balance"`
	Head             int64 `json:"head"`
	Source           int64 `json:"source"`
	Target           int64 `json:"target"`
}
type TotalAttestationRewardsContainer struct {
	Head           int64  `json:"head"`
	Source         int64  `json:"source"`
	Target         int64  `json:"target"`
	InclusionDelay int64  `json:"inclusion_delay"`
	ValidatorIndex uint64 `json:"validator_index"`
}

func (a *AttestationRewardsApiResponse) UnmarshalJSON(data []byte) error {
	type internal struct {
		Data struct {
			IdealRewards []struct {
				EffectiveBalance string `json:"effective_balance"`
				Head             string `json:"head"`
				Source           string `json:"source"`
				Target           string `json:"target"`
			} `json:"ideal_rewards"`
			TotalRewards []struct {
				Head           string `json:"head"`
				Source         int64  `json:"source"`
				Target         int64  `json:"target"`
				InclusionDelay string `json:"inclusion_delay"`
				ValidatorIndex string `json:"validator_index"`
			} `json:"total_rewards"`
		} `json:"data"`
		ExecutionOptimistic bool `json:"execution_optimistic"`
	}

	var v internal
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	a.Data.IdealRewards = make([]*IdealAttestationRewardContainer, len(v.Data.IdealRewards))
	a.Data.TotalRewards = make([]*TotalAttestationRewardsContainer, len(v.Data.TotalRewards))

	var err error
	for i, r := range v.Data.IdealRewards {
		p := &IdealAttestationRewardContainer{}

		p.EffectiveBalance, err = strconv.ParseInt(r.EffectiveBalance, 10, 64)
		if err != nil {
			return err
		}

		p.Head, err = strconv.ParseInt(r.Head, 10, 64)
		if err != nil {
			return err
		}

		p.Source, err = strconv.ParseInt(r.Source, 10, 64)
		if err != nil {
			return err
		}

		p.Target, err = strconv.ParseInt(r.Target, 10, 64)
		if err != nil {
			return err
		}

		a.Data.IdealRewards[i] = p
	}

	for i, r := range v.Data.TotalRewards {
		p := &TotalAttestationRewardsContainer{}

		p.ValidatorIndex, err = strconv.ParseUint(r.ValidatorIndex, 10, 64)
		if err != nil {
			return err
		}

		p.Head, err = strconv.ParseInt(r.Head, 10, 64)
		if err != nil {
			return err
		}

		p.Source = r.Source

		p.Target = r.Target

		if r.InclusionDelay != "" {
			p.InclusionDelay, err = strconv.ParseInt(r.InclusionDelay, 10, 64)
			if err != nil {
				return err
			}
		}

		a.Data.TotalRewards[i] = p
	}

	return nil
}

type BlockRewardsApiResponse struct {
	Data struct {
		Attestations      uint64 `json:"attestations"`
		AttesterSlashings uint64 `json:"attester_slashings"`
		ProposerIndex     uint64 `json:"proposer_index"`
		ProposerSlashings uint64 `json:"proposer_slashings"`
		SyncAggregate     uint64 `json:"sync_aggregate"`
		Total             uint64 `json:"total"`
	} `json:"data"`
	ExecutionOptimistic bool `json:"execution_optimistic"`
}

func (b *BlockRewardsApiResponse) UnmarshalJSON(data []byte) error {

	type internal struct {
		Data struct {
			Attestations      string `json:"attestations"`
			AttesterSlashings string `json:"attester_slashings"`
			ProposerIndex     string `json:"proposer_index"`
			ProposerSlashings string `json:"proposer_slashings"`
			SyncAggregate     string `json:"sync_aggregate"`
			Total             string `json:"total"`
		} `json:"data"`
		ExecutionOptimistic bool `json:"execution_optimistic"`
	}

	var v internal
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	var err error
	b.Data.Attestations, err = strconv.ParseUint(v.Data.Attestations, 10, 64)
	if err != nil {
		return err
	}
	b.Data.AttesterSlashings, err = strconv.ParseUint(v.Data.AttesterSlashings, 10, 64)
	if err != nil {
		return err
	}
	b.Data.ProposerIndex, err = strconv.ParseUint(v.Data.ProposerIndex, 10, 64)
	if err != nil {
		return err
	}
	b.Data.ProposerSlashings, err = strconv.ParseUint(v.Data.ProposerSlashings, 10, 64)
	if err != nil {
		return err
	}
	b.Data.SyncAggregate, err = strconv.ParseUint(v.Data.SyncAggregate, 10, 64)
	if err != nil {
		return err
	}
	b.Data.Total, err = strconv.ParseUint(v.Data.Total, 10, 64)
	if err != nil {
		return err
	}

	return nil
}

type SyncCommitteeRewardsApiResponse struct {
	Data                []*SyncCommitteeRewardsContainer `json:"data"`
	ExecutionOptimistic bool                             `json:"execution_optimistic"`
}

type SyncCommitteeRewardsContainer struct {
	Reward         int64  `json:"reward"`
	ValidatorIndex uint64 `json:"validator_index"`
}

func (s *SyncCommitteeRewardsApiResponse) UnmarshalJSON(data []byte) error {
	type internal struct {
		Data []struct {
			Reward         int64  `json:"reward"`
			ValidatorIndex string `json:"validator_index"`
		} `json:"data"`
		ExecutionOptimistic bool `json:"execution_optimistic"`
	}

	var v internal
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	s.Data = make([]*SyncCommitteeRewardsContainer, len(v.Data))

	var err error
	for i, r := range v.Data {
		p := &SyncCommitteeRewardsContainer{}

		p.ValidatorIndex, err = strconv.ParseUint(r.ValidatorIndex, 10, 64)
		if err != nil {
			return err
		}

		p.Reward = r.Reward

		s.Data[i] = p
	}

	return nil
}

type EpochProposerAssignmentsApiResponse struct {
	Data                []*EpochProposerAssignmentsContainer `json:"data"`
	DependentRoot       string                               `json:"dependent_root"`
	ExecutionOptimistic bool                                 `json:"execution_optimistic"`
}

type EpochProposerAssignmentsContainer struct {
	Pubkey         string `json:"pubkey"`
	Slot           int64  `json:"slot"`
	ValidatorIndex int64  `json:"validator_index"`
}

func (e *EpochProposerAssignmentsApiResponse) UnmarshalJSON(data []byte) error {
	type internal struct {
		Data []struct {
			Pubkey         string `json:"pubkey"`
			Slot           string `json:"slot"`
			ValidatorIndex string `json:"validator_index"`
		} `json:"data"`
		DependentRoot       string `json:"dependent_root"`
		ExecutionOptimistic bool   `json:"execution_optimistic"`
	}

	var v internal
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	e.Data = make([]*EpochProposerAssignmentsContainer, len(v.Data))

	var err error
	for i, r := range v.Data {
		p := &EpochProposerAssignmentsContainer{}

		p.ValidatorIndex, err = strconv.ParseInt(r.ValidatorIndex, 10, 64)
		if err != nil {
			return err
		}
		p.Slot, err = strconv.ParseInt(r.Slot, 10, 64)
		if err != nil {
			return err
		}

		e.Data[i] = p
	}

	return nil
}
