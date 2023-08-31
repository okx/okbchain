maDEP := $(shell command -v dep 2> /dev/null)
SUM := $(shell which shasum)

COMMIT := $(shell git rev-parse HEAD)
CAT := $(if $(filter $(OS),Windows_NT),type,cat)
export GO111MODULE=on

GithubTop=github.com

GO_VERSION=1.20
ROCKSDB_VERSION=6.27.3
IGNORE_CHECK_GO=false
install_rocksdb_version:=$(ROCKSDB_VERSION)


Version=v0.1.7
CosmosSDK=v0.39.2
Tendermint=v0.33.9
Iavl=v0.14.3
Name=brczero
ServerName=brczerod
ClientName=brczerocli

LINK_STATICALLY = false
cgo_flags=

ifeq ($(IGNORE_CHECK_GO),true)
    GO_VERSION=0
endif

# process linker flags
ifeq ($(VERSION),)
    VERSION = $(COMMIT)
endif

ifeq ($(MAKECMDGOALS),mainnet)
   WITH_ROCKSDB=true
else ifeq ($(MAKECMDGOALS),testnet)
    WITH_ROCKSDB=true
endif

build_tags = netgo

system=$(shell $(shell pwd)/libs/scripts/system.sh)
ifeq ($(system),alpine)
  ifeq ($(LINK_STATICALLY),false)
      $(warning Your system is alpine. It must be compiled statically. Now we start compile statically.)
  endif
  LINK_STATICALLY=true
else
  ifeq ($(LINK_STATICALLY),true)
      $(error your system is $(system) which can not be complied statically. please set LINK_STATICALLY=false)
  endif
endif

ifeq ($(WITH_ROCKSDB),true)
  CGO_ENABLED=1
  build_tags += rocksdb
  ifeq ($(LINK_STATICALLY),true)
      cgo_flags += CGO_CFLAGS="-I/usr/include/rocksdb"
      cgo_flags += CGO_LDFLAGS="-L/usr/lib -lrocksdb -lstdc++ -lm  -lsnappy -llz4"
  endif
else
  ROCKSDB_VERSION=0
endif

ifeq ($(LINK_STATICALLY),true)
  build_tags += muslc
  dummy := $(shell $(shell pwd)/libs/scripts/wasm_static_install.sh)
endif

build_tags += $(BUILD_TAGS)
build_tags := $(strip $(build_tags))

ldflags = -X $(GithubTop)/okx/brczero/libs/cosmos-sdk/version.Version=$(Version) \
	-X $(GithubTop)/okx/brczero/libs/cosmos-sdk/version.Name=$(Name) \
  -X $(GithubTop)/okx/brczero/libs/cosmos-sdk/version.ServerName=$(ServerName) \
  -X $(GithubTop)/okx/brczero/libs/cosmos-sdk/version.ClientName=$(ClientName) \
  -X $(GithubTop)/okx/brczero/libs/cosmos-sdk/version.Commit=$(COMMIT) \
  -X $(GithubTop)/okx/brczero/libs/cosmos-sdk/version.CosmosSDK=$(CosmosSDK) \
  -X $(GithubTop)/okx/brczero/libs/cosmos-sdk/version.Tendermint=$(Tendermint) \
  -X "$(GithubTop)/okx/brczero/libs/cosmos-sdk/version.BuildTags=$(build_tags)" \

ifeq ($(WITH_ROCKSDB),true)
  ldflags += -X github.com/okx/brczero/libs/tendermint/types.DBBackend=rocksdb
endif

ifeq ($(MAKECMDGOALS),testnet)
  ldflags += -X github.com/okx/brczero/libs/cosmos-sdk/server.ChainID=brczerotest-195
endif

ifeq ($(LINK_STATICALLY),true)
	ldflags += -linkmode=external -extldflags "-Wl,-z,muldefs -static"
endif

ifeq ($(BRCZEROMALLOC),tcmalloc)
  ldflags += -extldflags "-ltcmalloc_minimal"
endif

ifeq ($(BRCZEROMALLOC),jemalloc)
  ldflags += -extldflags "-ljemalloc"
endif

BUILD_FLAGS := -ldflags '$(ldflags)'

ifeq ($(DEBUG),true)
	BUILD_FLAGS += -gcflags "all=-N -l"
endif

ifeq ($(PGO),true)
	PGO_AUTO = -pgo=auto
endif

all: install

install: brczero


brczero: check_version
	$(cgo_flags) go install $(PGO_AUTO) -v $(BUILD_FLAGS) -tags "$(build_tags)" ./cmd/brczerod
	$(cgo_flags) go install $(PGO_AUTO) -v $(BUILD_FLAGS) -tags "$(build_tags)" ./cmd/brczerocli

