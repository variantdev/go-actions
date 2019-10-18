package actions

import (
	"strings"
	"testing"
)

func TestRunCmd(t *testing.T) {
	stdout, fullout, err := RunCmd("sh", []string{"-c", "echo stdout1; echo stderr1 1>&2"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if stdout != "stdout1\n" {
		t.Errorf("unexpected stdout: %s", stdout)
	}

	if !strings.Contains(fullout, "stdout1\n") || !strings.Contains(fullout, "stderr1\n") {
		t.Errorf("unexpected fullout: %s", fullout)
	}
}
