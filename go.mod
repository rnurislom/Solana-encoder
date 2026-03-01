module wallet-monitor

go 1.24.0

toolchain go1.24.4

require (
	github.com/mr-tron/base58 v1.2.0
	github.com/rpcpool/yellowstone-grpc/examples/golang v0.0.0
	google.golang.org/grpc v1.75.0
)

require (
	github.com/golang/protobuf v1.5.4 // indirect
	golang.org/x/net v0.43.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/text v0.28.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250826171959-ef028d996bc1 // indirect
	google.golang.org/protobuf v1.36.9 // indirect
)

replace github.com/rpcpool/yellowstone-grpc/examples/golang => ./yellowstone-grpc/examples/golang
