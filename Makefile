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
