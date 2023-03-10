package main

import (
	"flag"
	"time"

	ethrewards "github.com/gobitfly/eth-rewards"
	"github.com/gobitfly/eth-rewards/beacon"
	"github.com/sirupsen/logrus"
)

func main() {
	clNode := flag.String("cl-node", "http://localhost:4000", "CL Node API Endpoint")
	elNode := flag.String("el-node", "http://localhost:8545", "EL Node API Endpoint")
	// network := flag.String("network", "", "Config to use (can be mainnet, prater or sepolia")
	epoch := flag.Uint64("epoch", 1, "Epoch to calculate rewards for")
	flag.Parse()

	client := beacon.NewClient(*clNode, time.Second*30)

	rewardsApi := int64(0)
	rewardsBalance := int64(0)
	for i := *epoch; i < i+225; i++ {
		rewards, err := ethrewards.GetRewardsForEpoch(i, client, *elNode)

		if err != nil {
			logrus.Fatal(err)
		}

		balance, err := client.Balance((i+1)*32, 195851)
		if err != nil {
			logrus.Fatal(err)
		}

		balanceNext, err := client.Balance((i+2)*32, 195851)
		if err != nil {
			logrus.Fatal(err)
		}

		logrus.Infof("epoch %d: %s", i, rewards[uint64(195851)].String())
		logrus.Infof("epoch %d: %d income", i, rewards[uint64(195851)].TotalClRewards())
		logrus.Infof("epoch %d: %d balance", i, balance)
		logrus.Infof("epoch %d: %d balanceNext", i, balanceNext)
		logrus.Infof("epoch %d: %d delta", i, int64(balanceNext)-int64(balance))

		rewardsApi += rewards[uint64(195851)].TotalClRewards()
		rewardsBalance += int64(balanceNext) - int64(balance)

		logrus.Infof("epoch %d: %d api", i, rewardsApi)
		logrus.Infof("epoch %d: %d balance", i, rewardsBalance)
		logrus.Info()
	}
}
