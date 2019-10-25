package rebase

import (
	"context"
	"flag"
	"os"

	"github.com/google/go-github/v28/github"
	"github.com/variantdev/go-actions"
)

type Action struct {
	BaseURL, UploadURL string
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
	return c.ForcePushRebased(target)
}

// The rebase-via-github-api algorithm comes from https://github.com/tibdex/github-cherry-pick
// Thanks a lot for the original author!
func (c *Action) ForcePushRebased(pre *Target) error {
	client, err := c.getClient()
	if err != nil {
		return err
	}

	owner := pre.Owner
	repo := pre.Repo
	pr := pre.PullRequest

	pullHead := pr.Head.GetRef()

	comparison, _, err := client.Repositories.CompareCommits(context.Background(), owner, repo, pr.Base.GetSHA(), pr.Head.GetSHA())
	if err != nil {
		return err
	}

	commits := comparison.Commits

	tempBranch := "refs/heads/temp"

	latestBaseRef, _, err := client.Git.GetRef(context.Background(), owner, repo, pr.Base.GetRef())
	if err != nil {
		return err
	}

	newHeadStart := latestBaseRef.Object.GetSHA()

	parentOfNextChange := pr.Head.GetSHA()

	_, _, err = client.Git.CreateRef(context.Background(), owner, repo, &github.Reference{Ref: github.String(tempBranch), Object: &github.GitObject{SHA: github.String(newHeadStart)}})
	if err != nil {
		return err
	}

	newHeadCommit, _, err := client.Git.GetCommit(context.Background(), owner, repo, newHeadStart)
	if err != nil {
		return err
	}

	for i := 0; i < len(commits); i++ {
		emptyCommit := &github.Commit{
			Author:    &github.CommitAuthor{Name: github.String("go-actions")},
			Committer: &github.CommitAuthor{Name: github.String("go-actions")},
			Message:   github.String("rebase wip"),
			Tree:      newHeadCommit.GetTree(),
			Parents: []github.Commit{
				{
					SHA: github.String(parentOfNextChange),
				},
			},
		}
		emptyCommitCreated, _, cpErr := client.Git.CreateCommit(context.Background(), owner, repo, emptyCommit)
		if cpErr != nil {
			return cpErr
		}

		ref := &github.Reference{Ref: github.String(tempBranch), Object: &github.GitObject{SHA: emptyCommitCreated.SHA}}
		_, _, refErr := client.Git.UpdateRef(context.Background(), owner, repo, ref, false)
		if refErr != nil {
			return refErr
		}

		pickedCommit := commits[i]

		mergeCommit, _, mergeErr := client.Repositories.Merge(context.Background(), owner, repo, &github.RepositoryMergeRequest{Base: github.String(tempBranch), Head: pickedCommit.SHA})
		if mergeErr != nil {
			return mergeErr
		}

		newEmptyCommit := &github.Commit{
			Author:    pickedCommit.Commit.Author,
			Committer: pickedCommit.Commit.Committer,
			Parents:   []github.Commit{{SHA: newHeadCommit.SHA}},
			Tree:      mergeCommit.Commit.Tree,
		}
		newHeadCommit, _, err = client.Git.CreateCommit(context.Background(), owner, repo, newEmptyCommit)
		if err != nil {
			return err
		}

		_, _, err = client.Git.UpdateRef(context.Background(), owner, repo, &github.Reference{Ref: github.String(tempBranch), Object: &github.GitObject{SHA: newHeadCommit.SHA}}, true)
		if err != nil {
			return err
		}

		parentOfNextChange = commits[i].GetSHA()
	}

	refObj := &github.Reference{
		Ref: github.String(pullHead),
		Object: &github.GitObject{
			SHA: newHeadCommit.SHA,
		},
	}

	_, _, refErr := client.Git.UpdateRef(context.Background(), owner, repo, refObj, true)

	return refErr
}

func (c *Action) getClient() (*github.Client, error) {
	return actions.CreateClient(os.Getenv("GITHUB_TOKEN"), c.BaseURL, c.UploadURL)
}
