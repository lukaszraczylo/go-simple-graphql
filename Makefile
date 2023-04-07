CI_RUN?=false
ADDITIONAL_BUILD_FLAGS=""

ifeq ($(CI_RUN), true)
	ADDITIONAL_BUILD_FLAGS="-test.short"
endif

.PHONY: help
help:  ## display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

.PHONY: test
test: ## run tests on library
	@LOG_LEVEL=debug go test $(ADDITIONAL_BUILD_FLAGS) -v -cover ./... -race

.PHONY: test-packages
test-packages: ## run tests on packages
	@go test -v -cover ./pkg/...

.PHONY: all
all: test-packages test

.PHONY: update
update: ## update dependencies
	@go get -u -v ./...
	@go mod tidy -v

sonar:
	sonar-scanner \
  -Dsonar.projectKey=golang-simple-graphql \
  -Dsonar.sources=. \
  -Dsonar.host.url=http://localhost:9000 \
  -Dsonar.login=555ba78b315faa07b3231a3be0f3cbb24ab33814