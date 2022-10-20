package elrewards

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/gobitfly/eth-rewards/types"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
)

func GetELRewardForBlock(block interfaces.SignedBeaconBlock, client *rpc.Client) (*big.Int, error) {
	exec, err := block.Block().Body().Execution()
	if err != nil {
		return nil, err
	}

	txs, err := exec.Transactions()

	if err != nil {
		return nil, err
	}
	txHashes := []common.Hash{}
	for _, tx := range txs {
		var decTx gethTypes.Transaction
		err := decTx.UnmarshalBinary([]byte(tx))
		if err != nil {
			return nil, err
		}
		txHashes = append(txHashes, decTx.Hash())
	}

	var txReceipts []*types.TxReceipt
	for j := 0; j < 10; j++ { // retry up to 10 times
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		txReceipts, err = batchRequestReceipts(ctx, client, txHashes)
		if err == nil {
			cancel()
			break
		} else {
			log.Printf("error doing batchRequestReceipts for slot %v: %v", block.Block().Slot(), err)
			time.Sleep(time.Duration(j) * time.Second)
		}
		cancel()
	}
	if err != nil {
		return nil, fmt.Errorf("error doing batchRequestReceipts for slot %v: %w", block.Block().Slot(), err)
	}

	totalTxFee := big.NewInt(0)
	for _, r := range txReceipts {
		if r.EffectiveGasPrice == nil {
			return nil, fmt.Errorf("no EffectiveGasPrice for slot %v: %v", block.Block().Slot(), txHashes)
		}
		txFee := new(big.Int).Mul(r.EffectiveGasPrice.ToInt(), new(big.Int).SetUint64(uint64(r.GasUsed)))
		totalTxFee.Add(totalTxFee, txFee)
	}

	// base fee per gas is stored little-endian but we need it
	// big-endian for big.Int.
	var baseFeePerGasBEBytes [32]byte
	for i := 0; i < 32; i++ {
		baseFeePerGasBEBytes[i] = exec.BaseFeePerGas()[32-1-i]
	}
	baseFeePerGas := new(big.Int).SetBytes(baseFeePerGasBEBytes[:])
	burntFee := new(big.Int).Mul(baseFeePerGas, new(big.Int).SetUint64(exec.GasUsed()))

	totalTxFee.Sub(totalTxFee, burntFee)

	return totalTxFee, nil
}

func batchRequestReceipts(ctx context.Context, elClient *rpc.Client, txHashes []common.Hash) ([]*types.TxReceipt, error) {
	elems := make([]rpc.BatchElem, 0, len(txHashes))
	errors := make([]error, 0, len(txHashes))
	txReceipts := make([]*types.TxReceipt, len(txHashes))
	for i, h := range txHashes {
		txReceipt := &types.TxReceipt{}
		err := error(nil)
		elems = append(elems, rpc.BatchElem{
			Method: "eth_getTransactionReceipt",
			Args:   []interface{}{h.Hex()},
			Result: txReceipt,
			Error:  err,
		})
		txReceipts[i] = txReceipt
		errors = append(errors, err)
	}
	ioErr := elClient.BatchCallContext(ctx, elems)
	if ioErr != nil {
		return nil, fmt.Errorf("io-error when fetching tx-receipts: %w", ioErr)
	}
	for _, e := range errors {
		if e != nil {
			return nil, fmt.Errorf("error when fetching tx-receipts: %w", e)
		}
	}
	return txReceipts, nil
}
