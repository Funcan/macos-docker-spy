BINARIES := probe

.PHONY: build $(addprefix build-,$(BINARIES)) test lint format clean

build: $(addprefix build-,$(BINARIES))

build-probe:
	go build -o build/probe ./cmd/probe

test:
	go test -cover ./...

lint:
	go tool golangci-lint run

format:
	gofmt -w .

clean:
	rm -rf build
