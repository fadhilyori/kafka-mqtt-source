SHELL:=bash
GOCMD=go
GOTEST=$(GOCMD) test
GOVET=$(GOCMD) vet
BINARY_NAME=me-mqtt-source
VERSION?=1.1
TARGET_PLATFORMS=linux/amd64
DOCKER_REPO_URL=mataelang/kafka-mqtt-source

comma:= ,
empty:=
space:= $(empty) $(empty)
GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
WHITE  := $(shell tput -Txterm setaf 7)
CYAN   := $(shell tput -Txterm setaf 6)
RESET  := $(shell tput -Txterm sgr0)

build: vendor build-linux ## Build the project and put the output binary in out/bin/

build-linux:
	@go mod tidy
	@$(foreach platform, $(TARGET_PLATFORMS), \
		echo "Compiling for $(platform)"; \
		GOOS=$(word 1,$(subst /, ,$(platform))) GOARCH=$(word 2,$(subst /, ,$(platform))) GO111MODULE=on CGO_ENABLED=1 $(GOCMD) build -o out/bin/$(BINARY_NAME)-$(word 1,$(subst /, ,$(platform)))-$(word 2,$(subst /, ,$(platform))) ./cmd/ ;\
	)
	
build-docker-multiarch:
	@echo "Building docker image for platform: $(TARGET_PLATFORMS)"
	@docker buildx build --platform $(subst $(space),$(comma),$(TARGET_PLATFORMS)) -t $(DOCKER_REPO_URL):latest -t $(DOCKER_REPO_URL):$(VERSION) --push .

clean: ## Remove build related file
	@rm -rf ./bin
	@rm -rf ./out
	@echo "Any build output removed."

vendor: ## Copy of all packages needed to support builds in the vendor directory
	@ $(GOCMD) mod vendor

run: ## Run with go run
	@go run main.go

help: ## Show this help.
	@echo ''
	@echo 'Usage:'
	@echo '  ${YELLOW}make${RESET} ${GREEN}<target>${RESET}'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} { \
		if (/^[a-zA-Z_-]+:.*?##.*$$/) {printf "    ${YELLOW}%-20s${GREEN}%s${RESET}\n", $$1, $$2} \
		else if (/^## .*$$/) {printf "  ${CYAN}%s${RESET}\n", substr($$1,4)} \
		}' $(MAKEFILE_LIST)
