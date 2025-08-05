.PHONY: build-master build-slave build-all clean test

build-master:
	@echo "Building master node..."
	go build -o den_master main.go

build-slave:
	@echo "Building slave node..."
	go build -tags slave -o den_slave main.go

build-all: build-master build-slave

clean:
	rm -f den-master den-slave

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