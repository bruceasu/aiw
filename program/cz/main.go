package main

import (
	"fmt"
	"os"

	czcmd "aiw-cz/cmd/cz"
)

func main() {
	if err := czcmd.Dispatch(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
