package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/variantdev/go-actions/pkg/pullvet"
)

// pullvet checks for the existence of the specified pull request label(s)  exists with a non-zero status
// whenever one ore more required labels are missing in the pull request
//
// This should be useful for compliance purpose. that is, it will help preventing any  pr from being merged when it misses required labels.
// when run on GitHub Actions v2
func main() {
	cmd := pullvet.New()

	fs := flag.CommandLine
	fs.Init("pullvet", flag.ExitOnError)
	cmd.AddFlags(fs)

	flag.Parse()

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
