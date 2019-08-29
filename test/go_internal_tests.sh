#!/bin/bash

cd "$( dirname "${BASH_SOURCE[0]}" )/../"

ERRORS=0

error() {
	if [ $1 -gt 0 ]; then
		let ERRORS++
	fi
}

for application in $(find cmd/ -type d); do
	if ! ls $pkg/*test.go 1> /dev/null 2>&1; then
		continue
	fi
	go test "./$application"
	error $?
done

for pkg in $(find internal/ -type d); do
	if ! ls $pkg/*test.go 1> /dev/null 2>&1; then
		continue
	fi
	go test "./$pkg"
	error $?
done

if [ $ERRORS -gt 0 ]; then
	printf "\n\e[31mErrors occurred when running tests.\e[0m Please see above output for more information.\n"
	exit $ERRORS
fi

printf "\n\e[32mAll internal tests completed successfully.\e[0m\n\n"
exit 0
