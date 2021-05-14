CI_RUN?=false
ADDITIONAL_BUILD_FLAGS=""
SET_PRIVATE="github.com/telegram-bot-app/*"

ifeq ($(CI_RUN), true)
	ADDITIONAL_BUILD_FLAGS="-test.short"
endif

prepare:
	GOPRIVATE=$(SET_PRIVATE) go get github.com/go-critic/go-critic/cmd/gocritic
	gocritic check

test: lint tidy
	GOPRIVATE=$(SET_PRIVATE) go test -race $(ADDITIONAL_BUILD_FLAGS) -cover

lint:
	go fmt

readme:
	godocdown > README.md

update:
	GOPRIVATE=$(SET_PRIVATE) go get -u ./...

tidy:
	GOPRIVATE=$(SET_PRIVATE) go mod tidy