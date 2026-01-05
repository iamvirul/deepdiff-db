package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/cli"
)

func TestNewPrompter(t *testing.T) {
	p := cli.NewPrompter()
	if p == nil {
		t.Fatal("NewPrompter returned nil")
	}
}

func TestNewPrompterWithIO(t *testing.T) {
	r := strings.NewReader("")
	w := &bytes.Buffer{}
	p := cli.NewPrompterWithIO(r, w)
	if p == nil {
		t.Fatal("NewPrompterWithIO returned nil")
	}
}

func TestPromptSelect(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		options  []cli.SelectOption
		wantIdx  int
		wantErr  bool
	}{
		{
			name:  "select by key",
			input: "1\n",
			options: []cli.SelectOption{
				{Key: "1", Label: "Option 1"},
				{Key: "2", Label: "Option 2"},
			},
			wantIdx: 0,
			wantErr: false,
		},
		{
			name:  "select by numeric index",
			input: "2\n",
			options: []cli.SelectOption{
				{Key: "1", Label: "Option 1"},
				{Key: "2", Label: "Option 2"},
			},
			wantIdx: 1,
			wantErr: false,
		},
		{
			name:  "quit with q",
			input: "q\n",
			options: []cli.SelectOption{
				{Key: "1", Label: "Option 1"},
			},
			wantIdx: -1,
			wantErr: false,
		},
		{
			name:  "quit with quit",
			input: "quit\n",
			options: []cli.SelectOption{
				{Key: "1", Label: "Option 1"},
			},
			wantIdx: -1,
			wantErr: false,
		},
		{
			name:  "invalid choice",
			input: "invalid\n",
			options: []cli.SelectOption{
				{Key: "1", Label: "Option 1"},
			},
			wantIdx: -1,
			wantErr: true,
		},
		{
			name:  "case insensitive",
			input: "Q\n",
			options: []cli.SelectOption{
				{Key: "1", Label: "Option 1"},
			},
			wantIdx: -1,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			w := &bytes.Buffer{}
			p := cli.NewPrompterWithIO(r, w)

			idx, err := p.PromptSelect("Choose:", tt.options)

			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if idx != tt.wantIdx {
				t.Errorf("expected index %d, got %d", tt.wantIdx, idx)
			}
		})
	}
}

func TestPromptSelectWithDescription(t *testing.T) {
	r := strings.NewReader("1\n")
	w := &bytes.Buffer{}
	p := cli.NewPrompterWithIO(r, w)

	options := []cli.SelectOption{
		{Key: "1", Label: "Option 1", Description: "Description 1"},
		{Key: "2", Label: "Option 2"},
	}

	idx, err := p.PromptSelect("Choose:", options)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx != 0 {
		t.Errorf("expected index 0, got %d", idx)
	}

	output := w.String()
	if !strings.Contains(output, "Description 1") {
		t.Error("output should contain description")
	}
}

func TestPromptConfirm(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    bool
		wantErr bool
	}{
		{"yes with y", "y\n", true, false},
		{"yes with yes", "yes\n", true, false},
		{"no with n", "n\n", false, false},
		{"no with empty", "\n", false, false},
		{"no with other", "other\n", false, false},
		{"case insensitive", "Y\n", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			w := &bytes.Buffer{}
			p := cli.NewPrompterWithIO(r, w)

			got, err := p.PromptConfirm("Continue?")
			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestPromptContinue(t *testing.T) {
	r := strings.NewReader("\n")
	w := &bytes.Buffer{}
	p := cli.NewPrompterWithIO(r, w)

	err := p.PromptContinue("Press Enter...")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPromptContinueDefault(t *testing.T) {
	r := strings.NewReader("\n")
	w := &bytes.Buffer{}
	p := cli.NewPrompterWithIO(r, w)

	err := p.PromptContinue("")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	output := w.String()
	if !strings.Contains(output, "Press Enter to continue") {
		t.Error("should use default prompt")
	}
}

func TestPrint(t *testing.T) {
	w := &bytes.Buffer{}
	r := strings.NewReader("")
	p := cli.NewPrompterWithIO(r, w)

	p.Print("test %s", "message")
	if w.String() != "test message" {
		t.Errorf("expected 'test message', got %q", w.String())
	}
}

func TestPrintln(t *testing.T) {
	w := &bytes.Buffer{}
	r := strings.NewReader("")
	p := cli.NewPrompterWithIO(r, w)

	p.Println("test", "message")
	output := w.String()
	if !strings.Contains(output, "test") || !strings.Contains(output, "message") {
		t.Errorf("Println output incorrect: %q", output)
	}
}

func TestPrintf(t *testing.T) {
	w := &bytes.Buffer{}
	r := strings.NewReader("")
	p := cli.NewPrompterWithIO(r, w)

	p.Printf("test %s\n", "message")
	output := w.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Printf output incorrect: %q", output)
	}
}

func TestPromptResolution(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    cli.ResolutionChoice
		wantErr bool
	}{
		{"keep prod", "1\n", cli.ChoiceKeepProd, false},
		{"use dev", "2\n", cli.ChoiceUseDev, false},
		{"skip", "3\n", cli.ChoiceSkip, false},
		{"ours for table", "4\n", cli.ChoiceOursForTable, false},
		{"theirs for table", "5\n", cli.ChoiceTheirsForTable, false},
		{"ours for all", "6\n", cli.ChoiceOursForAll, false},
		{"theirs for all", "7\n", cli.ChoiceTheirsForAll, false},
		{"quit", "q\n", cli.ChoiceQuit, false},
		{"invalid", "invalid\n", cli.ChoiceInvalid, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			w := &bytes.Buffer{}
			p := cli.NewPrompterWithIO(r, w)

			got, err := p.PromptResolution("users")
			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("expected choice %v, got %v", tt.want, got)
			}
		})
	}
}

