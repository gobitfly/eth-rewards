package main

import (
	"flag"
	"os"

	ethrewards "github.com/gobitfly/eth-rewards"

	runtime "github.com/banzaicloud/logrus-runtime-formatter"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prysmaticlabs/prysm/v3/api/client/beacon"
	"github.com/sirupsen/logrus"
)

// bug with epoch 132263 on prater (invalid state root)
func main() {

	formatter := runtime.Formatter{ChildFormatter: &logrus.TextFormatter{
		FullTimestamp: true,
	}}
	formatter.Line = true
	logrus.SetFormatter(&formatter)
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.InfoLevel)

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

	income, state, err := ethrewards.GetRewardsForEpoch(*epoch, client, elClient, nil, *network)
	if err != nil {
		logrus.Fatal(err)
	}

	logrus.Infof("epoch %d: %s", *epoch, income[uint64(10)].IncomeDetails.String())
	logrus.Infof("epoch %d: %d", *epoch, income[uint64(10)].IncomeDetails.TotalClRewards())

	income, state, err = ethrewards.GetRewardsForEpoch(*epoch+1, client, elClient, state, *network)
	if err != nil {
		logrus.Fatal(err)
	}

	logrus.Infof("epoch %d: %s", *epoch, income[uint64(10)].IncomeDetails.String())
	logrus.Infof("epoch %d: %d", *epoch, income[uint64(10)].IncomeDetails.TotalClRewards())
}