check_version:
	@sh $(shell pwd)/libs/check/check-version.sh $(GO_VERSION) $(ROCKSDB_VERSION)

mainnet: brczero

testnet: brczero

test-unit:
	@VERSION=$(VERSION) go test -mod=readonly -tags='ledger test_ledger_mock' ./app/...
	@VERSION=$(VERSION) go test -mod=readonly -tags='ledger test_ledger_mock' ./x/common/...
	@VERSION=$(VERSION) go test -mod=readonly -tags='ledger test_ledger_mock' ./x/distribution/...
	@VERSION=$(VERSION) go test -mod=readonly -tags='ledger test_ledger_mock' ./x/genutil/...
	@VERSION=$(VERSION) go test -mod=readonly -tags='ledger test_ledger_mock' ./x/gov/...
	@VERSION=$(VERSION) go test -mod=readonly -tags='ledger test_ledger_mock' ./x/params/...
	@VERSION=$(VERSION) go test -mod=readonly -tags='ledger test_ledger_mock' ./x/staking/...
	@VERSION=$(VERSION) go test -mod=readonly -tags='ledger test_ledger_mock' ./x/token/...
	@VERSION=$(VERSION) go test -mod=readonly -tags='ledger test_ledger_mock' ./x/upgrade/...

get_vendor_deps:
	@echo "--> Generating vendor directory via dep ensure"
	@rm -rf .vendor-new
	@dep ensure -v -vendor-only

update_vendor_deps:
	@echo "--> Running dep ensure"
	@rm -rf .vendor-new
	@dep ensure -v -update

go-mod-cache: go.sum
	@echo "--> Download go modules to local cache"
	@go mod download
.PHONY: go-mod-cache

go.sum: go.mod
	@echo "--> Ensure dependencies have not been modified"
	@go mod verify
	@go mod tidy

cli:
	go install -v $(BUILD_FLAGS) -tags "$(build_tags)" ./cmd/brczerocli

server:
	go install -v $(BUILD_FLAGS) -tags "$(build_tags)" ./cmd/brczerod

format:
	find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" -not -path "./client/lcd/statik/statik.go" | xargs gofmt -w -s

build:
ifeq ($(OS),Windows_NT)
	go build $(PGO_AUTO) $(BUILD_FLAGS) -tags "$(build_tags)" -o build/brczerod.exe ./cmd/brczerod
	go build $(PGO_AUTO) $(BUILD_FLAGS) -tags "$(build_tags)" -o build/brczerocli.exe ./cmd/brczerocli
else
	go build $(PGO_AUTO) $(BUILD_FLAGS) -tags "$(build_tags)" -o build/brczerod ./cmd/brczerod
	go build $(PGO_AUTO) $(BUILD_FLAGS) -tags "$(build_tags)" -o build/brczerocli ./cmd/brczerocli
endif


test:
	go list ./app/... |xargs go test -count=1
	go list ./x/... |xargs go test -count=1
	go list ./libs/cosmos-sdk/... |xargs go test -count=1 -tags='norace ledger test_ledger_mock'
	go list ./libs/tendermint/... |xargs go test -count=1
	go list ./libs/tm-db/... |xargs go test -count=1
	go list ./libs/iavl/... |xargs go test -count=1
	go list ./libs/ibc-go/... |xargs go test -count=1

testapp:
	go list ./app/... |xargs go test -count=1

testx:
	go list ./x/... |xargs go test -count=1

testcm:
	go list ./libs/cosmos-sdk/... |xargs go test -count=1 -tags='norace ledger test_ledger_mock'

testtm:
	go list ./libs/tendermint/... |xargs go test -count=1 -tags='norace ledger test_ledger_mock'

testibc:
	go list ./libs/ibc-go/... |xargs go test -count=1 -tags='norace ledger test_ledger_mock'


build-linux:
	LEDGER_ENABLED=false GOOS=linux GOARCH=amd64 $(MAKE) build

build-docker-brczeronode:
	$(MAKE) -C networks/local

# Run a 4-node testnet locally
localnet-start: localnet-stop
	@if ! [ -f build/node0/brczerod/config/genesis.json ]; then docker run --rm -v $(CURDIR)/build:/brczerod:Z brczero/node testnet --v 4 -o . --starting-ip-address 192.168.10.2 --keyring-backend=test ; fi
	docker-compose up -d

# Stop testnet
localnet-stop:
	docker-compose down

rocksdb:
	@echo "Installing rocksdb..."
	@bash ./libs/rocksdb/install.sh --version v$(install_rocksdb_version)
.PHONY: rocksdb

.PHONY: build

tcmalloc:
	@echo "Installing tcmalloc..."
	@bash ./libs/malloc/tcinstall.sh

jemalloc:
	@echo "Installing jemalloc..."
	@bash ./libs/malloc/jeinstall.sh
