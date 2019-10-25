package pullvet

import (
	"flag"

	"github.com/variantdev/go-actions/pkg/cli"
	"github.com/variantdev/go-actions/pkg/pullvet"
)

var Command cli.Command = &cmd{}

type cmd struct {}

func (c *cmd) Name() string {
	return "pullvet"
}

func (c *cmd) Run(args []string) error {
	action := pullvet.New()

	usage := `pullvet checks for the existence of the specified pull request label(s)  exists with a non-zero status
whenever one ore more required labels are missing in the pull request

This should be useful for compliance purpose. that is, it will help preventing any  pr from being merged when it misses required labels.
when run on GitHub Actions v2
`

	if err := cli.Setup(c, args, usage, func(fs *flag.FlagSet) {
		fs.BoolVar(&action.RequireAny, "require-any", true, "If set, pullvet fails whenever the pull request was unable to fullfill all the requirements")
		fs.BoolVar(&action.RequireAll, "require-all", false, "If set, pullvet fails whenever the pull request was unable to fullfill any of the requirements")
		fs.Var(&action.Labels, "label", "Required label. When provided multiple times, pullvet succeeds if one or more of required labels exist")
		fs.BoolVar(&action.AnyMilestone, "any-milestone", false, "If set, pullvet fails whenever the pull request misses a milestone")
		fs.StringVar(&action.Milestone, "milestone", "", "If set, pullvet fails whenever the pull request misses a milestone")
		fs.Var(&action.LabelMatches, "label-match", "Regexp pattern to match label name against. If set, pullvet tries to find the label matches any of patterns and fail if none matched.")
		fs.Var(&action.MilestoneMatches, "milestone-match", "Regexp pattern to match milestone title against. If set, pullvet tries to find the milestone matches any of patterns and fail if none matched.")
		fs.Var(&action.NoteTitles, "note", "Require a note with the specified title. pullvet fails whenever the pr misses the note in the pr description. A note can be written in Markdown as: **<title>**:\n```\n<body>\n```")
		fs.Var(&action.RequireApprovalsBy, "approved-by", "Require approval from user(s). Use GitHub login name like `mumoshu` without `@`")
		fs.IntVar(&action.MinApprovals, "min-approvals", 0, "Require N approval(s)")
		fs.StringVar(&action.NoteRegex, "note-regex", pullvet.DefaultNoteRegex, "Regexp pattern of each note(including the title and the body)")
	}); err != nil {
		return err
	}

	return action.Run()
}
