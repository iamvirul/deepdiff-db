package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/cli"
)

func TestPromptSelectValidChoice(t *testing.T) {
	input := strings.NewReader("1\n")
	output := &bytes.Buffer{}

	p := cli.NewPrompterWithIO(input, output)

	options := []cli.SelectOption{
		{Key: "1", Label: "Option One"},
		{Key: "2", Label: "Option Two"},
	}

	idx, err := p.PromptSelect("Choose:", options)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx != 0 {
		t.Errorf("expected index 0, got %d", idx)
	}
}

func TestPromptSelectSecondOption(t *testing.T) {
	input := strings.NewReader("2\n")
	output := &bytes.Buffer{}

	p := cli.NewPrompterWithIO(input, output)

	options := []cli.SelectOption{
		{Key: "1", Label: "Option One"},
		{Key: "2", Label: "Option Two"},
	}

	idx, err := p.PromptSelect("Choose:", options)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx != 1 {
		t.Errorf("expected index 1, got %d", idx)
	}
}

func TestPromptSelectQuit(t *testing.T) {
	input := strings.NewReader("q\n")
	output := &bytes.Buffer{}

	p := cli.NewPrompterWithIO(input, output)

	options := []cli.SelectOption{
		{Key: "1", Label: "Option One"},
	}

	idx, err := p.PromptSelect("Choose:", options)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx != -1 {
		t.Errorf("expected -1 for quit, got %d", idx)
	}
}

func TestPromptSelectQuitWord(t *testing.T) {
	input := strings.NewReader("quit\n")
	output := &bytes.Buffer{}

	p := cli.NewPrompterWithIO(input, output)

	options := []cli.SelectOption{
		{Key: "1", Label: "Option One"},
	}

	idx, err := p.PromptSelect("Choose:", options)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx != -1 {
		t.Errorf("expected -1 for quit, got %d", idx)
	}
}

func TestPromptSelectInvalidChoice(t *testing.T) {
	input := strings.NewReader("invalid\n")
	output := &bytes.Buffer{}

	p := cli.NewPrompterWithIO(input, output)

	options := []cli.SelectOption{
		{Key: "1", Label: "Option One"},
	}

	_, err := p.PromptSelect("Choose:", options)
	if err == nil {
		t.Error("expected error for invalid choice")
	}
	if !strings.Contains(err.Error(), "invalid choice") {
		t.Errorf("expected 'invalid choice' error, got: %v", err)
	}
}

func TestPromptSelectCaseInsensitive(t *testing.T) {
	input := strings.NewReader("Q\n")
	output := &bytes.Buffer{}

	p := cli.NewPrompterWithIO(input, output)

	options := []cli.SelectOption{
		{Key: "1", Label: "Option One"},
	}

	idx, err := p.PromptSelect("Choose:", options)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx != -1 {
		t.Errorf("expected -1 for quit (uppercase Q), got %d", idx)
	}
}

func TestPromptSelectWithDescription(t *testing.T) {
	input := strings.NewReader("1\n")
	output := &bytes.Buffer{}

	p := cli.NewPrompterWithIO(input, output)

	options := []cli.SelectOption{
		{Key: "1", Label: "Option One", Description: "This is option one"},
	}

	_, err := p.PromptSelect("Choose:", options)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that description is displayed
	if !strings.Contains(output.String(), "This is option one") {
		t.Error("description should be displayed in output")
	}
}

func TestPromptConfirmYes(t *testing.T) {
	input := strings.NewReader("y\n")
	output := &bytes.Buffer{}

	p := cli.NewPrompterWithIO(input, output)

	result, err := p.PromptConfirm("Are you sure?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result {
		t.Error("expected true for 'y'")
	}
}

func TestPromptConfirmYesWord(t *testing.T) {
	input := strings.NewReader("yes\n")
	output := &bytes.Buffer{}

	p := cli.NewPrompterWithIO(input, output)

	result, err := p.PromptConfirm("Are you sure?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result {
		t.Error("expected true for 'yes'")
	}
}

func TestPromptConfirmNo(t *testing.T) {
	input := strings.NewReader("n\n")
	output := &bytes.Buffer{}

	p := cli.NewPrompterWithIO(input, output)

	result, err := p.PromptConfirm("Are you sure?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result {
		t.Error("expected false for 'n'")
	}
}

