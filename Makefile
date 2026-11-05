VERSION := 1.2.6
PACKAGE_NAME := netokeep
DISPLAY_NAME := "NetoKeep"

# Directory definitions
RELEASE_ROOT := ./release

# Target definitions
CLIENT_SRC := ./cmd/nk
SERVER_SRC := ./cmd/nks

# Build flags
LDFLAGS := -s -w -X main.version=$(VERSION)

# Platform matrix: OS/Arch
PLATFORMS := linux/amd64 darwin/amd64 darwin/arm64 windows/amd64

.PHONY: all format clean build pack nk nks

all: format clean build pack

format:
	@echo "🎨 Formatting code..."
	@gofmt -s -w .

clean:
	@echo "🧹 Cleaning old releases..."
	@rm -rf $(RELEASE_ROOT)

test:
	@echo "🧪 Running tests..."
	@go test ./...

build: format
	@set -e; for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		suffix=""; \
		if [ "$$os" = "windows" ]; then suffix=".exe"; fi; \
		binary_suffix="$$os-$$arch$$suffix"; \
		echo "🔨 Building $$os/$$arch..."; \
		mkdir -p $(RELEASE_ROOT); \
		GOOS=$$os GOARCH=$$arch go build -ldflags "$(LDFLAGS)" -o $(RELEASE_ROOT)/netokeep-client-$$binary_suffix $(CLIENT_SRC); \
		if [ "$$os" = "linux" ]; then \
			GOOS=$$os GOARCH=$$arch go build -ldflags "$(LDFLAGS)" -o $(RELEASE_ROOT)/netokeep-server-$$binary_suffix $(SERVER_SRC); \
		fi; \
	done

# Debug commands
ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
nk:
	@go run $(CLIENT_SRC) $(ARGS)
nks:
	@go run $(SERVER_SRC) $(ARGS)

%:
	@:
