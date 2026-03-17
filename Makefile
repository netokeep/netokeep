VERSION := 0.4.1
PACKAGE_NAME := netokeep
DISPLAY_NAME := "NetoKeep"

# Directory definitions
BUILD_ROOT := ./release/build
RELEASE_ROOT := ./release
TEMP_DIR := ./release/temp

# Target definitions
CLIENT_SRC := ./cmd/nk/main.go
SERVER_SRC := ./cmd/nks/main.go

# Build flags
LDFLAGS := -s -w -X main.version=$(VERSION)

# Platform matrix: OS/Arch
PLATFORMS := linux/amd64 darwin/amd64 windows/amd64

.PHONY: all clean build pack format

all: clean build pack

format:
	@echo "🎨 Formatting code..."
	@gofmt -s -w .

clean:
	@echo "🧹 Cleaning old releases..."
	@rm -rf $(RELEASE_ROOT)

# Build for all platforms
build: format
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
		output_dir=$(BUILD_ROOT)/$$os-$$arch; \
		echo "🔨 Building $$os/$$arch..."; \
		mkdir -p $$output_dir; \
		GOOS=$$os GOARCH=$$arch go build -ldflags "$(LDFLAGS)" -o $$output_dir/nk$$ext $(CLIENT_SRC); \
		GOOS=$$os GOARCH=$$arch go build -ldflags "$(LDFLAGS)" -o $$output_dir/nks$$ext $(SERVER_SRC); \
	done

# Unified packaging logic
pack:
	@mkdir -p $(RELEASE_ROOT)
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		build_dir=$(BUILD_ROOT)/$$os-$$arch; \
		if [ "$$os" = "windows" ]; then \
			echo "📦 Zipping Windows $$arch..."; \
			cd $$build_dir && zip -q -r ../../$(PACKAGE_NAME)-Windows-$$arch.zip ./*; cd - > /dev/null; \
		else \
			echo "📦 Making self-extracting installer for $$os/$$arch..."; \
			cp ./setup.sh $$build_dir/; \
			chmod +x $$build_dir/setup.sh; \
			makeself --quiet $$build_dir $(RELEASE_ROOT)/$(PACKAGE_NAME)-$$os-$$arch.sh $(DISPLAY_NAME) ./setup.sh; \
		fi; \
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
