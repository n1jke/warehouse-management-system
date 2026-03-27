COVERAGE_FILE ?= coverage.out

TARGET_PKG ?= cmd/app
BINARY_NAME ?= app
WORK_DIR ?= ## keep the default value as current directory if not set
override WORK_DIR := $(or $(strip $(WORK_DIR)),.)

# Build
.PHONY: build
build:
	@echo "go build -o $(WORK_DIR)/bin/$(BINARY_NAME) ./$(TARGET_PKG) (dir: $(WORK_DIR))"
	@mkdir -p "$(WORK_DIR)/bin"
	@cd "$(WORK_DIR)" && go build -o "$(abspath $(WORK_DIR)/bin/$(BINARY_NAME))" ./$(TARGET_PKG)

.PHONY: run
run: build
	@"$(WORK_DIR)/bin/$(BINARY_NAME)"

# Test
.PHONY: test
test:
	@cd "$(WORK_DIR)" && go test -coverprofile='$(abspath $(WORK_DIR)/$(COVERAGE_FILE))' ./...
	@go tool cover -func='$(WORK_DIR)/$(COVERAGE_FILE)' | grep ^total | tr -s '\t'

.PHONY: test_race
test_race:
	@cd "$(WORK_DIR)" && go test --race -coverprofile='$(abspath $(WORK_DIR)/$(COVERAGE_FILE))' ./...
	@go tool cover -func='$(WORK_DIR)/$(COVERAGE_FILE)' | grep ^total | tr -s '\t'

.PHONY: html_test
html_test:
	@go tool cover -html='$(WORK_DIR)/$(COVERAGE_FILE)' -o "$(WORK_DIR)/coverage.html"
	@echo "Coverage report saved to $(WORK_DIR)/coverage.html"

# Lint
.PHONY: fmt
fmt:
	@echo "cd $(WORK_DIR) && go fmt ./..."
	@cd "$(WORK_DIR)" && go fmt ./...

.PHONY: lint
lint:
	@golangci-lint --version && echo "cd $(WORK_DIR) && golangci-lint -v run --fix ./..." || echo "golangci-lint not found"
	@cd "$(WORK_DIR)" && golangci-lint -v run --fix ./...

# Cleanup
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf "$(WORK_DIR)/bin"
	@rm -f "$(WORK_DIR)/$(COVERAGE_FILE)"
	@rm -f "$(WORK_DIR)/coverage.html"