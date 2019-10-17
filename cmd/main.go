package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/variantdev/go-actions/pkg/checks"
	"github.com/variantdev/go-actions/pkg/pullvet"
)

func flagUsage() {
	text := `actions A collection of usable commands for GitHub v2 Actions

Usage:
  actions [command]
Available Commands:
  pullvet	checks labels and milestones associated to each pull request for project management and compliance
  checks	drives GitHub Checks by creating CheckSuite and CheckRun, running and updating CheckRun

Use "actions [command] --help" for more information about a command
`

	fmt.Fprintf(os.Stderr, "%s\n", text)
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

const (
	CmdPullvet = "pullvet"
	CmdChecks  = "checks"
)

func main() {
	flag.Usage = flagUsage

	if len(os.Args) == 1 {
		flag.Usage()
		return
	}

	switch os.Args[1] {
	case CmdPullvet:
		fs := flag.NewFlagSet(CmdPullvet, flag.ExitOnError)
		cmd := pullvet.New()
		cmd.AddFlags(fs)

		fs.Parse(os.Args[2:])

		if err := cmd.Run(); err != nil {
			fatal("%v\n", err)
		}
	case CmdChecks:
		fs := flag.NewFlagSet(CmdChecks, flag.ExitOnError)
		cmd := checks.New()
		cmd.AddFlags(fs)

		fs.Parse(fs.Args())

		if err := cmd.Run(os.Args); err != nil {
			fatal("%v\n", err)
		}
	default:
		flag.Usage()
	}
}
