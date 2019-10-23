package say

import (
	"context"
	"flag"
	"os"

	"github.com/google/go-github/v28/github"
	"github.com/variantdev/go-actions"
)

type Command struct {
	BaseURL, UploadURL string
	Body string
}

type Target struct {
	Owner, Repo string
	IssueNumber int
}

func New() *Command {
	return &Command{
		BaseURL:   "",
		UploadURL: "",
	}
}

func (c *Command) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.BaseURL, "github-base-url", "", "")
	fs.StringVar(&c.UploadURL, "github-upload-url", "", "")
	fs.StringVar(&c.Body, "body", "", " The contents of the comment.")
}

func (c *Command) Run() error {
	evt, err := actions.ParseEvent()
	if err != nil {
		return err
	}
	return c.HandleEvent(evt)
}

func (c *Command) HandleEvent(payload interface{}) error {
	switch e := payload.(type) {
	case *github.IssuesEvent:
		target := &Target{
			Owner:       e.Repo.Owner.GetLogin(),
			Repo:        e.Repo.GetName(),
			IssueNumber: e.Issue.GetNumber(),
		}
		return c.AddComment(target)
	case *github.PullRequestEvent:
		owner := e.Repo.Owner.GetLogin()
		repo := e.Repo.GetName()
		target := &Target{
			Owner:       owner,
			Repo:        repo,
			IssueNumber: e.PullRequest.GetNumber(),
		}
		return c.AddComment(target)
	}
	return nil
}

func (c *Command) AddComment(target *Target) error {
	client, err := c.createClient()
	if err != nil {
		return err
	}

	_, _, err = client.Issues.CreateComment(context.Background(), target.Owner, target.Repo, target.IssueNumber, &github.IssueComment{
		Body: github.String(c.Body),
	})

	return err
}

func (c *Command) createClient() (*github.Client, error) {
	return actions.CreateClient(os.Getenv("GITHUB_TOKEN"), c.BaseURL, c.UploadURL)
}
