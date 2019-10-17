package pullvet

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/google/go-github/v28/github"
	"github.com/variantdev/go-actions"
	"github.com/variantdev/go-actions/pkg/cmd"
	"log"
	"os"
	"regexp"
	"strings"
)

const defaultNoteRegex = "[\\*]*([^\\*:]+)[\\*]*:\\s```\n([^`]+)\n```"

var newlineRegex = regexp.MustCompile(`\r\n|\r|\n`)

type Command struct {
	labels     cmd.StringSlice
	noteTitles cmd.StringSlice

	noteRegex string

	milestone string

	anyMilestone bool
	requireAny   bool
	requireAll   bool
}

func normalizeNewlines(str string) string {
	return newlineRegex.Copy().ReplaceAllString(str, "\n")
}

func New() *Command {
	return &Command{
	}
}

func (c *Command) AddFlags(fs *flag.FlagSet) {
	fs.BoolVar(&c.requireAny, "require-any", true, "If set, pullvet fails whenever the pull request was unable to fullfill all the requirements. Default: true")
	fs.BoolVar(&c.requireAll, "require-all", false, "If set, pullvet fails whenever the pull request was unable to fullfill any of the requirements. Default: false")
	fs.Var(&c.labels, "label", "Required label. When provided multiple times, pullvet succeeds if one or more of required labels exist")
	fs.BoolVar(&c.anyMilestone, "any-milestone", false, "If set, pullvet fails whenever the pull request misses a milestone")
	fs.StringVar(&c.milestone, "milestone", "", "If set, pullvet fails whenever the pull request misses a milestone")
	fs.Var(&c.noteTitles, "note", "Require a note with the specified title. pullvet fails whenever the pr misses the note in the pr description. A note can be written in Markdown as: **<title>**:\n```\n<body>\n```")
	fs.StringVar(&c.noteRegex, "note-regex", defaultNoteRegex, "Regexp pattern of each note(including the title and the body). Default: "+defaultNoteRegex)
}

func (c *Command) Run() error {
	pr, owner, repo, err := actions.PullRequest()
	if err != nil {
		return err
	}
	return c.HandlePullRequest(owner, repo, pr)
}

func (c *Command) HandlePullRequest(owner, repo string, pullRequest *github.PullRequest) error {
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

	for _, requiredLabel := range c.labels {
		if _, ok := labelSet[requiredLabel]; ok {
			any = true
			passed += 1
		} else {
			all = false
			failures = append(failures, fmt.Sprintf("missing label: %s", requiredLabel))
		}
	}

	milestone := pullRequest.Milestone.GetTitle()

	if c.milestone != "" {
		if milestone == c.milestone {
			any = true
			passed += 1
		} else {
			all = false
			failures = append(failures, fmt.Sprintf("unexpected milestone: expected %q, got %q", c.milestone, milestone))
		}
	}

	if c.anyMilestone {
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
		client, err := actions.CreateInstallationTokenClient(os.Getenv("GITHUB_TOKEN"), "", "")
		if err != nil {
			return err
		}

		pr, _, err := client.PullRequests.Get(context.Background(), owner, repo, pullRequest.GetNumber())
		if err != nil {
			return err
		}

		buf := bytes.Buffer{}
		enc := json.NewEncoder(&buf)
		enc.SetIndent("", "  ")
		if err := enc.Encode(pr); err != nil {
			return err
		}
		log.Printf("Pull request:\n%s", buf.String())

		body = pr.GetBody()
	} else {
		body = pullRequest.GetBody()
	}

	regex := regexp.MustCompile(c.noteRegex)

	allNoteMatches := regex.FindAllStringSubmatch(normalizeNewlines(body), -1)
	for _, m := range allNoteMatches {
		noteTitles[m[1]] = struct{}{}
	}

	for _, requiredNoteTitle := range c.noteTitles {
		if _, ok := noteTitles[requiredNoteTitle]; ok {
			any = true
			passed += 1
		} else {
			all = false
			failures = append(failures, fmt.Sprintf("missing note titled %q", requiredNoteTitle))
		}
	}

	if c.requireAny && !any {
		return fmt.Errorf("%d check(s) failed:\n%s\n", len(failures), formatFailures(failures))
	}

	if c.requireAll && !all {
		return fmt.Errorf("%d check(s) failed:\n%s\n", len(failures), formatFailures(failures))
	}

	fmt.Fprintf(os.Stdout, "%d check(s) passed\n", passed)

	return nil
}

func formatFailures(failures []string) string {
	var lines []string
	for _, f := range failures {
		lines = append(lines, fmt.Sprintf("* %s", f))
	}
	return strings.Join(lines, "\n")
}
