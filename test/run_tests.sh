#!/bin/bash

####
# This is a simple testing script. It assumes that an exit code of 0 implies successful compilation
# or execution. It also does not output the errors to reduce clutter.
# At the end a summary of the tests is given. If a task fails, please execute it manually to find
# the issue.
####

# --------- define the files that should be compiled/run ---------------------
COMPILE_FILES=(
	"examples/testcomponent"
)

# these files should fail to compile
COMPILE_FILES_NEGATIVE=(
)

RUN_FILES=(
)

# only run these tests if we explicitly tell it to
if [ -n "$SLOW_TESTS" ]; then
    RUN_FILES+=(
    )
fi

# --------- setup the required variables -------------------------------------
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && cd ../ && pwd )"

FYRBASE="$DIR"

# ARCH=`bin/fyrarch`
echo $FYRBASE

COMPILE_ERRORS=""
COMPILE_FALSE_POSITIVE=""
RUN_ERRORS=""
RUN_LEAKS=""
EXIT=0

# --------- compile/run the files silently -----------------------------------
compile_positive() {
    for file in "${COMPILE_FILES[@]}"; do
        printf "%s: Compiling %s...\n" `date +%F_%T` $file
        $DIR/fyrc "$file" >/dev/null 2>&1
        if [ $? -ne 0 ]; then
            COMPILE_ERRORS="$COMPILE_ERRORS $file"
        fi
    done
}

compile_negative() {
    for file in "${COMPILE_FILES_NEGATIVE[@]}"; do
        printf "%s: Compiling %s should fail...\n" `date +%F_%T` $file
        $DIR/fyrc "$file" >/dev/null 2>&1
        if [ $? -eq 0 ]; then
            COMPILE_FALSE_POSITIVE="$COMPILE_FALSE_POSITIVE $file"
        fi
    done
}

run_files() {
    for file in "${RUN_FILES[@]}"; do
        printf "%s: Running %s...\n" `date +%F_%T` $file
        eval "$DIR/bin/$ARCH/$file" >/dev/null 2>&1
        if [ $? -ne 0 ]; then
            RUN_ERRORS="$RUN_ERRORS $file"
        fi
    done
}

run_files_valgrind() {
    if ! [ -x "$(command -v valgrind)" ]; then
        printf "\nUnable to check for memory leaks. Please install valgrind.\n"
        return
    fi
    for file in "${RUN_FILES[@]}"; do
        printf "%s: Checking %s for memory leaks...\n" `date +%F_%T` $file
        eval "valgrind --leak-check=yes -q $DIR/bin/$ARCH/$file" >/dev/null 2>&1
        if [ $? -ne 0 ]; then
            RUN_LEAKS="$RUN_LEAKS $file"
        fi
    done
}

# --------- output a summary -------------------------------------------------
output_results() {
    printf "\n"

    if [ -n "$COMPILE_ERRORS" ]; then
        printf "ERROR: %s did not compile successfully\n" $COMPILE_ERRORS
        EXIT=1
    fi

    if [ -n "$COMPILE_FALSE_POSITIVE" ]; then
        printf "ERROR: %s should not have compiled successfully\n" $COMPILE_FALSE_POSITIVE
        EXIT=1
    fi

    if [ -n "$RUN_ERRORS" ]; then
        printf "ERROR: %s did not run successfully\n" $RUN_ERRORS
        EXIT=1
    fi

    if [ -n "$RUN_LEAKS" ]; then
        printf "ERROR: %s may have leaked memory\n" $RUN_LEAKS
        EXIT=1
    fi

    if [ $EXIT -eq 0 ]; then
        printf "All tests completed successfully.\n"
    fi

    exit $EXIT
}

compile_positive
compile_negative

run_files
run_files_valgrind

output_results
