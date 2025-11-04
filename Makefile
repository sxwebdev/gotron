genproto:
	cd schema
	buf generate --template schema/buf.gen.yaml

genabi:
	abigen --abi pkg/contracts/trc20/trc20.abi --pkg trc20 --type TRC20 --out pkg/contracts/trc20/trc20.go

install-dev-tools:
	go install github.com/ethereum/go-ethereum/cmd/abigen@latest
