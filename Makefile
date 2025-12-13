.PHONY: build-all build clean

# Build directory
BIN_DIR := bin

# Build all targets
build-all: clean
	@echo "Building all binaries..."
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BIN_DIR)/deepdiffdb-linux-amd64 ./cmd/deepdiffdb
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o $(BIN_DIR)/deepdiffdb-linux-arm64 ./cmd/deepdiffdb
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $(BIN_DIR)/deepdiffdb-darwin-amd64 ./cmd/deepdiffdb
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $(BIN_DIR)/deepdiffdb-darwin-arm64 ./cmd/deepdiffdb
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $(BIN_DIR)/deepdiffdb-windows-amd64.exe ./cmd/deepdiffdb
	@echo "Build complete. Binaries in $(BIN_DIR)/"

# Build for current platform
build:
	@echo "Building for current platform..."
	@mkdir -p $(BIN_DIR)
	go build -ldflags="-s -w" -o $(BIN_DIR)/deepdiffdb ./cmd/deepdiffdb
	@echo "Build complete. Binary in $(BIN_DIR)/deepdiffdb"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BIN_DIR)
	@echo "Clean complete."

