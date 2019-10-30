package pullvet

import (
	"github.com/google/go-github/v28/github"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestNoteRegex(t *testing.T) {
	testcases := []struct {
		input    string
		expected [][]string
	}{
		{
			input: "mynote1:\n```\nNOTE1\n```\n",
			expected: [][]string{
				{
					"mynote1:\n```\nNOTE1\n```",
					"mynote1",
					"NOTE1",
				},
			},
		},
		{
			input: "**mynote1**:\n```\nNOTE1\nNOTE2\n```\n",
			expected: [][]string{
				{
					"**mynote1**:\n```\nNOTE1\nNOTE2\n```",
					"mynote1",
					"NOTE1\nNOTE2",
				},
			},
		},
		{
			input: "changelog1:\r\n```\r\nchange1\r\n```\r\n\r\n**changelog2**:\r\n```\r\nchange2\r\n```\r\n",
			expected: [][]string{
				{
					"changelog1:\n```\nchange1\n```",
					"changelog1",
					"change1",
				},
				{
					"**changelog2**:\n```\nchange2\n```",
					"changelog2",
					"change2",
				},
			},
		},
		{
			input: "releasenote:\n```\nreleasenotecontent\n```",
			expected: [][]string{
				{
					"releasenote:\n```\nreleasenotecontent\n```",
					"releasenote",
					"releasenotecontent",
				},
			},
		},
		{
			input: "releasenote:\r\n```\r\nreleasenotecontent\r\n```\r\n\r\nchangelog1:\r\n```\r\nchange1\r\n```\r\n",
			expected: [][]string{
				{
					"releasenote:\n```\nreleasenotecontent\n```",
					"releasenote",
					"releasenotecontent",
				},
				{
					// ensure that \r\n\r\n before "changelog1:" needs to be removed from here
					"changelog1:\n```\nchange1\n```",
					"changelog1",
					"change1",
				},
			},
		},
		// extraneous new line between the title and the body
		{
			input: "changelog1:\r\n\r\n```\r\nchange1\r\n```\r\n",
			expected: [][]string{
				{
					// ensure that \r\n\r\n before "changelog1:" needs to be removed from here
					"changelog1:\n\n```\nchange1\n```",
					"changelog1",
					"change1",
				},
			},
		},
		// extraneous header in the start of the code block
		{
			input: "changelog1:\r\n```foobar\r\nchange1\r\n```\r\n",
			expected: [][]string{
				{
					// ensure that \r\n\r\n before "changelog1:" needs to be removed from here
					"changelog1:\n```foobar\nchange1\n```",
					"changelog1",
					"change1",
				},
			},
		},
		// inline code block in code block
		{
			input: "changelog1:\r\n```\r\nchange `foo` to `bar`\r\n second line \r\n```\r\n",
			expected: [][]string{
				{
					// ensure that \r\n\r\n before "changelog1:" needs to be removed from here
					"changelog1:\n```\nchange `foo` to `bar`\n second line \n```",
					"changelog1",
					"change `foo` to `bar`\n second line ",
				},
			},
		},
	}

	for i := range testcases {
		tc := testcases[i]

		got := regexp.MustCompile(DefaultNoteRegex).FindAllStringSubmatch(normalizeNewlines(tc.input), -1)

		if !reflect.DeepEqual(tc.expected, got) {
			t.Errorf("unexpected result:\nexpected=\n%v\n\ngot=\n%v\n", tc.expected, got)
		}
	}
}

func TestRun(t *testing.T) {
	stubPRBody := func(body string) func(owner, repo string, num int) (string, error) {
		return func(owner, repo string, num int) (string, error) {
			return body, nil
		}
	}

	testcases := []struct {
		cmd      *Action
		input    *github.PullRequest
		expected string
	}{
		{
			cmd:      &Action{RequireAny: true, Labels: []string{"v1"}, NoteRegex: DefaultNoteRegex, GetPullRequestBody: stubPRBody("")},
			input:    &github.PullRequest{Labels: []*github.Label{&github.Label{Name: github.String("v1")}}},
			expected: "",
		},
		{
			cmd:      &Action{RequireAny: true, Labels: []string{"v2"}, NoteRegex: DefaultNoteRegex, GetPullRequestBody: stubPRBody("")},
			input:    &github.PullRequest{Labels: []*github.Label{&github.Label{Name: github.String("v1")}}},
			expected: "1 check(s) failed:\n* missing label: v2",
		},
		{
			cmd:      &Action{RequireAny: true, Labels: []string{"v2", "v3"}, NoteRegex: DefaultNoteRegex, GetPullRequestBody: stubPRBody("")},
			input:    &github.PullRequest{Labels: []*github.Label{&github.Label{Name: github.String("v1")}}},
			expected: "2 check(s) failed:\n* missing label: v2\n* missing label: v3",
		},
		{
			cmd:      &Action{RequireAny: true, Milestone: "v1", NoteRegex: DefaultNoteRegex, GetPullRequestBody: stubPRBody("")},
			input:    &github.PullRequest{Milestone: &github.Milestone{Title: github.String("v1")}},
			expected: "",
		},
		{
			cmd:      &Action{RequireAny: true, AnyMilestone: true, NoteRegex: DefaultNoteRegex, GetPullRequestBody: stubPRBody("")},
			input:    &github.PullRequest{Milestone: &github.Milestone{Title: github.String("v1")}},
			expected: "",
		},
		{
			cmd:      &Action{RequireAny: true, Milestone: "v2", NoteRegex: DefaultNoteRegex, GetPullRequestBody: stubPRBody("")},
			input:    &github.PullRequest{Milestone: &github.Milestone{Title: github.String("v1")}},
			expected: "1 check(s) failed:\n* unexpected milestone: expected \"v2\", got \"v1\"",
		},
		{
			cmd:      &Action{RequireAny: true, AnyMilestone: true, NoteRegex: DefaultNoteRegex, GetPullRequestBody: stubPRBody("")},
			input:    &github.PullRequest{},
			expected: "1 check(s) failed:\n* missing milestone",
		},
		{
			cmd: &Action{RequireAny: true, Labels: []string{"v2", "v3"}, NoteRegex: DefaultNoteRegex, GetPullRequestBody: stubPRBody("")},
			input: &github.PullRequest{
				Labels: []*github.Label{
					&github.Label{Name: github.String("v2")},
					&github.Label{Name: github.String("v1")},
				},
			},
			expected: "",
		},
		{
			cmd: &Action{RequireAll: true, Labels: []string{"v2", "v3"}, NoteRegex: DefaultNoteRegex, GetPullRequestBody: stubPRBody("")},
			input: &github.PullRequest{
				Labels: []*github.Label{
					&github.Label{Name: github.String("v2")},
					&github.Label{Name: github.String("v1")},
				},
			},
			expected: "1 check(s) failed:\n* missing label: v3",
		},
		//
		// -label-match
		//
		{
			cmd: &Action{RequireAll: true, LabelMatches: []string{"release-v.+"}, NoteRegex: DefaultNoteRegex, GetPullRequestBody: stubPRBody("")},
			input: &github.PullRequest{
				Labels: []*github.Label{
					&github.Label{Name: github.String("rel-v1")},
				},
			},
			expected: "1 check(s) failed:\n* no label matched \"release-v.+\"",
		},
		{
			cmd: &Action{RequireAll: true, LabelMatches: []string{"release-v.+", "releasenote/none"}, NoteRegex: DefaultNoteRegex, GetPullRequestBody: stubPRBody("")},
			input: &github.PullRequest{
				Labels: []*github.Label{
					&github.Label{Name: github.String("release-v1")},
					&github.Label{Name: github.String("releasenote/none")},
				},
			},
			expected: "",
		},
		{
			cmd: &Action{RequireAll: true, LabelMatches: []string{"release-v.+", "releasenote/none"}, NoteRegex: DefaultNoteRegex, GetPullRequestBody: stubPRBody("")},
			input: &github.PullRequest{
				Labels: []*github.Label{
					&github.Label{Name: github.String("rel-v1")},
					&github.Label{Name: github.String("releasenote/none")},
				},
			},
			expected: "1 check(s) failed:\n* no label matched \"release-v.+\"",
		},
		{
			cmd: &Action{RequireAny: true, LabelMatches: []string{"release-v.+", "releasenote/none"}, NoteRegex: DefaultNoteRegex, GetPullRequestBody: stubPRBody("")},
			input: &github.PullRequest{
				Labels: []*github.Label{
					&github.Label{Name: github.String("rel-v1")},
					&github.Label{Name: github.String("releasenote/none")},
				},
			},
			expected: "",
		},
		{
			cmd: &Action{RequireAll: true, LabelMatches: []string{"release-v.+"}, NoteRegex: DefaultNoteRegex, GetPullRequestBody: stubPRBody("")},
			input: &github.PullRequest{
				Labels: []*github.Label{
					&github.Label{Name: github.String("release-v1")},
				},
			},
			expected: "",
		},
		//
		// -milestone-match
		//
		{
			cmd: &Action{RequireAny: true, MilestoneMatches: []string{"release-v.+"}, NoteRegex: DefaultNoteRegex, GetPullRequestBody: stubPRBody("")},
			input: &github.PullRequest{
				Milestone: &github.Milestone{
					Title: github.String("rel-v1"),
				},
			},
			expected: "1 check(s) failed:\n* milestone did not match \"release-v.+\"",
		},
		{
			cmd: &Action{RequireAny: true, MilestoneMatches: []string{"release-v.+", "rel-v.+"}, NoteRegex: DefaultNoteRegex, GetPullRequestBody: stubPRBody("")},
			input: &github.PullRequest{
				Milestone: &github.Milestone{
					Title: github.String("rel-v1"),
				},
			},
			expected: "",
		},
	}

	for i := range testcases {
		tc := testcases[i]

		err := tc.cmd.HandlePullRequest("myuser", "myrepo", tc.input)

		if tc.expected != "" && !strings.Contains(err.Error(), tc.expected) {
			t.Errorf("unexpected error: expected=%q, got=%q", tc.expected, err)
		}

		if tc.expected == "" && err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	}
}
