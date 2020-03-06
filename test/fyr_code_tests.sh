#!/bin/bash

####
# This is a simple testing script. It assumes that an exit code of 0 implies successful compilation
# or execution. It also does not output the errors to reduce clutter.
# At the end a summary of the tests is given. If a task fails, please execute it manually to find
# the issue.
# IMPORTANT: the modules are assumed to be located under examples/
####

# --------- define the modules that should be compiled/run ---------------------
COMPILE_FILES=(
	"grouptest"
	"grouptest2"
	"grouptest5"
	"helloworld"
	"mandelbrot"
	"testcomponent"
	"strings"
)

# these modules should fail to compile
COMPILE_FILES_NEGATIVE=(
	"collections"
	"opengl"
)

RUN_FILES=(
	"grouptest"
	"grouptest2"
	"helloworld"
	"testcomponent"
	"strings"
)

# only run these tests if we explicitly tell it to
if [ -n "$SLOW_TESTS" ]; then
	RUN_FILES+=(
		# "mandelbrot"
	)
fi

# --------- setup configured variables ---------------------------------------
if [ -n "$CC" ]; then
	printf "\e[33mINFO:\e[0m Using %s C compiler\n\n" $CC
	CC_FLAG="-nc $CC"
else
	CC_FLAG=""
fi

# --------- setup the required variables -------------------------------------
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && cd ../ && pwd )"

FYRBASE="$DIR"

TARGET_ARCH="$($DIR/fyrarch)"

COMPILE_ERRORS=""
COMPILE_FALSE_POSITIVE=""
RUN_ERRORS=""
RUN_LEAKS=""
EXIT=0

# --------- compile/run the modules silently -----------------------------------
compile_positive() {
	for module in "${COMPILE_FILES[@]}"; do
		printf "%s: Compiling %s...\n" `date +%F_%T` $module
		$DIR/fyrc -n $CC_FLAG "examples/$module" >/dev/null 2>&1
		if [ $? -ne 0 ]; then
			COMPILE_ERRORS="$COMPILE_ERRORS $module"
		fi
	done
}

compile_negative() {
	for module in "${COMPILE_FILES_NEGATIVE[@]}"; do
		printf "%s: Compiling %s should fail...\n" `date +%F_%T` $module
		$DIR/fyrc -n $CC_FLAG "examples/$module" >/dev/null 2>&1
		if [ $? -eq 0 ]; then
			COMPILE_FALSE_POSITIVE="$COMPILE_FALSE_POSITIVE $module"
		fi
	done
}

run_modules() {
	for module in "${RUN_FILES[@]}"; do
		printf "%s: Running %s...\n" `date +%F_%T` $module
		eval "examples/$module/bin/$TARGET_ARCH/$module" >/dev/null 2>&1
		if [ $? -ne 0 ]; then
			RUN_ERRORS="$RUN_ERRORS $module"
		fi
	done
}

run_modules_valgrind() {
	if ! [ -x "$(command -v valgrind)" ]; then
		printf "\nUnable to check for memory leaks. Please install valgrind.\n"
		return
	fi
	for module in "${RUN_FILES[@]}"; do
		printf "%s: Checking %s for memory leaks...\n" `date +%F_%T` $module
		eval "valgrind --leak-check=yes -q examples/$module/bin/$TARGET_ARCH/$module" >/dev/null 2>&1
		if [ $? -ne 0 ]; then
			RUN_LEAKS="$RUN_LEAKS $module"
		fi
	done
}

# --------- output a summary -------------------------------------------------
output_results() {
	printf "\n"

	if [ -n "$COMPILE_ERRORS" ]; then
		printf "\e[31mERROR:\e[0m %s did not compile successfully\n" $COMPILE_ERRORS
		EXIT=1
	fi

	if [ -n "$COMPILE_FALSE_POSITIVE" ]; then
		printf "\e[31mERROR:\e[0m %s should not have compiled successfully\n" $COMPILE_FALSE_POSITIVE
		EXIT=1
	fi

	if [ -n "$RUN_ERRORS" ]; then
		printf "\e[31mERROR:\e[0m %s did not run successfully\n" $RUN_ERRORS
		EXIT=1
	fi

	if [ -n "$RUN_LEAKS" ]; then
		printf "\e[31mERROR:\e[0m %s may have leaked memory\n" $RUN_LEAKS
		EXIT=1
	fi

	if [ $EXIT -eq 0 ]; then
		printf "\n\e[32mAll compiler tests completed successfully.\e[0m\n\n"
	fi

	exit $EXIT
}

compile_positive
compile_negative

run_modules
run_modules_valgrind

output_results
