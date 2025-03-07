.PHONY: run test build install lint
default: test

t: test
test:
	go test -v ./test/...

b: bench
bench:
	go test -bench=. ./test/...

# Install dependencies
i: install
install:
	go mod download
	go mod tidy

# formatting & linting
l: lint
lint:
	go fmt ./...
	go vet ./...