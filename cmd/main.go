package main

import (
	"flag"
	"fmt"
	"github.com/variantdev/go-actions"
	"github.com/variantdev/go-actions/pkg/pullvet"
	"os"
)

func flagUsage() {
	text := `actions A collection of usable commands for GitHub v2 Actions

Usage:
  actions [command]
Available Commands:
  pullvet	checks labels and milestones associated to each pull request for project management and compliance

Use "actions [command] --help" for more information about a command
`

	fmt.Fprintf(os.Stderr, "%s\n", text)
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func main() {
	flag.Usage = flagUsage

	CmdPullvet := "pullvet"

	if len(os.Args) == 1 {
		flag.Usage()
		return
	}

	switch os.Args[1] {
	case CmdPullvet:
		fs := flag.NewFlagSet(CmdPullvet, flag.ExitOnError)
		cmd := pullvet.NewCommand()
		cmd.AddFlags(fs)

		fs.Parse(os.Args[2:])

		if err := cmd.Run(actions.PullRequestEvent().PullRequest); err != nil {
			fatal("%v\n", err)
		}
	default:
		flag.Usage()
	}
}
