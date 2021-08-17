# add all your cmd/<things> in here
TARGETS = executable

# install linter
golangci-lint = ./bin/golangci-lint
$(golangci-lint):
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.27.0

# install gosec
gosec = ./bin/gosec
$(gosec):
	curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh | sh -s v2.3.0

CMD_DIR := ./cmd
PKG_DIR := ./pkg
OUT_DIR := ./out

COV_FILE := cover.out

GO111MODULE := on

GO_TEST_FLAGS := -v -count=1 -race -coverprofile=$(OUT_DIR)/$(COV_FILE) -covermode=atomic

.PHONY: $(OUT_DIR) clean build test mod cover test-deps fmt vet purge bench lint sec

all: clean mod fmt vet test build lint build-container build-and-publish-container sec

$(OUT_DIR):
	@mkdir -p $(OUT_DIR)

clean:
	@rm -rf $(OUT_DIR)

purge: clean
	go mod tidy
	go clean -cache
	go clean -testcache
	go clean -modcache

build: $(OUT_DIR)
	$(foreach target,$(TARGETS),go build -o $(OUT_DIR)/$(target) $(CMD_DIR)/$(target)/*.go;)

test: $(OUT_DIR)
	go test $(GO_TEST_FLAGS) ./...

mod:
	go mod tidy
	go mod verify

cover:
	go tool cover -html=$(OUT_DIR)/$(COV_FILE)

test-deps:
	go test all

fmt:
	go fmt ./...

vet:
	go vet ./...

bench:
	go test -bench=. -benchmem -benchtime=10s ./...

lint: 
	golangci-lint run

build-container:
	docker build --tag test_build .

sec: ## Security scan
	gosec -exclude G601,G404 ./...

