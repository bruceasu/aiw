package main

import (
	"fmt"
	"os"
)

func main() {
	o, err := parseOptions(os.Args[1:])
	if err == nil && o.help {
		fmt.Print(helpText)
		return
	}
	if err == nil {
		err = execute(o, os.Stdin, os.Stdout)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "aiw-packer:", err)
		os.Exit(1)
	}
}
