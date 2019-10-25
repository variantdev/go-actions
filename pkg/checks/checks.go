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
	"strings"
	"time"

	"github.com/google/go-github/v28/github"
	"github.com/variantdev/go-actions"
)

type Action struct {
	BaseURL, UploadURL string
	createRuns         actions.StringSlice

	checkRunName string

	StatusContext     string
	StatusDescription string

	Cmd  string
	Args []string
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
	fs.Var(&c.createRuns, "create-run", "Name of CheckRun to be created on CheckSuite `(re)requested` event. Specify multiple times to create two or more runs")
	fs.StringVar(&c.checkRunName, "check-run-name", "", "CheckRun's name to be updated after the command in run")
	fs.StringVar(&c.StatusContext, "status-context", "", "Commit status' context. If not empty, `exec` creates a status with this context")
	fs.StringVar(&c.StatusDescription, "status-description", "", "Commit status' description. `exec` creates a status with this description")
}

func (c *Action) Run(args []string) error {
	numArgs := len(args)
	if numArgs > 0 {
		c.Cmd = args[0]
	}
	if numArgs > 1 {
		c.Args = args[1:]
	}

	evt, err := actions.ParseEvent()
	if err != nil {
		return err
	}
	return c.HandleEvent(evt)
}

func (c *Action) HandleEvent(payload interface{}) error {
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

func (c *Action) CreateCheckRunsForSuite(e *github.CheckSuite) error {
	owner, repo := e.Repository.GetOwner().GetLogin(), e.Repository.GetName()
	for _, name := range c.createRuns {
		_, err := c.createCheckRun(e, Run{owner: owner, repo: repo, name: name})
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Action) createCheckRun(suite *github.CheckSuite, cr Run) (*github.CheckRun, error) {
	client, err := c.instTokenClient()
	if err != nil {
		return nil, err
	}

	log.Printf("Running CreateCheckRun experiment")

	_, _, err = c.CreateCheckRun(
		client,
		context.Background(),
		cr.owner, cr.repo,
		CreateCheckRunOptions{
			Name:         cr.name,
			HeadBranch:   suite.GetHeadBranch(),
			HeadSHA:      suite.GetHeadSHA(),
			StartedAt:    &github.Timestamp{Time: time.Now()},
			Status:       github.String("queued"),
			CheckSuiteID: suite.ID,
		})

	if err != nil {
		log.Printf("Failed experimentation on CreateCheckRun: %v", err)
	}

	log.Printf("Creating a check run")

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

type CreateCheckRunOptions struct {
	Name       string  `json:"name"`                  // The name of the check (e.g., "code-coverage"). (Required.)
	HeadBranch string  `json:"head_branch"`           // The name of the branch to perform a check against. (Required.)
	HeadSHA    string  `json:"head_sha"`              // The SHA of the commit. (Required.)
	DetailsURL *string `json:"details_url,omitempty"` // The URL of the integrator's site that has the full details of the check. (Optional.)
	ExternalID *string `json:"external_id,omitempty"` // A reference for the run on the integrator's system. (Optional.)
	Status     *string `json:"status,omitempty"`      // The current status. Can be one of "queued", "in_progress", or "completed". Default: "queued". (Optional.)
	Conclusion *string `json:"conclusion,omitempty"`  // Can be one of "success", "failure", "neutral", "cancelled", "timed_out", or "action_required". (Optional. Required if you provide a status of "completed".)
	// Does this really work?
	CheckSuiteID *int64                   `json:"check_suite_id,omitempty`
	StartedAt    *github.Timestamp        `json:"started_at,omitempty"`   // The time that the check run began. (Optional.)
	CompletedAt  *github.Timestamp        `json:"completed_at,omitempty"` // The time the check completed. (Optional. Required if you provide conclusion.)
	Output       *github.CheckRunOutput   `json:"output,omitempty"`       // Provide descriptive details about the run. (Optional)
	Actions      []*github.CheckRunAction `json:"actions,omitempty"`      // Possible further actions the integrator can perform, which a user may trigger. (Optional.)
}

func (c *Action) CreateCheckRun(client *github.Client, ctx context.Context, owner, repo string, opt CreateCheckRunOptions) (*github.CheckRun, *github.Response, error) {
	u := fmt.Sprintf("repos/%v/%v/check-suites/%v/check-runs", owner, repo, *opt.CheckSuiteID)
	req, err := client.NewRequest("POST", u, opt)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("Accept", "application/vnd.github.antiope-preview+json")

	checkRun := new(github.CheckRun)
	resp, err := client.Do(ctx, req, checkRun)
	if err != nil {
		return nil, resp, err
	}

	return checkRun, resp, nil
}

func (c *Action) EnsureCheckRun(pre *github.PullRequestEvent) error {
	client, err := c.instTokenClient()
	if err != nil {
		return err
	}

	log.Printf("Running command: %q", c.Cmd)

	summary, text, runErr := c.runIt()

	owner := pre.Repo.Owner.GetLogin()
	repo := pre.Repo.GetName()

	if c.checkRunName != "" {
		suite, err := c.EnsureCheckSuite(pre)
		if err != nil {
			return err
		}

		cr := Run{
			name:    c.checkRunName,
			owner:   owner,
			repo:    repo,
			suiteId: suite.GetID(),
		}

		checkRunsList, _, err := client.Checks.ListCheckRunsCheckSuite(context.Background(), cr.owner, cr.repo, cr.suiteId, &github.ListCheckRunsOptions{
			CheckName: github.String(c.checkRunName),
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

		c.logCheckRun(checkRun)

		log.Printf("Updating CheckRun")
		if err := c.UpdateCheckRun(owner, repo, checkRun, summary, text, runErr); err != nil {
			return err
		}
	}

	if c.StatusContext != "" {
		sha := pre.PullRequest.Head.GetSHA()
		var state string
		if runErr != nil {
			state = "failure"
		} else {
			state = "success"
		}

		// Otherwise you get errors like:
		//  2019/10/17 18:25:08 Failed creating status: POST https://api.github.com/repos/variantdev/go-actions/statuses/ceb4320db3c54081d55daa6d7a50ed8dc7fafc86: 422 Validation Failed [{Resource:Status Field:description Code:custom Message:description is too long (maximum is 140 characters)}]
		desc := text[0:140]

		status := &github.RepoStatus{
			State:       github.String(state),
			Context:     github.String(c.StatusContext),
			Description: github.String(desc),
		}
		repoStatus, _, err := client.Repositories.CreateStatus(context.Background(), owner, repo, sha, status)
		if err != nil {
			log.Printf("Failed creating status: %v", err)
		} else {
			buf := bytes.Buffer{}
			enc := json.NewEncoder(&buf)
			enc.SetIndent("", "  ")
			if err := enc.Encode(repoStatus); err != nil {
				return err
			}
			log.Printf("Created repo status:\n%s", buf.String())
		}
	}

	return runErr
}

func (c *Action) logCheckRun(checkRun *github.CheckRun) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(checkRun); err != nil {
		panic(err)
	}
	log.Printf("CheckRun:\n%s", buf.String())
}

func (c *Action) EnsureCheckSuite(pre *github.PullRequestEvent) (*github.CheckSuite, error) {
	suite, err := c.getSuite(pre)
	if err != nil {
		return nil, err
	}

	if suite == nil {
		return c.CreateCheckSuite(pre)
	}

	return suite, nil
}

func (c *Action) ExecCheckRun(e *github.CheckRunEvent) error {
	stdout, fullout, err := c.runIt()

	return c.UpdateCheckRun(e.GetRepo().Owner.GetLogin(), e.GetRepo().GetName(), e.CheckRun, stdout, fullout, err)
}

func (c *Action) UpdateCheckRun(owner, repo string, checkRun *github.CheckRun, summary, text string, runErr error) error {
	if checkRun.GetName() != c.checkRunName {
		return fmt.Errorf("unexpected run name: expected %q, got %q", c.checkRunName, checkRun.GetName())
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

	// This panics due to missing field(in perhaps some cases)
	//owner := checkRun.CheckSuite.Repository.Owner.GetLogin()
	//repo := checkRun.CheckSuite.Repository.GetName()
	_, _, err = client.Checks.UpdateCheckRun(context.Background(), owner, repo, checkRun.GetID(), github.UpdateCheckRunOptions{
		Name: checkRun.GetName(),
		//HeadBranch:  nil,
		//HeadSHA:     nil,
		//DetailsURL:  nil,
		//ExternalID:  nil,
		Status:      github.String("completed"),
		Conclusion:  github.String(conclusion),
		CompletedAt: &github.Timestamp{Time: time.Now()},
		// See https://developer.github.com/v3/checks/runs/#output-object-1
		Output: &github.CheckRunOutput{
			Title:   github.String(c.Cmd),
			Summary: github.String(fmt.Sprintf("```\n%s\n```", summary)),
			Text:    github.String(fmt.Sprintf("```\n%s\n```", text)),
		},
		//Actions:     nil,
	})

	return err
}

func (c *Action) runIt() (string, string, error) {
	return actions.RunCmd(c.Cmd, c.Args)
}

func (c *Action) logResponseAndError(suites *github.ListCheckSuiteResults, res *github.Response, err error) error {
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

func (c *Action) getSuite(pre *github.PullRequestEvent) (*github.CheckSuite, error) {
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

	log.Printf("Listing relevant suites for check name %q...", c.checkRunName)

	suites, res, err = client.Checks.ListCheckSuitesForRef(context.Background(), owner, repo, sha, &github.ListCheckSuiteOptions{
		CheckName: github.String(c.checkRunName),
	})

	if err := c.logResponseAndError(suites, res, err); err != nil {
		return nil, err
	}

	if suites.GetTotal() == 1 {
		return suites.CheckSuites[0], nil
	} else if suites.GetTotal() > 1 {
		log.Printf("too many suites exist(%d). maybe a bug? Returning the first item anyway", suites.GetTotal())
		return suites.CheckSuites[0], nil
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
func (c *Action) CreateCheckSuite(pre *github.PullRequestEvent) (*github.CheckSuite, error) {
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

	return cs, err
}

func (c *Action) instTokenClient() (*github.Client, error) {
	return actions.CreateClient(os.Getenv("GITHUB_TOKEN"), c.BaseURL, c.UploadURL)
}
