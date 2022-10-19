module github.com/gobitfly/eth-rewards

go 1.15

require (
	github.com/ethereum/go-ethereum v1.10.23
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.11.3 // indirect
	github.com/pkg/errors v0.9.1
	github.com/prysmaticlabs/go-bitfield v0.0.0-20210809151128-385d8c5e3fb7
	github.com/prysmaticlabs/prysm/v3 v3.1.1
	github.com/sirupsen/logrus v1.9.0
	go.opencensus.io v0.23.0
	google.golang.org/protobuf v1.28.1
)

replace github.com/grpc-ecosystem/grpc-gateway/v2 => github.com/prysmaticlabs/grpc-gateway/v2 v2.3.1-0.20220721162526-0d1c40b5f064
