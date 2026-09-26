package main

import (
	"fmt"
	"os"

	cz "aiw/internal/cz"
)

// The standalone cz program owns the Conventional Commit wizard. The root
// aiw executable discovers and launches it as the aiw-cz plugin.
func main() {
	if err := cz.Dispatch(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "cz:", err)
		os.Exit(1)
	}
}
