package main

import (
	"flag"
	"time"

	ethrewards "github.com/gobitfly/eth-rewards"
	"github.com/gobitfly/eth-rewards/beacon"
	"github.com/sirupsen/logrus"
)

// bug with epoch 132263 on prater (invalid state root)
func main() {
	clNode := flag.String("cl-node", "http://localhost:4000", "CL Node API Endpoint")
	elNode := flag.String("el-node", "http://localhost:8545", "EL Node API Endpoint")
	// network := flag.String("network", "", "Config to use (can be mainnet, prater or sepolia")
	epoch := flag.Uint64("epoch", 1, "Epoch to calculate rewards for")
	flag.Parse()

	client := beacon.NewClient(*clNode, time.Second*30)

	rewards, err := ethrewards.GetRewardsForEpoch(*epoch, client, *elNode)

	if err != nil {
		logrus.Fatal(err)
	}

	logrus.Infof("epoch %d: %s", *epoch, rewards[uint64(1200)].String())
	logrus.Infof("epoch %d: %d", *epoch, rewards[uint64(1200)].TotalClRewards())
}
