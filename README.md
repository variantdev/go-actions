# go-actions

A collection of usable commands for GitHub v2 Actions, written in Go. 

Use for as-is, or reference and inspiration of your own command.

## Included commands

- [pullvet](https://github.com/variantdev/go-actions/tree/master/cmd/pullvet) checks labels and milestones associated to each pull request for project management and compliance.
   A pullvet rule looks like `accept only PR that does have at least one of these labels and one or more release notes in the description`.

## Usage

TL;DR; Grab `actions` or sub-components like `pullvet` from releases and run it on GitHub Actions v2.

Although each command is buildable and available as a standalone application, there's also a "hyper" app called 
`actions`, which is also a standalone binary that is composed of multiple apps listed in the `Included commands` section:

```
$ actions
actions is a collection of usable commands for GitHub v2 Actions

Usage:
  actions [command]
Available Commands:
  pullvet	checks labels and milestones associated to each pull request for project management and compliance

Use "actions [command] --help" for more information about a command
```

## Developing

Run `make build` to build `bin/actions`:

```
$ make build
$ bin/actions
actions A collection of usable commands for GitHub v2 Actions

Usage:
  actions [command]
Available Commands:
  pullvet	checks labels and milestones associated to each pull request for project management and compliance

Use "actions [command] --help" for more information about a command
```
