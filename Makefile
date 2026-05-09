.PHONY: build test install lint clean run

BINARY_NAME=repox
BUILD_DIR=bin

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/repox

test:
	go test ./... -v -count=1

install:
	go install ./cmd/repox

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BUILD_DIR)

run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

# Quick test: init + generate in a temp directory
demo: build
	@echo "=== Repox Demo ==="
	@rm -rf /tmp/repox-demo
	@mkdir -p /tmp/repox-demo
	@cd /tmp/repox-demo && $(CURDIR)/$(BUILD_DIR)/$(BINARY_NAME) init
	@cd /tmp/repox-demo && $(CURDIR)/$(BUILD_DIR)/$(BINARY_NAME) generate feature watchlist
	@echo "=== Done ==="
	@find /tmp/repox-demo -type f | sort
