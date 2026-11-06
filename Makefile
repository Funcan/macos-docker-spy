BINARIES := probe macos-docker-spy

.PHONY: build $(addprefix build-,$(BINARIES)) test lint format clean

build: $(addprefix build-,$(BINARIES))

build-probe:
	go build -o build/probe ./cmd/probe

build-macos-docker-spy:
	go build -o build/macos-docker-spy ./cmd/macos-docker-spy

test:
	go test -cover ./...

lint:
	go tool golangci-lint run

format:
	gofmt -w .

clean:
	rm -rf build
