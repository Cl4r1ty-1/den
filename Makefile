.PHONY: build-master build-slave build-all build-webapp clean test

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