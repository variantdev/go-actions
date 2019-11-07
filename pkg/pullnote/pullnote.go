package pullnote

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
)

var newlineRegex = regexp.MustCompile(`\r\n|\r|\n`)

type Action struct {
	NoteRegex string
	File      string
	BodyKey   string
	KindKey   string
	DescKey   string
}

func normalizeNewlines(str string) string {
	return newlineRegex.ReplaceAllString(str, "\n")
}

func New() *Action {
	return &Action{
	}
}

func (c *Action) Run() error {
	regex := regexp.MustCompile(c.NoteRegex)

	var in io.Reader

	if c.File != "" {
		file, err := os.Open(c.File)
		if err != nil {
			return err
		}
		defer file.Close()
		in = file
	} else {
		in = os.Stdin
	}

	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		line := scanner.Text()
		m := map[string]interface{}{}
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			return err
		}
		key := c.BodyKey
		value, ok := m[key]
		if !ok {
			return fmt.Errorf("required key %q does not exist: %s", key, line)
		}

		body, ok := value.(string)
		if !ok {
			return fmt.Errorf("unexpected type of body: expected string, got %T(%v)", value, value)
		}

		allNoteMatches := regex.FindAllStringSubmatch(normalizeNewlines(body), -1)
		for _, match := range allNoteMatches {
			kind, desc := match[1], match[2]
			delete(m, c.BodyKey)
			m[c.KindKey] = kind
			m[c.DescKey] = desc
			res, err := json.Marshal(m)
			if err != nil {
				return err
			}
			println(string(res))
		}
	}

	return nil
}
