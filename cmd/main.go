package main

import (
	"flag"

	"eth-rewards-calculator/rewards"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prysmaticlabs/prysm/v3/api/client/beacon"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

func main() {
	clNode := flag.String("cl-node", "http://localhost:4000", "CL Node API Endpoint")
	elNode := flag.String("el-node", "http://localhost:8545", "EL Node API Endpoint")
	epoch := flag.Int("epoch", 1, "Epoch to calculate rewards for")

	flag.Parse()

	client, err := beacon.NewClient(*clNode)
	if err != nil {
		logrus.Fatal(err)
	}

	elClient, err := rpc.Dial(*elNode)
	if err != nil {
		logrus.Fatal(err)
	}

	income, err := rewards.GetRewardsForEpoch(*epoch, client, elClient)
	if err != nil {
		logrus.Fatal(err)
	}

	totalSize := 0

	for _, reward := range income {

		data, err := proto.Marshal(reward)
		if err != nil {
			logrus.Fatal(err)
		}
		totalSize += len(data)
	}

	logrus.Infof("epoch reward data is is %v", totalSize)
	logrus.Infof("epoch %d: %s", *epoch, income[uint64(180976)].String())
	logrus.Infof("epoch %d: %s", *epoch, income[uint64(216267)].String())
}
