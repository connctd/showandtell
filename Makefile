VERSION                 ?= $(shell git describe --tags --always --dirty)
RELEASE_VERSION     		?= $(shell git describe --abbrev=0)
LDFLAGS         				?= -X github.com/connctd/showandtell.Version=$(VERSION) -w -s
GO_ENV 									 = GO111MODULE=on

REVEAL_JS_VERSION				= 3.8.0
REVEAL_JS_URL						= https://github.com/hakimel/reveal.js/archive/$(REVEAL_JS_VERSION).zip

GO_BUILD								= $(GO_ENV) go build -ldflags "$(LDFLAGS)"
GO_TEST									= $(GO_ENV) go test -v

.PHONY: clean dist-clean test dist build

build: dist/sat

dist/sat: dist_temp/reveal
	@mkdir -p ./dist
	packr2
	$(GO_BUILD) -o ./dist/sat ./cmd/sat
	packr2 clean

dist_temp/reveal:
	@mkdir -p ./dist_temp
	@wget -o /dev/null -O ./dist_temp/reveal.tar.gz $(REVEAL_JS_URL)
	@cd ./dist_temp/ && tar xzf reveal.tar.gz
	@mv ./dist_temp/reveal.js-$(REVEAL_JS_VERSION) ./dist_temp/reveal

dist: dist/sat 

test:
	@echo Running tests
	$(GO_TEST) ./...

clean: dist-clean
	@packr2 clean
	@rm -rf ./dist
	@rm -rf ./test_out

dist-clean:
	@rm -rf ./dist_temp
	
