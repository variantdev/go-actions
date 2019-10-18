# checks

`exec` runs an arbitrary command and updates GitHub "Check Run" and/or "Status" accordingly. 

## Rationale

GitHub Actions v2 has a known issue that prevents the "Required Status Checks" branch protection from working due to
duplicate pull request statuses created by multiple workflow runs against a specific Git commit.

Give me a simple CLI program that updates Check Run and/or pull request Status without duplication so that the branch protection works.

## Usage

```
$ bin/actions exec -help
Usage of exec:
  -check-run-name string
    	CheckRun's name to be updated after the command in run
  -github-base-url string

  -github-upload-url string

  -status-context exec
    	Commit status' context. If not empty, exec creates a status with this context
  -status-description exec
    	Commit status' description. exec creates a status with this description
```

## Running locally

Capture actual webhook payloads for `pull_request`by running `cat $GITHUB_EVENT_PATH` on GitHub Actions.

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

Then build and run `exec` against json files containing captured events:

```
$ make build/exec

# Typically this is run on Actions with `on: pull_requqest`
$ GITHUB_TOKEN_TYPE=bearer GITHUB_EVENT_PATH=$(pwd)/pull_request_event.json GITHUB_EVENT_NAME=pull_request bin/exec -status-context ci/test
```
