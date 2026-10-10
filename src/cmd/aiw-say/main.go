package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "aiw say:", err)
		os.Exit(1)
	}
}
