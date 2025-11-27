# ChaosNTPd Makefile

BINARY_NAME=chaosntpd
TEST_CLIENT=test_client
MONITOR_CLIENT=monitor_client

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOMOD=$(GOCMD) mod
GOCLEAN=$(GOCMD) clean

# Build directories
BUILD_DIR=build

.PHONY: all build daemon test-client monitor-client clean deps

all: build

build: daemon test-client monitor-client

daemon:
	$(GOBUILD) -o $(BINARY_NAME) .

test-client:
	$(GOBUILD) -o $(TEST_CLIENT) ./cmd/test_client

monitor-client:
	$(GOBUILD) -o $(MONITOR_CLIENT) ./cmd/monitor_client

deps:
	$(GOMOD) download

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME) $(TEST_CLIENT) $(MONITOR_CLIENT)

# Cross-compilation targets
.PHONY: build-linux build-darwin build-windows

build-linux:
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME)-linux-amd64 .

build-darwin:
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(BINARY_NAME)-darwin-arm64 .

build-windows:
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME)-windows-amd64.exe .
