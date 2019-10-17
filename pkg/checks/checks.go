package checks

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/google/go-github/v28/github"
	"github.com/variantdev/go-actions"
	"github.com/variantdev/go-actions/pkg/cmd"
	"golang.org/x/oauth2"
)

type Command struct {
	BaseURL, UploadURL string
	createRuns         cmd.StringSlice

	runName string
	cmd     string
	args    []string
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
	fs.Var(&c.createRuns, "create-run", "Name of CheckRun to be created on CheckSuite `(re)requested` event. Specify multiple times to create two or more runs")
	fs.StringVar(&c.runName, "run", "", "CheckRun's name to be updated after the command in run")
}

func (c *Command) Run(args []string) error {
	numArgs := len(args)
	if numArgs > 0 {
		c.cmd = args[0]
	}
	if numArgs > 1 {
		c.args = args[1:]
	}

	evt, err := actions.ParseEvent()
	if err != nil {
		return err
	}
	return c.HandleEvent(evt)
}

func (c *Command) HandleEvent(payload interface{}) error {
	switch e := payload.(type) {
	case *github.PullRequestEvent:
		action := *e.Action
		if action == "opened" || action == "synchronize" || action == "reopened" {
			if err := c.RequestCheckSuite(e); err != nil {
				return err
			}
		}
	case *github.CheckSuiteEvent:
		owner, repo := e.GetRepo().GetOwner().GetLogin(), e.GetRepo().GetName()
		for _, name := range c.createRuns {
			_, err := c.createRun(Run{owner: owner, repo: repo, name: name})
			if err != nil {
				return err
			}
		}
	case *github.CheckRunEvent:
		return c.ExecCheckRun(e)
	}
	return nil
}

type Run struct {
	owner, repo, name string
}

func (c *Command) createRun(cr Run) (string, error) {
	client, err := c.instTokenClient()
	if err != nil {
		return "", err
	}

	_, res, err := client.Checks.CreateCheckRun(
		context.Background(),
		cr.owner, cr.repo,
		github.CreateCheckRunOptions{
			Name: cr.name,
			Status:      github.String("queued"),
		})

	body, _ := ioutil.ReadAll(res.Body)
	res.Body.Close()
	fmt.Printf("%+v", res)

	return string(body), err
}

func (c *Command) ExecCheckRun(e *github.CheckRunEvent) error {
	if e.CheckRun.GetName() != c.runName {
		return fmt.Errorf("unexpected run name: expected %q, got %q", c.runName, e.CheckRun.GetName())
	}

	client, err := c.instTokenClient()
	if err != nil {
		return err
	}

	var conclusion string
	stdout, fullout, err := c.runIt()
	if err != nil {
		conclusion = "failure"
	} else {
		conclusion = "success"
	}

	owner := e.GetRepo().GetOwner().GetLogin()
	repo := e.GetRepo().GetName()
	_, res, err := client.Checks.UpdateCheckRun(context.Background(), owner, repo, e.CheckRun.GetID(), github.UpdateCheckRunOptions{
		//Name:        "",
		//HeadBranch:  nil,
		//HeadSHA:     nil,
		//DetailsURL:  nil,
		//ExternalID:  nil,
		Status:      github.String("completed"),
		Conclusion:  github.String(conclusion),
		//CompletedAt: nil,
		// See https://developer.github.com/v3/checks/runs/#output-object-1
		Output:      &github.CheckRunOutput{
			Title:            github.String(c.cmd),
			Summary:          github.String(fmt.Sprintf("```\n%s\n```", stdout)),
			Text:             github.String(fmt.Sprintf("```\n%s\n```", fullout)),
		},
		//Actions:     nil,
	})

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	res.Body.Close()
	fmt.Printf("%s\n", string(body))

	return err
}

func (c *Command) runIt() (string, string, error) {
	return run(c.cmd, c.args)
}

func run(cmd string, args []string) (string, string, error) {
	c := exec.Command(cmd, args...)
	c.Stdin = os.Stdin
	//var out bytes.Buffer
	//cmd.Stdout = &out
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err := c.Run()
	return "", "", err
}

