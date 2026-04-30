BINARY_NAME := switchboard
GO          := go
GOFLAGS     :=

.PHONY: all build test lint fmt coverage clean run

all: lint test build

## build: Compile the binary
build:
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME) .

## run: Build and run the application
run: build
	./$(BINARY_NAME)

## test: Run all unit tests
test:
	$(GO) test $(GOFLAGS) ./...

## coverage: Run tests and produce an HTML coverage report
coverage:
	$(GO) test $(GOFLAGS) -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

## lint: Run go vet and check formatting
lint:
	$(GO) vet ./...
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "The following files are not gofmt-formatted:"; \
		gofmt -l .; \
		exit 1; \
	fi

## fmt: Format all Go source files
fmt:
	gofmt -w .

## clean: Remove build artifacts
clean:
	rm -f $(BINARY_NAME) coverage.out coverage.html
