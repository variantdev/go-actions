package cli

import (
	"flag"
	"fmt"
)

type Command interface {
	Name() string
	Run([]string) error
}

func Setup(cmd Command, args []string, usage string, addFlags func(fs *flag.FlagSet)) error {
	fs := flag.NewFlagSet(cmd.Name(), flag.ExitOnError)

	addFlags(fs)

	printDefaultUsage := fs.Usage

	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "%s\n", usage)

		printDefaultUsage()
	}

	return fs.Parse(args)
}
