package elrewards

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/gobitfly/eth-rewards/types"
	"github.com/sirupsen/logrus"
)

func GetELRewardForBlock(executionBlockNumber uint64, endpoint string) (*big.Int, error) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	rpcClient, err := rpc.Dial(endpoint)
	if err != nil {
		return nil, err
	}
	nativeClient, err := ethclient.Dial(endpoint)
	if err != nil {
		return nil, err
	}

	block, err := nativeClient.BlockByNumber(ctx, big.NewInt(int64(executionBlockNumber)))
	if err != nil {
		return nil, err
	}

	if len(block.Transactions()) == 0 {
		return big.NewInt(0), nil
	}

	txHashes := []common.Hash{}
	for _, tx := range block.Transactions() {
		txHashes = append(txHashes, tx.Hash())
	}

	var txReceipts []*types.TxReceipt
	for j := 1; j <= 16; j++ { // retry up to 16 times
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*16)
		txReceipts, err = batchRequestReceipts(ctx, rpcClient, txHashes)
		if err == nil {
			cancel()
			break
		} else {
			logrus.Infof("error (%d) doing batchRequestReceipts for execution block %v: %v", j, executionBlockNumber, err)
			time.Sleep(time.Duration(j) * time.Second)
		}
		cancel()
	}
	if err != nil {
		return nil, fmt.Errorf("error doing batchRequestReceipts for execution block %v: %w", executionBlockNumber, err)
	}

	totalTxFee := big.NewInt(0)
	for _, r := range txReceipts {
		if r.EffectiveGasPrice == nil {
			return nil, fmt.Errorf("no EffectiveGasPrice for execution block %v: %v", executionBlockNumber, txHashes)
		}
		txFee := new(big.Int).Mul(r.EffectiveGasPrice.ToInt(), new(big.Int).SetUint64(uint64(r.GasUsed)))
		totalTxFee.Add(totalTxFee, txFee)
	}

	// base fee per gas is stored little-endian but we need it
	// big-endian for big.Int.

	burntFee := new(big.Int).Mul(block.BaseFee(), new(big.Int).SetUint64(block.GasUsed()))

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
