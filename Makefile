# ccswresp Makefile
# Builds static binaries for all platforms.

VERSION := 1.0.0
BINARY := ccswresp
LDFLAGS := -s -w

# Build directory
BUILD_DIR := build

# Platform targets
PLATFORMS := \
	darwin/amd64 \
	darwin/arm64 \
	linux/amd64 \
	linux/arm64 \
	windows/amd64

.PHONY: all clean test build dist help

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

test: ## Run tests
	go test -v ./test/ -count=1

build: ## Build for current platform
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

build-all: ## Build for all platforms
	@mkdir -p $(BUILD_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} \
		GOARCH=$${platform#*/} \
		OUTPUT="$(BUILD_DIR)/$(BINARY)_$${GOOS}_$${GOARCH}"; \
		if [ "$$GOOS" = "windows" ]; then OUTPUT="$$OUTPUT.exe"; fi; \
		echo "Building $$OUTPUT..."; \
		GOOS=$$GOOS GOARCH=$$GOARCH go build -ldflags "$(LDFLAGS)" -o "$$OUTPUT" . ; \
	done
	@echo ""
	@echo "Build complete. Binaries in $(BUILD_DIR)/:"
	@ls -lh $(BUILD_DIR)/

dist: build-all ## Build and package for distribution
	@echo ""
	@echo "Creating distribution archives..."
	@cd $(BUILD_DIR) && \
		for f in $(BINARY)_*; do \
			if echo $$f | grep -q '\.exe$$'; then \
				zip "$$f.zip" "$$f" && rm "$$f"; \
			else \
				tar -czf "$$f.tar.gz" "$$f" && rm "$$f"; \
			fi; \
		done
	@echo "Distribution archives ready in $(BUILD_DIR)/"
	@ls -lh $(BUILD_DIR)/

clean: ## Clean build artifacts
	rm -rf $(BUILD_DIR)
	rm -f $(BINARY) $(BINARY).exe

install: build ## Install to /usr/local/bin
	install -m 755 $(BINARY) /usr/local/bin/$(BINARY)
	@echo "Installed to /usr/local/bin/$(BINARY)"
