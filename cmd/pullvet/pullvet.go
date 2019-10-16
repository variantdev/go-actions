package main

import (
	"flag"
	"fmt"
	"github.com/google/go-github/github"
	"github.com/variantdev/go-actions"
	"github.com/variantdev/go-actions/pkg/pullvet"
	"os"
)

// pullvet checks for the existence of the specified pull request label(s)  exists with a non-zero status
// whenever one ore more required labels are missing in the pull request
//
// This should be useful for compliance purpose. that is, it will help preventing any  pr from being merged when it misses required labels.
// when run on GitHub Actions v2
func main() {
	cmd := pullvet.NewCommand()

	fs := flag.CommandLine
	fs.Init("pullvet", flag.ExitOnError)
	cmd.AddFlags(fs)

	flag.Parse()

	var pr *github.PullRequest

	checkRun, err := actions.CheckRunEvent()
	if err != nil {
		pull, err := actions.PullRequestEvent()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
		pr = pull.PullRequest
	} else {
		pr = checkRun.CheckRun.PullRequests[0]
	}

	if err := cmd.Run(pr); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
