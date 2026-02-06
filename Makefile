.PHONY: build test lint run clean

build:
	CGO_ENABLED=0 go build -o skladisce ./cmd/server

test:
	go test -timeout 10s ./...

lint:
	go vet ./...

run: build
	./skladisce serve

clean:
	rm -f skladisce