func TestPromptResolutionTableName(t *testing.T) {
	r := strings.NewReader("q\n")
	w := &bytes.Buffer{}
	p := cli.NewPrompterWithIO(r, w)

	_, err := p.PromptResolution("users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := w.String()
	if !strings.Contains(output, "users") {
		t.Error("output should contain table name")
	}
}

func TestPromptSelectNumericInput(t *testing.T) {
	r := strings.NewReader("1\n")
	w := &bytes.Buffer{}
	p := cli.NewPrompterWithIO(r, w)

	options := []cli.SelectOption{
		{Key: "a", Label: "Option A"},
		{Key: "b", Label: "Option B"},
	}

	// Numeric input should work even if keys are not numeric
	idx, err := p.PromptSelect("Choose:", options)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx != 0 {
		t.Errorf("expected index 0, got %d", idx)
	}
}

func TestPromptSelectOutOfRange(t *testing.T) {
	r := strings.NewReader("99\n")
	w := &bytes.Buffer{}
	p := cli.NewPrompterWithIO(r, w)

	options := []cli.SelectOption{
		{Key: "1", Label: "Option 1"},
		{Key: "2", Label: "Option 2"},
	}

	_, err := p.PromptSelect("Choose:", options)
	if err == nil {
		t.Error("expected error for out of range input")
	}
}

func TestPromptSelectEmptyInput(t *testing.T) {
	r := strings.NewReader("\n")
	w := &bytes.Buffer{}
	p := cli.NewPrompterWithIO(r, w)

	options := []cli.SelectOption{
		{Key: "1", Label: "Option 1"},
	}

	_, err := p.PromptSelect("Choose:", options)
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestPromptSelectEOF(t *testing.T) {
	r := strings.NewReader("")
	w := &bytes.Buffer{}
	p := cli.NewPrompterWithIO(r, w)

	options := []cli.SelectOption{
		{Key: "1", Label: "Option 1"},
	}

	_, err := p.PromptSelect("Choose:", options)
	if err == nil {
		t.Error("expected error for EOF")
	}
}

func TestPromptConfirmEOF(t *testing.T) {
	r := strings.NewReader("")
	w := &bytes.Buffer{}
	p := cli.NewPrompterWithIO(r, w)

	_, err := p.PromptConfirm("Continue?")
	if err == nil {
		t.Error("expected error for EOF")
	}
}

func TestPromptContinueEOF(t *testing.T) {
	r := strings.NewReader("")
	w := &bytes.Buffer{}
	p := cli.NewPrompterWithIO(r, w)

	err := p.PromptContinue("Press Enter...")
	if err == nil {
		t.Error("expected error for EOF")
	}
}

func TestPromptSelectWithNumericKeys(t *testing.T) {
	r := strings.NewReader("1\n")
	w := &bytes.Buffer{}
	p := cli.NewPrompterWithIO(r, w)

	options := []cli.SelectOption{
		{Key: "1", Label: "Option 1"},
		{Key: "2", Label: "Option 2"},
	}

	// Should match by key first
	idx, err := p.PromptSelect("Choose:", options)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx != 0 {
		t.Errorf("expected index 0, got %d", idx)
	}
}

func TestPromptResolutionAllChoices(t *testing.T) {
	choices := []struct {
		input string
		want  cli.ResolutionChoice
	}{
		{"1\n", cli.ChoiceKeepProd},
		{"2\n", cli.ChoiceUseDev},
		{"3\n", cli.ChoiceSkip},
		{"4\n", cli.ChoiceOursForTable},
		{"5\n", cli.ChoiceTheirsForTable},
		{"6\n", cli.ChoiceOursForAll},
		{"7\n", cli.ChoiceTheirsForAll},
		{"q\n", cli.ChoiceQuit},
	}

	for _, tc := range choices {
		t.Run(tc.input, func(t *testing.T) {
			r := strings.NewReader(tc.input)
			w := &bytes.Buffer{}
			p := cli.NewPrompterWithIO(r, w)

			got, err := p.PromptResolution("test_table")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("expected %v, got %v", tc.want, got)
			}
		})
	}
}
