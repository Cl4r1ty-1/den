.PHONY: build-master build-slave build-all build-webapp build-cli build-cli-linux build-cli-amd64 build-cli-arm64 build-cli-checksums clean test

build-all: build-master build-slave build-webapp

build-master:
	@echo "Building master node..."
	go build -o den_master main.go

build-slave:
	@echo "Building slave node..."
	go build -tags slave -o den_slave main.go

build-webapp:
	@echo "Building webapp..."
	cd webapp && npm run build


CLI_SRC=./cli/den
CLI_DIST=./cli/dist

build-cli: build-cli-linux build-cli-checksums

build-cli-linux: build-cli-amd64 build-cli-arm64

build-cli-amd64:
	@echo "Building CLI linux/amd64..."
	@mkdir -p $(CLI_DIST)
	cd $(CLI_SRC) && \
		CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags='-s -w' -o ../dist/den-linux-amd64 .

build-cli-arm64:
	@echo "Building CLI linux/arm64..."
	@mkdir -p $(CLI_DIST)
	cd $(CLI_SRC) && \
		CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags='-s -w' -o ../dist/den-linux-arm64 .

build-cli-checksums:
	@echo "Generating CLI checksums..."
	@cd $(CLI_DIST) && \
		sha256sum den-linux-amd64 den-linux-arm64 > SHA256SUMS 2>/dev/null || \
		shasum -a 256 den-linux-amd64 den-linux-arm64 > SHA256SUMS


clean:
	rm -f den_master den_slave webapp/dist

test-master:
	go test ./... -tags ""

test-slave:
	go test ./... -tags slave

run-master:
	./den_master -mode=master

run-slave:
	./den_slave -mode=slave

deps-master:
	go mod download

deps-slave:
	sudo apt update
	sudo apt install -y liblxc-dev pkg-config || \
	sudo apt install -y lxc-dev pkg-config
	go mod download

deps-webapp:
	cd webapp && npm install