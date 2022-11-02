package main

import (
	"flag"

	ethrewards "github.com/gobitfly/eth-rewards"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prysmaticlabs/prysm/v3/api/client/beacon"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

// bug with epoch 132263 on prater (invalid state root)
func main() {
	clNode := flag.String("cl-node", "http://localhost:4000", "CL Node API Endpoint")
	elNode := flag.String("el-node", "http://localhost:8545", "EL Node API Endpoint")
	network := flag.String("network", "", "Config to use (can be mainnet, prater or sepolia")
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
	config, err := params.ByName(*network)
	if err != nil {
		logrus.Fatalf("no config found for network %v: %v", *network, err)
	}
	err = params.SetActive(config)
	if err != nil {
		logrus.Fatalf("error setting: %v", err)
	}

	income, err := ethrewards.GetRewardsForEpoch(*epoch, client, elClient)
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

	logrus.Infof("epoch %d: %s", *epoch, income[uint64(278383)].String())
	logrus.Infof("epoch %d: %d", *epoch, income[uint64(278383)].TotalClRewards())
}
