genproto:
	cd schema
	buf generate --template schema/buf.gen.yaml

genabi:
	abigen --abi pkg/contracts/trc20/trc20.abi --pkg trc20 --type TRC20 --out pkg/contracts/trc20/trc20.go

install-dev-tools:
	go install github.com/ethereum/go-ethereum/cmd/abigen@latest

make test:
	go test ./tests/... -v

# Local private network for tests/local_*_test.go (see tests/localnet/).
LOCALNET_COMPOSE = docker compose -f tests/localnet/docker-compose.yml

localnet-up:
	$(LOCALNET_COMPOSE) up -d
	@echo "waiting for the first blocks..."
	@for i in $$(seq 1 90); do \
		curl -s -m 2 -X POST http://127.0.0.1:18190/wallet/getnowblock | grep -q '"number":[1-9]' && exit 0; \
		sleep 2; \
	done; echo "node did not produce blocks in time" >&2; exit 1

localnet-down:
	$(LOCALNET_COMPOSE) down -v

test-local:
	GOTRON_LOCAL_NODE=1 go test -race ./tests/ -run Local -v -timeout 20m

fmt:
	go fix ./...
	gofumpt -l -w .

lint:
	golangci-lint run
