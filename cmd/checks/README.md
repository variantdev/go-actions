# checks

`checks` drives GitHub Checks by creating CheckSuite and CheckRun, running and updating CheckRun

## Rationale

I'm too lazy to subscribe any commercial SaaS or host my own GitHub app just to run [GitHub Checks](https://developer.github.com/v3/checks/).

Just give me a simple Golang program that reads GitHub Actions v2 event json file and do anything other than the business logic(=some shell snippet for tasks like building, testing, etc.)

## Usage

```
$ bin/checks -h
Usage of bin/checks:
  -create-run (re)requested
    	Name of CheckRun to be created on CheckSuite (re)requested event. Specify multiple times to create two or more runs
  -github-base-url string

  -github-upload-url string

  -run string
    	CheckRun's name to be updated after the command in run
```

## Running locally

Capture actual webhook payloads for `pull_request`, `check_suite`, `check_run` events by running `cat /github/workflow/event.json` on GitHub Actions.

Then build and run `checks` against json files containing captured events:

```
$ make build/checks

# Typically this is run on Actions with `on: pull_requqest`
$ GITHUB_EVENT_PATH=$(pwd)/pull_request_event.json bin/checks

# `on: check_suite`
$ GITHUB_EVENT_PATH=$(pwd)/check_suite_event.json bin/checks -create-run foo -create-run bar

# `on: check_run`
$ GITHUB_EVENT_PATH=$(pwd)/check_run_event.json bin/checks -run foo -- actions pullvet ...
```