// RequestCheckSuite creates a new check suite and rerequests it based on a pull request.
//
// The Check Suite webhook events are normally only triggered on `push` events. This function acts as an
// adapter to take a PR and trigger a check suite.
//
// The GitHub API is still evolving, so the current way we do this is...
//
//	- generate auth tokens for the instance/app combo. This is required to perform the action as a
//		GitHub app
//	- try to create a check_suite
//		- if success, run a `rerequest` on this check suite because merely creating a check suite does
// 		  not actually trigger a check_suite:requested webhook event
//		- if failure, check to see if we already have a check suite object, and merely run the rerequest
//		  on that check suite.
func (c *Command) RequestCheckSuite(pre *github.PullRequestEvent) error {
	repoFullname := pre.Repo.GetFullName()
	ref := fmt.Sprintf("refs/pull/%d/head", pre.PullRequest.GetNumber())
	sha := pre.PullRequest.Head.GetSHA()

	client, err := c.instTokenClient()
	if err != nil {
		return fmt.Errorf("Failed to create a new installation token client: %s", err)
	}

	ownerRepo := strings.Split(repoFullname, "/")
	owner, repo := ownerRepo[0], ownerRepo[1]

	_, res, err := client.Checks.ListCheckSuitesForRef(context.Background(), owner, repo, sha, &github.ListCheckSuiteOptions{})
	if err != nil {
		log.Printf("Error listing suites: %v", err)
	}
	if res != nil {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Printf("Error reading suites response: %v", err)
		}
		res.Body.Close()
		if body != nil {
			log.Printf("Listing suites: %s", string(body))
		}
	}

	csOpts := github.CreateCheckSuiteOptions{
		HeadSHA:    sha,
		HeadBranch: &ref,
	}
	log.Printf("requesting check suite run for %s/%s, SHA: %s", owner, repo, csOpts.HeadSHA)

	cs, res, err := client.Checks.CreateCheckSuite(context.Background(), owner, repo, csOpts)
	if err != nil {
		log.Printf("Failed to create check suite: %s", err)

		// 422 means the suite already exists.
		if res.StatusCode != 422 {
			return errors.New("could not create check suite")
		}

		log.Println("rerunning the last suite")
		csl, _, err := client.Checks.ListCheckSuitesForRef(context.Background(), owner, repo, sha, &github.ListCheckSuiteOptions{})
		if err == nil && csl.GetTotal() > 0 {
			log.Printf("Loading check suite %d", csl.CheckSuites[0].GetID())
			_, err := client.Checks.ReRequestCheckSuite(context.Background(), owner, repo, csl.CheckSuites[0].GetID())
			if err != nil {
				log.Printf("error rerunning suite: %s", err)
			}
		} else {
			log.Printf("error fetching check suites: %s", err)
		}
		return nil
	}

	if res != nil {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Printf("error reading create check suite response: %v", err)
		}
		res.Body.Close()
		if body != nil {
			log.Printf("CreateCheckSuite: %s", string(body))
		}
	}

	log.Printf("Created check suite for %s with ID %d. Triggering :rerequested", ref, cs.GetID())
	// It appears that merely creating the check suite does not trigger a check_suite:request.
	// So we manually trigger a rerequest.
	_, err = client.Checks.ReRequestCheckSuite(context.Background(), owner, repo, cs.GetID())
	return err
}

func (c *Command) instTokenClient() (*github.Client, error) {
	return instTokenClient(os.Getenv("GITHUB_TOKEN"), c.BaseURL, c.UploadURL)
}

// instTokenClient uses an installation token to authenticate to the Github API.
func instTokenClient(instToken, baseURL, uploadURL string) (*github.Client, error) {
	// For installation tokens, Github uses a different token type ("token" instead of "bearer")
	tokenType := "token"
	if os.Getenv("GITHUB_TOKEN_TYPE") != "" {
		tokenType = os.Getenv("GITHUB_TOKEN_TYPE")
	}
	t := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: instToken, TokenType: tokenType})
	c := context.Background()
	tc := oauth2.NewClient(c, t)
	if baseURL != "" {
		return github.NewEnterpriseClient(baseURL, uploadURL, tc)
	}
	return github.NewClient(tc), nil
}
