package exec

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/google/go-github/v28/github"
	"github.com/variantdev/go-actions"
)

type Action struct {
	BaseURL, UploadURL string

	checkRunName string

	StatusContext     string
	StatusDescription string
	StatusTargetURL   string

	Cmd  string
	Args []string
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
	fs.StringVar(&c.checkRunName, "check-run-name", "", "CheckRun's name to be updated after the command in run")
	fs.StringVar(&c.StatusContext, "status-context", "", "Commit status' context. If not empty, `exec` creates a status with this context")
	fs.StringVar(&c.StatusDescription, "status-description", "", "Commit status' description. `exec` creates a status with this description")
	fs.StringVar(&c.StatusTargetURL, "status-target-url", "", "Commit status' target_url. `exec` creates a status with this url as the link target")
}

func (c *Action) Run(args []string) error {
	numArgs := len(args)
	if numArgs > 0 {
		c.Cmd = args[0]
	}
	if numArgs > 1 {
		c.Args = args[1:]
	}

	pr, owner, repo, err := actions.PullRequest()
	if err != nil {
		return err
	}
	target := &Target{
		Owner:       owner,
		Repo:        repo,
		PullRequest: pr,
	}
	return c.EnsureCheckRun(target)
}

type Run struct {
	owner, repo, name string
	suiteId           int64
	runId             int64
}

func (c *Action) createCheckRun(suite *github.CheckSuite, cr Run) (*github.CheckRun, error) {
	client, err := c.instTokenClient()
	if err != nil {
		return nil, err
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

func (c *Action) CreateAndLogStatus(client *github.Client, owner, repo, sha string, status *github.RepoStatus) error {
	desc := status.GetDescription()
	if len(desc) > 140 {
		// Otherwise you get errors like:
		//  2019/10/17 18:25:08 Failed creating status: POST https://api.github.com/repos/variantdev/go-actions/statuses/ceb4320db3c54081d55daa6d7a50ed8dc7fafc86: 422 Validation Failed [{Resource:Status Field:description Code:custom Message:description is too long (maximum is 140 characters)}]
		desc = desc[0:140]

		status.Description = &desc
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

	return nil
}

func (c *Action) EnsureCheckRun(pre *Target) error {
	client, err := c.instTokenClient()
	if err != nil {
		return err
	}

	owner := pre.Owner
	repo := pre.Repo
	sha := pre.PullRequest.Head.GetSHA()

	if c.StatusContext != "" {
		status := &github.RepoStatus{
			State:       github.String("pending"),
			Context:     github.String(c.StatusContext),
			Description: github.String(c.StatusDescription),
		}

		if c.StatusTargetURL != "" {
			status.TargetURL = github.String(c.StatusTargetURL)
		}

		if err := c.CreateAndLogStatus(client, owner, repo, sha, status); err != nil {
			return err
		}
	}

	log.Printf("Running command: %q", c.Cmd)

	summary, text, runErr := c.runIt()

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
		var state string
		if runErr != nil {
			state = "failure"
		} else {
			state = "success"
		}

		var desc string

		if c.StatusDescription != "" {
			desc = c.StatusDescription + ". " + summary
		} else {
			desc = summary
		}

		status := &github.RepoStatus{
			State:       github.String(state),
			Context:     github.String(c.StatusContext),
			Description: github.String(desc),
		}

		if c.StatusTargetURL != "" {
			status.TargetURL = github.String(c.StatusTargetURL)
		}

		if err := c.CreateAndLogStatus(client, owner, repo, sha, status); err != nil {
			return err
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

func (c *Action) EnsureCheckSuite(pre *Target) (*github.CheckSuite, error) {
	return c.getOneOfSuitesAlreadyCreatedByGitHubActions(pre)
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

func (c *Action) getOneOfSuitesAlreadyCreatedByGitHubActions(pre *Target) (*github.CheckSuite, error) {
	client, err := c.instTokenClient()
	if err != nil {
		return nil, fmt.Errorf("Failed to create a new installation token client: %s", err)
	}

	owner, repo := pre.Owner, pre.Repo

	sha := pre.PullRequest.Head.GetSHA()

	log.Printf("Listing all suites...")

	suites, res, err := client.Checks.ListCheckSuitesForRef(context.Background(), owner, repo, sha, &github.ListCheckSuiteOptions{
	})

	c.logResponseAndError(suites, res, err)

	if err != nil {
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

func (c *Action) instTokenClient() (*github.Client, error) {
	return actions.CreateClient(os.Getenv("GITHUB_TOKEN"), c.BaseURL, c.UploadURL)
}