func TestPromptConfirmEmpty(t *testing.T) {
	input := strings.NewReader("\n")
	output := &bytes.Buffer{}

	p := cli.NewPrompterWithIO(input, output)

	result, err := p.PromptConfirm("Are you sure?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result {
		t.Error("expected false for empty input (default is N)")
	}
}

func TestPromptResolutionKeepProd(t *testing.T) {
	input := strings.NewReader("1\n")
	output := &bytes.Buffer{}

	p := cli.NewPrompterWithIO(input, output)

	choice, err := p.PromptResolution("users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if choice != cli.ChoiceKeepProd {
		t.Errorf("expected ChoiceKeepProd, got %d", choice)
	}
}

func TestPromptResolutionUseDev(t *testing.T) {
	input := strings.NewReader("2\n")
	output := &bytes.Buffer{}

	p := cli.NewPrompterWithIO(input, output)

	choice, err := p.PromptResolution("users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if choice != cli.ChoiceUseDev {
		t.Errorf("expected ChoiceUseDev, got %d", choice)
	}
}

func TestPromptResolutionSkip(t *testing.T) {
	input := strings.NewReader("3\n")
	output := &bytes.Buffer{}

	p := cli.NewPrompterWithIO(input, output)

	choice, err := p.PromptResolution("users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if choice != cli.ChoiceSkip {
		t.Errorf("expected ChoiceSkip, got %d", choice)
	}
}

func TestPromptResolutionOursForTable(t *testing.T) {
	input := strings.NewReader("4\n")
	output := &bytes.Buffer{}

	p := cli.NewPrompterWithIO(input, output)

	choice, err := p.PromptResolution("users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if choice != cli.ChoiceOursForTable {
		t.Errorf("expected ChoiceOursForTable, got %d", choice)
	}
}

func TestPromptResolutionTheirsForTable(t *testing.T) {
	input := strings.NewReader("5\n")
	output := &bytes.Buffer{}

	p := cli.NewPrompterWithIO(input, output)

	choice, err := p.PromptResolution("users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if choice != cli.ChoiceTheirsForTable {
		t.Errorf("expected ChoiceTheirsForTable, got %d", choice)
	}
}

func TestPromptResolutionOursForAll(t *testing.T) {
	input := strings.NewReader("6\n")
	output := &bytes.Buffer{}

	p := cli.NewPrompterWithIO(input, output)

	choice, err := p.PromptResolution("users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if choice != cli.ChoiceOursForAll {
		t.Errorf("expected ChoiceOursForAll, got %d", choice)
	}
}

func TestPromptResolutionTheirsForAll(t *testing.T) {
	input := strings.NewReader("7\n")
	output := &bytes.Buffer{}

	p := cli.NewPrompterWithIO(input, output)

	choice, err := p.PromptResolution("users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if choice != cli.ChoiceTheirsForAll {
		t.Errorf("expected ChoiceTheirsForAll, got %d", choice)
	}
}

func TestPromptResolutionQuit(t *testing.T) {
	input := strings.NewReader("q\n")
	output := &bytes.Buffer{}

	p := cli.NewPrompterWithIO(input, output)

	choice, err := p.PromptResolution("users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if choice != cli.ChoiceQuit {
		t.Errorf("expected ChoiceQuit, got %d", choice)
	}
}

func TestPromptResolutionInvalid(t *testing.T) {
	input := strings.NewReader("invalid\n")
	output := &bytes.Buffer{}

	p := cli.NewPrompterWithIO(input, output)

	choice, err := p.PromptResolution("users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if choice != cli.ChoiceInvalid {
		t.Errorf("expected ChoiceInvalid, got %d", choice)
	}
}

func TestPromptResolutionShowsTableName(t *testing.T) {
	input := strings.NewReader("1\n")
	output := &bytes.Buffer{}

	p := cli.NewPrompterWithIO(input, output)

	_, _ = p.PromptResolution("customers")

	// Check that table name appears in options
	if !strings.Contains(output.String(), "customers") {
		t.Error("table name should appear in resolution options")
	}
}
