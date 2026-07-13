package sessions

import (
	"errors"
	"testing"
)

func TestValidateAgentTextRejectsOnlyEmptyText(t *testing.T) {
	if err := validateAgentText(""); !errors.Is(err, ErrAgentTextRequired) {
		t.Fatalf("validateAgentText() error = %v, want %v", err, ErrAgentTextRequired)
	}

	if err := validateAgentText(" \n\t"); err != nil {
		t.Fatalf("validateAgentText() rejected whitespace: %v", err)
	}
}
