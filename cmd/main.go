package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/variantdev/go-actions/pkg/exec"
	"github.com/variantdev/go-actions/pkg/merge"
	"github.com/variantdev/go-actions/pkg/pullvet"
	"github.com/variantdev/go-actions/pkg/rebase"
	"github.com/variantdev/go-actions/pkg/say"
)

func flagUsage() {
	text := `actions A collection of usable commands for GitHub v2 Actions

Usage:
  actions [command]
Available Commands:
  pullvet	checks labels and milestones associated to each pull request for project management and compliance
  exec		runs an arbitrary command and updates GitHub "Check Run" and/or "Status" accordingly.
  merge		merges a PR when it is passing all the required status checks.
  say		adds a comment to an issue or a pull request that triggered the event.
  rebase	rebases the pull request onto the specified branch and force pushes it to the head branch.

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
	CmdExec    = "exec"
	CmdMerge   = "merge"
	CmdRebase  = "rebase"
	CmdSay     = "say"
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
	case CmdExec:
		fs := flag.NewFlagSet(CmdExec, flag.ExitOnError)
		cmd := exec.New()
		cmd.AddFlags(fs)

		fs.Parse(os.Args[2:])

		if err := cmd.Run(fs.Args()); err != nil {
			fatal("%v\n", err)
		}
	case CmdMerge:
		fs := flag.NewFlagSet(CmdMerge, flag.ExitOnError)
		cmd := merge.New()
		cmd.AddFlags(fs)

		fs.Parse(os.Args[2:])

		if err := cmd.Run(); err != nil {
			fatal("%v\n", err)
		}
	case CmdRebase:
		fs := flag.NewFlagSet(CmdRebase, flag.ExitOnError)
		cmd := rebase.New()
		cmd.AddFlags(fs)

		fs.Parse(os.Args[2:])

		if err := cmd.Run(); err != nil {
			fatal("%v\n", err)
		}
	case CmdSay:
		fs := flag.NewFlagSet(CmdSay, flag.ExitOnError)
		cmd := say.New()
		cmd.AddFlags(fs)

		fs.Parse(os.Args[2:])

		if err := cmd.Run(); err != nil {
			fatal("%v\n", err)
		}
	default:
		flag.Usage()
	}
}
