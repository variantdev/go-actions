# go-actions

[![dockeri.co](https://dockeri.co/image/variantdev/actions)](https://hub.docker.com/r/variantdev/actions)

A collection of usable commands for GitHub v2 Actions, written in Go. 

Use for as-is, or reference and inspiration of your own command.

## Included commands

- [pullvet](https://github.com/variantdev/go-actions/tree/master/cmd/pullvet) checks labels and milestones associated to each pull request for project management and compliance.
   A pullvet rule looks like `accept only PR that does have at least one of these labels and one or more release notes in the description`.
- [exec](https://github.com/variantdev/go-actions/tree/master/cmd/exec) runs an arbitrary command and updates GitHub "Check Run" and/or "Status" accordingly

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
  exec      runs an arbitrary command and updates GitHub "Check Run" and/or "Status" accordingly

Use "actions [command] --help" for more information about a command
```

### Examples

#### Regexp-match pull request label(s)

Set pull request status named `label` to green only when it has a "size" label like "size/s":

```
actions exec -status-context label -- actions pullvet -label-match 'size/.+'
```

#### Regexp-match pull request milestone or alternative label 

Set pull request status named `milestone` to green only when it has a milestone titled like "test-v1", or a label "milestone/none" to express there's exactly no milestone associated:

```
actions exec -status-context milestone -- actions pullvet -require-any -milestone-match 'test-v.+' label milestone/none
```

### GitHub Actions

Provide `GITHUB_TOKEN` as you usually do on GitHub Actions:

```
name: pullvet
on:
  pull_request:
    types: [opened, reopened, edited, milestoned, demilestoned, labeled, unlabeled, synchronize ]
jobs:
  pullvet:
    runs-on: ubuntu-latest
    steps:
    - uses: docker://variantdev/actions:latest
      with:
        args: pullvet -require-any -label releasenote/none -note releasenote
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
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
