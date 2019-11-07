package pullnote

import (
	"flag"

	"github.com/variantdev/go-actions/pkg/cli"
	"github.com/variantdev/go-actions/pkg/pullnote"
	"github.com/variantdev/go-actions/pkg/pullvet"
)

var Command cli.Command = &cmd{}

type cmd struct {}

func (c *cmd) Name() string {
	return "pullnote"
}

func (c *cmd) Run(args []string) error {
	action := pullnote.New()

	usage := `pullnote extracts notes detected by pullvet. The input should be provided as NDJSON via stdin or a file
`

	if err := cli.Setup(c, args, usage, func(fs *flag.FlagSet) {
		fs.StringVar(&action.File, "file", "", "NDJSON file to read")
		fs.StringVar(&action.NoteRegex, "note-regex", pullvet.DefaultNoteRegex, "Regexp pattern of each note(including the title and the body)")
		fs.StringVar(&action.BodyKey, "body-key", "body", "Read pull request body containing notes from this key in the JSON object read from a NDJSON line")
		fs.StringVar(&action.KindKey, "kind-key", "kind", "Extracted kind of the note is set to this key")
		fs.StringVar(&action.DescKey, "desc-key", "desc", "Extracted description of the note is set to this key")
	}); err != nil {
		return err
	}

	return action.Run()
}
