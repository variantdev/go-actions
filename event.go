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

func PullRequestEvent() *github.PullRequestEvent {
	evt, err := github.ParseWebHook("pull_request", Event())
	if err != nil {
		panic(err)
	}
	return evt.(*github.PullRequestEvent)
}
