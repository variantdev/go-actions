# go-actions

A collection of usable commands for GitHub v2 Actions, written in Go. 

Use for as-is, or reference and inspiration of your own command.

## Included commands:

- [pullvet](https://github.com/variantdev/go-actions/tree/master/cmd/pullvet) checks labels and milestones associated to each pull request for project management and compliance.
   A pullvet rule looks like `accept only PR that does have at least one of these labels and one or more release notes in the description`.
