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

go-build-release:
    go-build-release-linux-arm64 \
    go-build-release-linux-arm32-v7 \
    go-build-release-linux-arm32-v6 \
    go-build-release-linux-amd64 \
    go-build-release-linux-riscv64 \
    go-build-release-darwin-arm64 \
    go-build-release-darwin-amd64

go-build-release-linux-arm64:
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/pangolin-cli_linux_arm64

go-build-release-linux-arm32-v7:
	GOOS=linux GOARCH=arm GOARM=7 go build $(LDFLAGS) -o bin/pangolin-cli_linux_arm32

go-build-release-linux-arm32-v6:
	GOOS=linux GOARCH=arm GOARM=6 go build $(LDFLAGS) -o bin/pangolin-cli_linux_arm32v6

go-build-release-linux-amd64:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/pangolin-cli_linux_amd64

go-build-release-linux-riscv64:
	GOOS=linux GOARCH=riscv64 go build $(LDFLAGS) -o bin/pangolin-cli_linux_riscv64

go-build-release-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/pangolin-cli_darwin_arm64

go-build-release-darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/pangolin-cli_darwin_amd64
