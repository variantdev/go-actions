package checks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/go-github/v28/github"
	"github.com/variantdev/go-actions"
	"github.com/variantdev/go-actions/pkg/cmd"
	"golang.org/x/oauth2"
)

type Command struct {
	BaseURL, UploadURL string
	createRuns         cmd.StringSlice

	checkName string
	cmd       string
	args      []string
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
	fs.StringVar(&c.checkName, "run", "", "CheckRun's name to be updated after the command in run")
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
		return c.EnsureCheckRun(e)
	case *github.CheckSuiteEvent:
		return c.CreateCheckRunsForSuite(e.CheckSuite)
	case *github.CheckRunEvent:
		return c.ExecCheckRun(e)
	}
	return nil
}

type Run struct {
	owner, repo, name string
	suiteId           int64
	runId             int64
}

func (c *Command) CreateCheckRunsForSuite(e *github.CheckSuite) error {
	owner, repo := e.Repository.GetOwner().GetLogin(), e.Repository.GetName()
	for _, name := range c.createRuns {
		_, err := c.createCheckRun(e, Run{owner: owner, repo: repo, name: name})
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Command) createCheckRun(suite *github.CheckSuite, cr Run) (*github.CheckRun, error) {
	client, err := c.instTokenClient()
	if err != nil {
		return nil, err
	}

	created, _, err := client.Checks.CreateCheckRun(
		context.Background(),
		cr.owner, cr.repo,
		github.CreateCheckRunOptions{
			Name:       cr.name,
			HeadBranch: suite.GetHeadBranch(),
			HeadSHA:    suite.GetHeadSHA(),
			StartedAt:  &github.Timestamp{Time: time.Now()},
			Status:     github.String("queued"),
		})

	return created, err
}

func (c *Command) EnsureCheckRun(pre *github.PullRequestEvent) error {
	client, err := c.instTokenClient()
	if err != nil {
		return err
	}

	suite, err := c.EnsureCheckSuite(pre)
	if err != nil {
		return err
	}

	cr := Run{
		name:    c.checkName,
		owner:   suite.Repository.Owner.GetLogin(),
		repo:    suite.Repository.GetName(),
		suiteId: suite.GetID(),
	}

	checkRunsList, _, err := client.Checks.ListCheckRunsCheckSuite(context.Background(), cr.owner, cr.repo, cr.suiteId, &github.ListCheckRunsOptions{
		CheckName: github.String(c.checkName),
		// TODO
		//ListOptions: github.ListOptions{},
	})

	var checkRun *github.CheckRun
	for _, existing := range checkRunsList.CheckRuns {
		if existing.GetName() == cr.name {
			checkRun = existing
		}
	}

	if checkRun == nil {
		log.Printf("Creating CheckRun %q", cr.name)
		created, err := c.createCheckRun(suite, cr)
		if err != nil {
			return err
		}
		checkRun = created
	}

	log.Printf("Running the commmand for CheckRun %q", cr.name)
	summary, text, runErr := c.runIt()

	log.Printf("Updating CheckRun")
	return c.UpdateCheckRun(checkRun, summary, text, runErr)
}

func (c *Command) EnsureCheckSuite(pre *github.PullRequestEvent) (*github.CheckSuite, error) {
	suite, err := c.getSuite(pre)
	if err != nil {
		return nil, err
	}

	if suite == nil {
		return c.CreateCheckSuite(pre)
	}

	return suite, nil
}

func (c *Command) ExecCheckRun(e *github.CheckRunEvent) error {
	stdout, fullout, err := c.runIt()

	return c.UpdateCheckRun(e.CheckRun, stdout, fullout, err)
}

func (c *Command) UpdateCheckRun(checkRun *github.CheckRun, summary, text string, runErr error) error {
	if checkRun.GetName() != c.checkName {
		return fmt.Errorf("unexpected run name: expected %q, got %q", c.checkName, checkRun.GetName())
	}

	client, err := c.instTokenClient()
	if err != nil {
		return err
	}

	var conclusion string
	if runErr != nil {
		conclusion = "failure"
	} else {
		conclusion = "success"
	}

	owner := checkRun.CheckSuite.Repository.Owner.GetLogin()
	repo := checkRun.CheckSuite.Repository.GetName()
	_, _, err = client.Checks.UpdateCheckRun(context.Background(), owner, repo, checkRun.GetID(), github.UpdateCheckRunOptions{
		//Name:        "",
		//HeadBranch:  nil,
		//HeadSHA:     nil,
		//DetailsURL:  nil,
		//ExternalID:  nil,
		Status:      github.String("completed"),
		Conclusion:  github.String(conclusion),
		CompletedAt: &github.Timestamp{Time: time.Now()},
		// See https://developer.github.com/v3/checks/runs/#output-object-1
		Output: &github.CheckRunOutput{
			Title:   github.String(c.cmd),
			Summary: github.String(fmt.Sprintf("```\n%s\n```", summary)),
			Text:    github.String(fmt.Sprintf("```\n%s\n```", text)),
		},
		//Actions:     nil,
	})

	return err
}

func (c *Command) runIt() (string, string, error) {
	return run(c.cmd, c.args)
}

func run(cmd string, args []string) (string, string, error) {
	c := exec.Command(cmd, args...)
	//c.Stdin = os.Stdin
	//var out bytes.Buffer
	//cmd.Stdout = &out
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err := c.Run()
	return "mysummary", "mytext", err
}

func (c *Command) logResponseAndError(suites *github.ListCheckSuiteResults, res *github.Response, err error) error {
	if err != nil {
		log.Printf("Error listing suites: %v", err)
	} else {
		buf := bytes.Buffer{}
		enc := json.NewEncoder(&buf)
		enc.SetIndent("", "  ")
		jsonErr := enc.Encode(suites)
		if jsonErr != nil {
			return jsonErr
		}
		suitesJson := buf.String()
		log.Printf("Listing suites: %s", suitesJson)
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
	return nil
}

func (c *Command) getSuite(pre *github.PullRequestEvent) (*github.CheckSuite, error) {
	client, err := c.instTokenClient()
	if err != nil {
		return nil, fmt.Errorf("Failed to create a new installation token client: %s", err)
	}

	owner, repo := pre.GetRepo().GetOwner().GetLogin(), pre.GetRepo().GetName()

	sha := pre.PullRequest.Head.GetSHA()

	log.Printf("Listing all suites...")

	suites, res, err := client.Checks.ListCheckSuitesForRef(context.Background(), owner, repo, sha, &github.ListCheckSuiteOptions{
	})

	c.logResponseAndError(suites, res, err)

	log.Printf("Listing relevant suites for check name %q...", c.checkName)

	suites, res, err = client.Checks.ListCheckSuitesForRef(context.Background(), owner, repo, sha, &github.ListCheckSuiteOptions{
		CheckName: github.String(c.checkName),
	})

	if err := c.logResponseAndError(suites, res, err); err != nil {
		return nil, err
	}

	if suites.GetTotal() == 1 {
		return suites.CheckSuites[0], nil
	} else if suites.GetTotal() > 1 {
		return nil, fmt.Errorf("too many suites exist(%d). maybe a bug?", suites.GetTotal())
	}

	return nil, nil
}

// CreateCheckSuite creates a new check suite and rerequests it based on a pull request.
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
func (c *Command) CreateCheckSuite(pre *github.PullRequestEvent) (*github.CheckSuite, error) {
	repoFullname := pre.Repo.GetFullName()
	ref := fmt.Sprintf("refs/pull/%d/head", pre.PullRequest.GetNumber())
	sha := pre.PullRequest.Head.GetSHA()

	client, err := c.instTokenClient()
	if err != nil {
		return nil, fmt.Errorf("Failed to create a new installation token client: %s", err)
	}

	ownerRepo := strings.Split(repoFullname, "/")
	owner, repo := ownerRepo[0], ownerRepo[1]

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
			return nil, errors.New("could not create check suite")
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
		return nil, err
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

	if err == nil {
		buf := bytes.Buffer{}
		enc := json.NewEncoder(&buf)
		enc.SetIndent("", "  ")
		jsonErr := enc.Encode(cs)
		if jsonErr != nil {
			return nil, jsonErr
		}
		csJson := buf.String()
		log.Printf("Created suite: %s", csJson)
	}

	//log.Printf("Created check suite for %s with ID %d. Triggering :rerequested", ref, cs.GetID())
	//// It appears that merely creating the check suite does not trigger a check_suite:request.
	//// So we manually trigger a rerequest.
	//res, err = client.Checks.ReRequestCheckSuite(context.Background(), owner, repo, cs.GetID())
	//
	//if res != nil {
	//	read, err := ioutil.ReadAll(res.Body)
	//	if err != nil {
	//		log.Printf("Failed reading rerequest response: %v", err)
	//	}
	//	if read != nil {
	//		log.Printf("Rerequest response: %s", string(read))
	//	}
	//}

	return cs, err
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
