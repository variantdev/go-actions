# pullvet

`pullvet` checks labels and milestones associated to each pull request for project management and compliance.

A pullvet rule looks like `accept only PR that does have at least one of these labels and one or more release notes in the description`.

## Rationale

I'm too lazy to subscribe any commercial SaaS or host my own GitHub app that acts like a bot.

Just give me a simple Golang program that reads GitHub Actions v2 event json file.

## Usage

```
$ bin/actions pullvet -help
Usage of pullvet:
  -any-milestone
    	If set, pullvet fails whenever the pull request misses a milestone
  -label value
    	Required label. When provided multiple times, pullvet succeeds if one or more of required labels exist
  -label-match value
    	Regexp pattern to match label name against. If set, pullvet tries to find the label matches any of patterns and fail if none matched.
  -milestone string
    	If set, pullvet fails whenever the pull request misses a milestone
  -milestone-match value
    	Regexp pattern to match milestone title against. If set, pullvet tries to find the milestone matches any of patterns and fail if none matched.
  -note
    	Require a note with the specified title. pullvet fails whenever the pr misses the note in the pr description. A note can be written in Markdown as: **<title>**:
    	`
    	<body>
    	```
  -note-regex string
    	Regexp pattern of each note(including the title and the body) (default "[\\*]*([^\\*\r\n:]+)[\\*]*:\\s```\n([^`]+)\n```")
  -require-all
    	If set, pullvet fails whenever the pull request was unable to fullfill any of the requirements
  -require-any
    	If set, pullvet fails whenever the pull request was unable to fullfill all the requirements (default true)
```

## Running locally

Grab the example webhook payload from:

https://developer.github.com/v3/activity/events/types/#pullrequestevent

Save it as `pull_request_event.json`:

```
$ pbpaste > pull_request_event.json
```

Run it like:

```
$ make build/pullvet

$ GITHUB_EVENT_PATH=$(pwd)/pull_request_event.json bin/pullvet -label foo -label bar
2 check(s) failed:
* missing label: foo
* missing label: bar
```
