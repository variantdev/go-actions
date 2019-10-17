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

> This is just here to show the ideal developer experience. The reality is that you can't test `checks` locally.
>
> That's due to limitations in GitHub.
> As Checks API requires a GitHub App installation token(not bearer token you're probably familiar with) to access
> and there's currently no way to obtain an installation token locally,
> you can't test this locally.

Capture actual webhook payloads for `pull_request`, `check_suite`, `check_run` events by running `cat /github/workflow/event.json` on GitHub Actions.

Install this workflow onto your test repository:

https://github.com/variantdev/variant-github-actions-demo/blob/master/.github/workflows/dump-event.yml

Open a PR against the repo:

```
git checkout -b test
git commit --allow-empty
git push
hub pull-request
```

Browse to the Actions tab in your repo:

```
open https://github.com/USER/REPO/actions
```

Nagivate to the workflow run that corresponds to the pull request and browse `View raw logs` in the popup menu.

Copy the dump of the event.json and save it as a json file:

```
$ pbpaste | awk '{$1=""; print $0;}' > pull_request_event.json
```

Then build and run `checks` against json files containing captured events:

```
$ make build/checks

# Typically this is run on Actions with `on: pull_requqest`
$ GITHUB_TOKEN_TYPE=bearer GITHUB_EVENT_PATH=$(pwd)/pull_request_event.json GITHUB_EVENT_NAME=pull_request bin/checks

# `on: check_suite`
$ GITHUB_TOKEN_TYPE=bearer GITHUB_EVENT_PATH=$(pwd)/check_suite_event.json bin/checks -create-run foo -create-run bar

# `on: check_run`
$ GITHUB_TOKEN_TYPE=bearer GITHUB_EVENT_PATH=$(pwd)/check_run_event.json bin/checks -run foo -- actions pullvet ...
```
