package say

import (
	"context"
	"flag"
	"os"

	"github.com/google/go-github/v28/github"
	"github.com/variantdev/go-actions"
)

type Action struct {
	BaseURL, UploadURL string
	Body string
}

type Target struct {
	Owner, Repo string
	IssueNumber int
}

func New() *Action {
	return &Action{
		BaseURL:   "",
		UploadURL: "",
	}
}

func (c *Action) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.BaseURL, "github-base-url", "", "")
	fs.StringVar(&c.UploadURL, "github-upload-url", "", "")
	fs.StringVar(&c.Body, "body", "", " The contents of the comment.")
}

func (c *Action) Run() error {
	num, owner, repo, err := actions.IssueNumberOwnerRepo()
	if err != nil {
		return err
	}
	target := &Target{
		Owner:       owner,
		Repo:        repo,
		IssueNumber: num,
	}
	return c.AddComment(target)
}


func (c *Action) AddComment(target *Target) error {
	client, err := c.createClient()
	if err != nil {
		return err
	}

	_, _, err = client.Issues.CreateComment(context.Background(), target.Owner, target.Repo, target.IssueNumber, &github.IssueComment{
		Body: github.String(c.Body),
	})

	return err
}

func (c *Action) createClient() (*github.Client, error) {
	return actions.CreateClient(os.Getenv("GITHUB_TOKEN"), c.BaseURL, c.UploadURL)
}
