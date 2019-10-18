package actions

import (
	"context"
	"fmt"
	"github.com/google/go-github/v28/github"
	"io/ioutil"
	"os"
)

func EventPath() string {
	// See https://help.github.com/en/articles/virtual-environments-for-github-actions#default-environment-variables
	path := os.Getenv("GITHUB_EVENT_PATH")
	if path == "" {
		fmt.Fprintf(os.Stderr, "GITHUB_EVENT_PATH not set. Please run this command on GitHub Actions")
		os.Exit(1)
	}
	return path
}

func EventName() string {
	// See https://help.github.com/en/articles/virtual-environments-for-github-actions#default-environment-variables
	name := os.Getenv("GITHUB_EVENT_NAME")
	if name == "" {
		fmt.Fprintf(os.Stderr, "GITHUB_EVENT_NAME not set. Please run this command on GitHub Actions")
		os.Exit(1)
	}
	return name
}

func Event() []byte {
	payload, err := ioutil.ReadFile(EventPath())
	if err != nil {
		panic(err)
	}
	return payload
}

func ParseEvent() (interface{}, error) {
	return github.ParseWebHook(EventName(), Event())
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

func IssueEvent() (*github.IssuesEvent, error) {
	evt, err := github.ParseWebHook("issues", Event())
	if err != nil {
		return nil, err
	}
	return evt.(*github.IssuesEvent), nil
}

func GetPullRequest(issue *github.IssuesEvent) (*github.PullRequest, error) {
	client, err := CreateInstallationTokenClient(os.Getenv("GITHUB_TOKEN"), "", "")
	if err != nil {
		return nil, err
	}

	if issue.Issue.GetPullRequestLinks().GetURL() == "" {
		return nil, fmt.Errorf("issue %d is not a pull request", issue.Issue.GetNumber())
	}

	// This can be a pull_request milestoned/demilestoned events emitted as issue event
	owner := issue.Repo.Owner.GetLogin()
	repo := issue.Repo.GetName()
	pull, _, err := client.PullRequests.Get(context.Background(), owner, repo, issue.Issue.GetNumber())
	if err != nil {
		return nil, err
	}
	return pull, nil
}

func PullRequest() (*github.PullRequest, string, string, error) {
	var pr *github.PullRequest
	var owner, repo string
	switch EventName() {
	case "issues":
		issue, err := IssueEvent()
		if err != nil {
			return nil, "", "", err
		}

		pull, err := GetPullRequest(issue)
		if err != nil {
			return nil, "", "", err
		}

		pr = pull
	case "pull_request":
		pull, err := PullRequestEvent()
		if err != nil {
			return nil, "", "", err
		}
		pr = pull.PullRequest
		owner = pull.Repo.Owner.GetLogin()
		repo = pull.Repo.GetName()
	case "check_run":
		checkRun, err := CheckRunEvent()
		if err != nil {
			return nil, "", "", err
		} else {
			pr = checkRun.CheckRun.PullRequests[0]
		}
		owner = checkRun.Repo.Owner.GetLogin()
		repo = checkRun.Repo.GetName()
	case "check_suite":
		checkSuite, err := CheckSuiteEvent()
		if err != nil {
			return nil, "", "", err
		}
		pr = checkSuite.CheckSuite.PullRequests[0]
		owner = checkSuite.Repo.Owner.GetLogin()
		repo = checkSuite.Repo.GetName()
	}
	return pr, owner, repo, nil
}
