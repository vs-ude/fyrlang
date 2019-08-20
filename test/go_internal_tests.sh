#!/bin/bash

cd "$( dirname "${BASH_SOURCE[0]}" )/../"

ERRORS=0

error() {
	if [ $1 -gt 0 ]; then
		let ERRORS++
	fi
}

for application in cmd/*/; do
	go test "./$application"
	error $?
done

for pkg in internal/*/; do
	go test "./$pkg"
	error $?
done

if [ $ERRORS -gt 0 ]; then
	printf "\nErrors occurred when running tests. Please see above output for more information.\n"
	exit $ERRORS
fi

printf "\nAll tests completed successfully.\n"
exit 0
