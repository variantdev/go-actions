package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/variantdev/go-actions/pkg/checks"
)

func main() {
	cmd := checks.New()

	fs := flag.CommandLine
	fs.Init("checks", flag.ExitOnError)
	cmd.AddFlags(fs)

	flag.Parse()

	if err := cmd.Run(fs.Args()); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
