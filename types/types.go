package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
)

type BlockData struct {
	CLBlock        interfaces.SignedBeaconBlock
	TxFeeRewardWei *big.Int
}

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
