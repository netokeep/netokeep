VERSION := 0.2.0

CLIENT_BINARY_NAME := "nk"
SERVER_BINARY_NAME := "nks"
PACKAGE_NAME := "netokeep"
DISPLAY_NAME := "NetoKeep"
BUILD_DIR := "./release/build"
RELEASE_DIR := "./release"

ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))


format:
	@gofmt -s -w .

build:
	@mkdir -p $(BUILD_DIR) && \
	go build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(CLIENT_BINARY_NAME) ./cmd/nk/main.go && \
	go build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(SERVER_BINARY_NAME) ./cmd/nks/main.go

pack:
	@mkdir -p $(RELEASE_DIR) && \
	cp ./setup.sh $(BUILD_DIR)/ && \
	chmod +x $(BUILD_DIR)/setup.sh && \
	makeself $(BUILD_DIR) $(RELEASE_DIR)/$(PACKAGE_NAME)-Linux-amd64.sh $(DISPLAY_NAME) ./setup.sh

# This command is used for test like: command_name ...
nk:
	@go run ./cmd/nk/main.go $(ARGS)

nks:
	@go run ./cmd/nks/main.go $(ARGS)


# To prevent make from attempting to build a second target, add the catch-all rule
%:
	@:
