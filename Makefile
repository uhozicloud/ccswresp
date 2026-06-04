# ccswresp Makefile
# Builds static binaries for all platforms.

BINARY := ccswresp
LDFLAGS := -s -w
BUILD_DIR := build

.PHONY: all clean test build dist help

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

test: ## Run tests
	go test -v -count=1 .

build: ## Build for current platform
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

build-all: ## Build for all platforms (binary always named ccswresp)
	@mkdir -p $(BUILD_DIR)/darwin-amd64
	@mkdir -p $(BUILD_DIR)/darwin-arm64
	@mkdir -p $(BUILD_DIR)/linux-amd64
	@mkdir -p $(BUILD_DIR)/linux-arm64
	@mkdir -p $(BUILD_DIR)/windows-amd64
	@echo "Building darwin/amd64..."
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/darwin-amd64/$(BINARY) .
	@echo "Building darwin/arm64..."
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/darwin-arm64/$(BINARY) .
	@echo "Building linux/amd64..."
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/linux-amd64/$(BINARY) .
	@echo "Building linux/arm64..."
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/linux-arm64/$(BINARY) .
	@echo "Building windows/amd64..."
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/windows-amd64/$(BINARY).exe .
	@echo ""
	@echo "Build complete:"
	@find $(BUILD_DIR) -type f -exec ls -lh {} \;

dist: build-all ## Build and create tar.gz/zip archives for GitHub Releases
	@echo ""
	@echo "Creating distribution archives..."
	@cd $(BUILD_DIR) && for dir in */; do \
		platform=$$(basename "$$dir"); \
		if echo "$$platform" | grep -q 'windows'; then \
			zip -j "ccswresp_$$platform.zip" "$$dir/ccswresp.exe"; \
		else \
			tar -czf "ccswresp_$$platform.tar.gz" -C "$$dir" ccswresp; \
		fi; \
	done
	@echo "Archives ready in $(BUILD_DIR)/"
	@ls -lh $(BUILD_DIR)/*.tar.gz $(BUILD_DIR)/*.zip 2>/dev/null

clean: ## Clean build artifacts
	rm -rf $(BUILD_DIR)
	rm -f $(BINARY) $(BINARY).exe

install: build ## Install to /usr/local/bin (may need sudo)
	install -m 755 $(BINARY) /usr/local/bin/$(BINARY)
	@echo "Installed to /usr/local/bin/$(BINARY)"
