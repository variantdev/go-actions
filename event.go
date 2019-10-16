package actions

import (
	"fmt"
	"github.com/google/go-github/github"
	"io/ioutil"
	"os"
)

func EventPath() string {
	path := os.Getenv("GITHUB_EVENT_PATH")
	if path == "" {
		fmt.Fprintf(os.Stderr, "GITHUB_EVENT_PATH not set. Please run this command on GitHub Actions")
		os.Exit(1)
	}
	return path
}

func Event() []byte {
	payload, err := ioutil.ReadFile(EventPath())
	if err != nil {
		panic(err)
	}
	return payload
}

func PullRequestEvent() (*github.PullRequestEvent, error) {
	evt, err := github.ParseWebHook("pull_request", Event())
	if err != nil {
		return nil, err
	}
	return evt.(*github.PullRequestEvent), nil
}

func CheckRunEvent() (*github.CheckRunEvent, error) {
	evt, err := github.ParseWebHook("check_run", Event())
	if err != nil {
		return nil, err
	}
	return evt.(*github.CheckRunEvent), nil
}

func CheckSuiteEvent() (*github.CheckSuiteEvent, error) {
	evt, err := github.ParseWebHook("check_suite", Event())
	if err != nil {
		return nil, err
	}
	return evt.(*github.CheckSuiteEvent), nil
}
