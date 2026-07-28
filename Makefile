.PHONY: build test lint run clean

build:
	go build -o bin/taksumcp ./cmd/server

test:
	go test -race ./...

lint:
	golangci-lint run

run:
	go run ./cmd/server

clean:
	rm -rf bin/ data/