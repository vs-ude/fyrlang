prefix ?= /usr
datadir ?= $(prefix)/share

compiler_binaries := fyrc fyrarch

export FYRBASE = ${CURDIR}

NATIVE_LIBS_REPO ?= https://github.com/vs-ude/fyrlang-native-libs.git

SOURCE_DATE_EPOCH ?= $(shell date +%F\ %T)
buildFlags = -ldflags "-X 'main.buildDate=${SOURCE_DATE_EPOCH}'"

.PHONY: all
all: build

.PHONY: build
build: ${compiler_binaries}

.PHONY: ${compiler_binaries}
${compiler_binaries}:
	go build ${buildFlags} ./cmd/$@

.PHONY: run
run: fyrc
	./fyrc

.PHONY: test
test: test_go test_fyr

.PHONY: test_go
test_go:
	@go test ./... || \
	(printf "\n\e[31mErrors occurred when running tests.\e[0m Please see above output for more information.\n\n" && exit 1) && \
	printf "\n\e[32mAll internal tests completed successfully.\e[0m\n\n"

.PHONY: test_fyr
test_fyr: build clean_examples
	@./test/fyr_code_tests.sh

.PHONY: setup
setup: setup_libs

.PHONY: setup_libs
setup_libs:
	if [ ! -e 'lib/native' ]; then git clone ${NATIVE_LIBS_REPO} lib/native; fi
	cd lib/native; git pull
	$(MAKE) --directory=lib/native

.PHONY: clean
clean: clean_compiler clean_examples

.PHONY: clean_compiler
clean_compiler:
	rm -rf ${compiler_binaries}

.PHONY: clean_examples
clean_examples:
	find examples lib -type d -name 'pkg' ! -path 'lib/native/*' -exec rm -rf {} +
	find examples -type d -name 'bin' ! -path 'lib/native/*' -exec rm -rf {} +
