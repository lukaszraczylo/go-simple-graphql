CI_RUN?=false
ADDITIONAL_BUILD_FLAGS=""

ifeq ($(CI_RUN), true)
	ADDITIONAL_BUILD_FLAGS="-test.short"
endif

prepare:
	go get -u -v github.com/go-critic/go-critic/cmd/gocritic

test:
	# gocritic check
	go test $(ADDITIONAL_BUILD_FLAGS) -v -coverprofile cover.out -memprofile mem.out

lint:
	go fmt