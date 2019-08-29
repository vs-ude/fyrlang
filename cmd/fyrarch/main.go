package main

import (
	"fmt"
	"runtime"
)

func main() {
	fmt.Println(runtime.GOOS + "_" + runtime.GOARCH)
}
