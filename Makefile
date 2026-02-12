.PHONY: build clean install docs

BINARY_NAME=pangolin
OUTPUT_DIR=bin
LDFLAGS=-ldflags="-s -w"

# GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o bin/pangolin .
all: clean build

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(OUTPUT_DIR)
	@go build $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME) .
	@echo "Build complete: $(OUTPUT_DIR)/$(BINARY_NAME)"

clean:
	@echo "Cleaning..."
	@rm -rf $(OUTPUT_DIR)
	@echo "Clean complete"

install: build
	@echo "Installing $(BINARY_NAME)..."
	@go install $(LDFLAGS) .

docs:
	@echo "Generating markdown documentation..."
	@go run tools/gendocs/main.go -dir docs
	@echo "Documentation generated in docs/"

docker-build:
	docker build -t fosrl/pangolin-cli:latest .

docker-build-release:
	@if [ -z "$(tag)" ]; then \
		echo "Error: tag is required. Usage: make docker-build-release tag=<tag>"; \
		exit 1; \
	fi
	docker buildx build . \
		--platform linux/arm/v7,linux/arm64,linux/amd64 \
		-t fosrl/pangolin-cli:latest \
		-t fosrl/pangolin-cli:$(tag) \
		-f Dockerfile \
		--push

.PHONY: go-build-release \
        go-build-release-linux-arm64 go-build-release-linux-arm32-v7 \
        go-build-release-linux-arm32-v6 go-build-release-linux-amd64 \
        go-build-release-linux-riscv64 go-build-release-darwin-arm64 \
        go-build-release-darwin-amd64 go-build-release-windows-amd64 \
        go-build-release-freebsd-amd64 go-build-release-freebsd-arm64

go-build-release: \
    go-build-release-linux-arm64 \
    go-build-release-linux-arm32-v7 \
    go-build-release-linux-arm32-v6 \
    go-build-release-linux-amd64 \
    go-build-release-linux-riscv64 \
    go-build-release-darwin-arm64 \
    go-build-release-darwin-amd64 \
    go-build-release-windows-amd64 \
    go-build-release-freebsd-amd64 \
    go-build-release-freebsd-arm64

go-build-release-linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o bin/pangolin-cli_linux_arm64

go-build-release-linux-arm32-v7:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -o bin/pangolin-cli_linux_arm32

go-build-release-linux-arm32-v6:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=6 go build -o bin/pangolin-cli_linux_arm32v6

go-build-release-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/pangolin-cli_linux_amd64

go-build-release-linux-riscv64:
	CGO_ENABLED=0 GOOS=linux GOARCH=riscv64 go build -o bin/pangolin-cli_linux_riscv64

go-build-release-darwin-arm64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o bin/pangolin-cli_darwin_arm64

go-build-release-darwin-amd64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o bin/pangolin-cli_darwin_amd64

go-build-release-windows-amd64:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o bin/pangolin-cli_windows_amd64.exe

go-build-release-freebsd-amd64:
	CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 go build -o bin/pangolin-cli_freebsd_amd64

go-build-release-freebsd-arm64:
	CGO_ENABLED=0 GOOS=freebsd GOARCH=arm64 go build -o bin/pangolin-cli_freebsd_arm64
