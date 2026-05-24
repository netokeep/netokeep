VERSION := 1.0.2-alpha
PACKAGE_NAME := netokeep
DISPLAY_NAME := "NetoKeep"

# Directory definitions
BUILD_ROOT := ./release/build
RELEASE_ROOT := ./release

# Target definitions
CLIENT_SRC := ./cmd/nk
SERVER_SRC := ./cmd/nks

# Build flags
LDFLAGS := -s -w -X main.version=$(VERSION)

# Platform matrix: OS/Arch
PLATFORMS := linux/amd64 darwin/amd64 darwin/arm64 windows/amd64

.PHONY: all format clean build pack nk nks

all: clean build pack

format:
	@echo "🎨 Formatting code..."
	@gofmt -s -w .

clean:
	@echo "🧹 Cleaning old releases..."
	@rm -rf $(RELEASE_ROOT)

build: format
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		output_dir=$(BUILD_ROOT)/$$os-$$arch; \
		echo "🔨 Building $$os/$$arch..."; \
		mkdir -p $$output_dir; \
		GOOS=$$os GOARCH=$$arch go build -ldflags "$(LDFLAGS)" -o $$output_dir/nk $(CLIENT_SRC); \
		GOOS=$$os GOARCH=$$arch go build -ldflags "$(LDFLAGS)" -o $$output_dir/nks $(SERVER_SRC); \
	done

# Unified packaging logic
pack:
	@mkdir -p $(RELEASE_ROOT)
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		build_dir=$(BUILD_ROOT)/$$os-$$arch; \
		cp $(PWD)/cmd/installer/main.go $$build_dir/main.go; \
		ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
		echo "📦 Building installer for $$os/$$arch..."; \
		GOOS=$$os GOARCH=$$arch go build -ldflags "$(LDFLAGS)" \
		-o $(RELEASE_ROOT)/$(PACKAGE_NAME)-$$os-$$arch-installer$$ext $$build_dir/main.go; \
	done
	@rm -rf $(BUILD_ROOT) # Clean up the temporary binary directory after packaging

# Debug commands
ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
nk:
	@go run $(CLIENT_SRC) $(ARGS)
nks:
	@go run $(SERVER_SRC) $(ARGS)

%:
	@:
