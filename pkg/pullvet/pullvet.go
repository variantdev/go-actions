package pullvet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/google/go-github/v28/github"
	"github.com/variantdev/go-actions"
)

const DefaultNoteRegex = "[\\*]*([^\\*\r\n:]+)[\\*]*:\\s+```[^\n]*\n((?s).*?)\n```"

var newlineRegex = regexp.MustCompile(`\r\n|\r|\n`)

type Action struct {
	Labels     actions.StringSlice
	NoteTitles actions.StringSlice

	LabelMatches     actions.StringSlice
	MilestoneMatches actions.StringSlice

	NoteRegex string

	Milestone string

	AnyMilestone bool
	RequireAny   bool
	RequireAll   bool

	MinApprovals       int
	RequireApprovalsBy actions.StringSlice

	GetPullRequestBody func(string, string, int) (string, error)
}

func normalizeNewlines(str string) string {
	return newlineRegex.ReplaceAllString(str, "\n")
}

func New() *Action {
	return &Action{
		GetPullRequestBody: GetPullRequestBody,
	}
}

func (c *Action) Run() error {
	pr, owner, repo, err := actions.PullRequest()
	if err != nil {
		return err
	}
	return c.HandlePullRequest(owner, repo, pr)
}

func (c *Action) HandlePullRequest(owner, repo string, pullRequest *github.PullRequest) error {
	var labels []string
	labelSet := map[string]struct{}{}

	for _, l := range pullRequest.Labels {
		label := l.GetName()
		labelSet[label] = struct{}{}
		labels = append(labels, label)
	}

	any := false
	all := true

	var passed int
	var failures []string

	for _, requiredLabel := range c.Labels {
		if _, ok := labelSet[requiredLabel]; ok {
			any = true
			passed += 1
		} else {
			all = false
			failures = append(failures, fmt.Sprintf("missing label: %s", requiredLabel))
		}
	}

	labelRegexs := []*regexp.Regexp{}
	for _, p := range c.LabelMatches {
		labelRegexs = append(labelRegexs, regexp.MustCompile(p))
	}

	for _, r := range labelRegexs {
		var matched bool

		for _, lbl := range labels {
			if r.MatchString(lbl) {
				matched = true
			}
		}

		if matched {
			any = true
			passed += 1
		} else {
			all = false
			failures = append(failures, fmt.Sprintf("no label matched %q", r.String()))
		}
	}

	milestone := pullRequest.Milestone.GetTitle()

	if c.Milestone != "" {
		if milestone == c.Milestone {
			any = true
			passed += 1
		} else {
			all = false
			failures = append(failures, fmt.Sprintf("unexpected milestone: expected %q, got %q", c.Milestone, milestone))
		}
	}

	milestoneRegexs := []*regexp.Regexp{}
	for _, p := range c.MilestoneMatches {
		milestoneRegexs = append(milestoneRegexs, regexp.MustCompile(p))
	}

	for _, r := range milestoneRegexs {
		matched := r.MatchString(milestone)
		if matched {
			any = true
			passed += 1
		} else {
			all = false
			failures = append(failures, fmt.Sprintf("milestone did not match %q", r.String()))
		}
	}

	if len(c.RequireApprovalsBy) > 0 || c.MinApprovals > 0 {
		client, err := actions.CreateClient(os.Getenv("GITHUB_TOKEN"), "", "")
		if err != nil {
			return err
		}
		reviews, res, err := client.PullRequests.ListReviews(context.Background(), owner, repo, pullRequest.GetNumber(), &github.ListOptions{})
		if err != nil && res.StatusCode != 404 {
			return err
		}

		approvedUsers := map[string]struct{}{}
		for _, r := range reviews {
			approvedUsers[r.User.GetLogin()] = struct{}{}
		}

		if len(c.RequireApprovalsBy) > 0 {
			allApproved := true
			for _, u := range c.RequireApprovalsBy {
				_, ok := approvedUsers[u]
				allApproved = allApproved && ok
			}
			if allApproved {
				any = true
			} else {
				all = false
			}
		}

		if c.MinApprovals > 0 {
			if len(approvedUsers) > c.MinApprovals {
				any = true
			} else {
				all = false
			}
		}
	}

	if c.AnyMilestone {
		if milestone != "" {
			any = true
			passed += 1
		} else {
			all = false
			failures = append(failures, "missing milestone")
		}
	}

	noteTitles := map[string]struct{}{}

	var body string

	if owner != "" {
		var err error
		body, err = c.GetPullRequestBody(owner, repo, pullRequest.GetNumber())
		if err != nil {
			return err
		}
	} else {
		body = pullRequest.GetBody()
	}

	regex := regexp.MustCompile(c.NoteRegex)

	allNoteMatches := regex.FindAllStringSubmatch(normalizeNewlines(body), -1)
	for _, m := range allNoteMatches {
		log.Printf("match: %v", m)
		noteTitles[m[1]] = struct{}{}
	}

	log.Printf("note titles: %v", noteTitles)

	for _, requiredNoteTitle := range c.NoteTitles {
		if _, ok := noteTitles[requiredNoteTitle]; ok {
			any = true
			passed += 1
		} else {
			all = false
			failures = append(failures, fmt.Sprintf("missing note titled %q", requiredNoteTitle))
		}
	}

	if (c.RequireAny && !any) || c.RequireAll && !all {
		e := fmt.Errorf("%d check(s) failed:\n%s\n", len(failures), formatFailures(failures))

		fmt.Fprintf(os.Stdout, "%s\n", e.Error())

		return e
	}

	fmt.Fprintf(os.Stdout, "%d check(s) passed\n", passed)

	return nil
}

func GetPullRequestBody(owner, repo string, prNumber int) (string, error) {
	client, err := actions.CreateClient(os.Getenv("GITHUB_TOKEN"), "", "")
	if err != nil {
		return "", err
	}

	pr, _, err := client.PullRequests.Get(context.Background(), owner, repo, prNumber)
	if err != nil {
		return "", err
	}

	buf := bytes.Buffer{}
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(pr); err != nil {
		return "", err
	}
	log.Printf("Pull request:\n%s", buf.String())

	return pr.GetBody(), nil
}

func formatFailures(failures []string) string {
	var lines []string
	for _, f := range failures {
		lines = append(lines, fmt.Sprintf("* %s", f))
	}
	return strings.Join(lines, "\n")
}
