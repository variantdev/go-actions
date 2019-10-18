package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/variantdev/go-actions/pkg/exec"
)

func main() {
	cmd := exec.New()

	fs := flag.CommandLine
	fs.Init("exec", flag.ExitOnError)
	cmd.AddFlags(fs)

	flag.Parse()

	if err := cmd.Run(fs.Args()); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
