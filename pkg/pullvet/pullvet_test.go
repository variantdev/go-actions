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
	}

	for i := range testcases {
		tc := testcases[i]

		got := regexp.MustCompile(defaultNoteRegex).FindAllStringSubmatch(tc.input, -1)

		if !reflect.DeepEqual(tc.expected, got) {
			t.Errorf("unexpected result:\nexpected=\n%v\n\ngot=\n%v\n", tc.expected, got)
		}
	}
}

func TestRun(t *testing.T) {
	testcases := []struct {
		cmd      *Command
		input    *github.PullRequest
		expected string
	}{
		{
			cmd:      &Command{requireAny: true, labels: []string{"v1"}},
			input:    &github.PullRequest{Labels: []*github.Label{&github.Label{Name: github.String("v1")}}},
			expected: "",
		},
		{
			cmd:      &Command{requireAny: true, labels: []string{"v2"}},
			input:    &github.PullRequest{Labels: []*github.Label{&github.Label{Name: github.String("v1")}}},
			expected: "1 check(s) failed:\n* missing label: v2",
		},
		{
			cmd:      &Command{requireAny: true, labels: []string{"v2", "v3"}},
			input:    &github.PullRequest{Labels: []*github.Label{&github.Label{Name: github.String("v1")}}},
			expected: "2 check(s) failed:\n* missing label: v2\n* missing label: v3",
		},
		{
			cmd:      &Command{requireAny: true, milestone: "v1"},
			input:    &github.PullRequest{Milestone: &github.Milestone{Title: github.String("v1")}},
			expected: "",
		},
		{
			cmd:      &Command{requireAny: true, anyMilestone: true},
			input:    &github.PullRequest{Milestone: &github.Milestone{Title: github.String("v1")}},
			expected: "",
		},
		{
			cmd:      &Command{requireAny: true, milestone: "v2"},
			input:    &github.PullRequest{Milestone: &github.Milestone{Title: github.String("v1")}},
			expected: "1 check(s) failed:\n* unexpected milestone: expected \"v2\", got \"v1\"",
		},
		{
			cmd:      &Command{requireAny: true, anyMilestone: true},
			input:    &github.PullRequest{},
			expected: "1 check(s) failed:\n* missing milestone",
		},
		{
			cmd: &Command{requireAny: true, labels: []string{"v2", "v3"}},
			input: &github.PullRequest{
				Labels: []*github.Label{
					&github.Label{Name: github.String("v2")},
					&github.Label{Name: github.String("v1")},
				},
			},
			expected: "",
		},
		{
			cmd: &Command{requireAll: true, labels: []string{"v2", "v3"}},
			input: &github.PullRequest{
				Labels: []*github.Label{
					&github.Label{Name: github.String("v2")},
					&github.Label{Name: github.String("v1")},
				},
			},
			expected: "1 check(s) failed:\n* missing label: v3",
		},
	}

	for i := range testcases {
		tc := testcases[i]

		err := tc.cmd.HandlePullRequest(tc.input)

		if tc.expected != "" && !strings.Contains(err.Error(), tc.expected) {
			t.Errorf("unexpected error: expected=%q, got=%q", tc.expected, err)
		}

		if tc.expected == "" && err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	}
}
