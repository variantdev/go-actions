package merge

import (
	"context"
	"flag"
	"log"
	"os"
	"strings"

	"github.com/google/go-github/v28/github"
	"github.com/variantdev/go-actions"
)

type Action struct {
	BaseURL, UploadURL string

	Force  bool
	Method string
}

type Target struct {
	Owner, Repo string
	PullRequest *github.PullRequest
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
	fs.BoolVar(&c.Force, "force", false, "Merges the pull request even if required checks are NOT passing")
	fs.StringVar(&c.Method, "method", "merge", ` The merge method to use. Possible values include: "merge", "squash", and "rebase" with the default being merge`)
}

func (c *Action) Run() error {
	pr, owner, repo, err := actions.PullRequest()
	if err != nil {
		return err
	}
	target := &Target{
		Owner:       owner,
		Repo:        repo,
		PullRequest: pr,
	}
	return c.MergeIfNecessary(target)
}

func (c *Action) MergeIfNecessary(pre *Target) error {
	client, err := c.getClient()
	if err != nil {
		return err
	}

	owner := pre.Owner
	repo := pre.Repo
	num := pre.PullRequest.GetNumber()

	if !c.Force {
		ref := pre.PullRequest.Head.GetRef()

		headBranch := strings.TrimPrefix(ref, "refs/heads")

		contexts, _, err := client.Repositories.ListRequiredStatusChecksContexts(context.Background(), owner, repo, headBranch)
		if err != nil {
			return err
		}

		// TODO Pagination
		statuses, _, err := client.Repositories.ListStatuses(context.Background(), owner, repo, ref, &github.ListOptions{})
		if err != nil {
			return err
		}

		reqCheckContexts := map[string]struct{}{}
		for _, c := range contexts {
			reqCheckContexts[c] = struct{}{}
			log.Printf("Seen required status context %q", c)
		}

		reqChecksPassing := true
		for _, st := range statuses {
			log.Printf("Seen status context %q", st.GetContext())
			_, ok := reqCheckContexts[st.GetContext()]
			reqChecksPassing = reqChecksPassing && ok
		}

		if !reqChecksPassing {
			return nil
		}
	}

	log.Printf("Merging the pull request with method %q", c.Method)

	_, _, mergeErr := client.PullRequests.Merge(context.Background(), owner, repo, num, "", &github.PullRequestOptions{
		MergeMethod: c.Method,
	})

	return mergeErr
}

func (c *Action) getClient() (*github.Client, error) {
	return actions.CreateClient(os.Getenv("GITHUB_TOKEN"), c.BaseURL, c.UploadURL)
}
