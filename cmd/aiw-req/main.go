package main

import (
	"fmt"
	"os"

	airuntime "aiw/internal/runtime"
)

// The req plugin owns the public Requirement CLI and its domain model.
func main() {
	airuntime.ConfigureProduction()
	if err := Dispatch(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "req:", err)
		os.Exit(1)
	}
}
