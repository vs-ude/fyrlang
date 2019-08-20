prefix ?= /usr
datadir ?= $(prefix)/share

.PHONY: all
all: build

.PHONY: build
build:
	go build ./cmd/fyrc

.PHONY: run
run: build
	./fyrc
