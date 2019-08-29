prefix ?= /usr
datadir ?= $(prefix)/share

.PHONY: all
all: build

.PHONY: build
build:
	go build ./cmd/fyrc

.PHONY: build_fyrarch
build_fyrarch:
	go build ./cmd/fyrarch

.PHONY: run
run: build
	./fyrc

.PHONY: test
test: test_go test_fyr

.PHONY: test_go
test_go:
	./test/go_internal_tests.sh

.PHONY: test_fyr
test_fyr: build build_fyrarch
	./test/fyr_code_tests.sh
