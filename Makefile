.PHONY: build test lint run clean

build:
	CGO_ENABLED=0 go build -o skladisce ./cmd/skladisce

test:
	go test -timeout 10s ./...

lint:
	go vet ./...

run: build
	./skladisce

clean:
	rm -f skladisce
