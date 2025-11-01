.PHONY: build clean install help

BINARY_NAME=pangolin
OUTPUT_DIR=bin
LDFLAGS=-ldflags="-s -w"

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
