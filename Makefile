genproto:
	buf generate

genabi:
	abigen --abi contracts/trc20/trc20.abi --pkg trc20 --type TRC20 --out contracts/trc20/trc20.go

install-dev-tools:
	go install github.com/ethereum/go-ethereum/cmd/abigen@latest
