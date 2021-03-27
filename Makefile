CI_RUN?=false
ADDITIONAL_BUILD_FLAGS=""

ifeq ($(CI_RUN), true)
	ADDITIONAL_BUILD_FLAGS="-test.short"
endif

prepare:
	go get github.com/go-critic/go-critic/cmd/gocritic

test:
	gocritic check
	go test -race $(ADDITIONAL_BUILD_FLAGS) -v -coverprofile coverage.txt -covermode=atomic

lint:
	go fmt