// Package cli provides interactive CLI utilities for user input and display.
package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// Prompter handles interactive prompts with the user.
type Prompter struct {
	reader *bufio.Reader
	writer io.Writer
}

// NewPrompter creates a new Prompter using stdin and stdout.
func NewPrompter() *Prompter {
	return &Prompter{
		reader: bufio.NewReader(os.Stdin),
		writer: os.Stdout,
	}
}

// NewPrompterWithIO creates a Prompter with custom input/output.
// Useful for testing.
func NewPrompterWithIO(r io.Reader, w io.Writer) *Prompter {
	return &Prompter{
		reader: bufio.NewReader(r),
		writer: w,
	}
}

// SelectOption represents an option in a select prompt.
type SelectOption struct {
	Key         string // Short key like "1", "2", "q"
	Label       string // Display label
	Description string // Optional description
}

// PromptSelect displays options and gets user choice.
// Returns the index of the selected option (0-based) or -1 for quit.
// Special keys: "q" returns -1 (quit).
func (p *Prompter) PromptSelect(prompt string, options []SelectOption) (int, error) {
	// Display prompt and options
	fmt.Fprintln(p.writer, prompt)
	for _, opt := range options {
		if opt.Description != "" {
			fmt.Fprintf(p.writer, "  [%s] %s - %s\n", opt.Key, opt.Label, opt.Description)
		} else {
			fmt.Fprintf(p.writer, "  [%s] %s\n", opt.Key, opt.Label)
		}
	}
	fmt.Fprint(p.writer, "\nEnter choice: ")

	// Read input
	input, err := p.reader.ReadString('\n')
	if err != nil {
		return -1, fmt.Errorf("read input: %w", err)
	}

	input = strings.TrimSpace(strings.ToLower(input))

	// Handle quit
	if input == "q" || input == "quit" {
		return -1, nil
	}

	// Find matching option
	for i, opt := range options {
		if strings.ToLower(opt.Key) == input {
			return i, nil
		}
	}

	// Try numeric input
	if num, err := strconv.Atoi(input); err == nil {
		// Convert 1-based input to 0-based index
		idx := num - 1
		if idx >= 0 && idx < len(options) {
			return idx, nil
		}
	}

	return -1, fmt.Errorf("invalid choice: %s", input)
}

// PromptConfirm asks for yes/no confirmation.
// Returns true for yes, false for no.
func (p *Prompter) PromptConfirm(prompt string) (bool, error) {
	fmt.Fprintf(p.writer, "%s [y/N]: ", prompt)

	input, err := p.reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("read input: %w", err)
	}

	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes", nil
}

// PromptContinue waits for user to press Enter to continue.
func (p *Prompter) PromptContinue(prompt string) error {
	if prompt == "" {
		prompt = "Press Enter to continue..."
	}
	fmt.Fprint(p.writer, prompt)
	_, err := p.reader.ReadString('\n')
	return err
}

// Print writes a message to the output.
func (p *Prompter) Print(format string, args ...any) {
	fmt.Fprintf(p.writer, format, args...)
}

// Println writes a message with newline to the output.
func (p *Prompter) Println(args ...any) {
	fmt.Fprintln(p.writer, args...)
}

// Printf writes a formatted message to the output.
func (p *Prompter) Printf(format string, args ...any) {
	fmt.Fprintf(p.writer, format, args...)
}

// ResolutionChoice represents the user's resolution choice.
type ResolutionChoice int

const (
	// ChoiceKeepProd keeps the production value (ours).
	ChoiceKeepProd ResolutionChoice = iota
	// ChoiceUseDev uses the development value (theirs).
	ChoiceUseDev
	// ChoiceSkip leaves the conflict pending.
	ChoiceSkip
	// ChoiceOursForTable applies "ours" to all remaining in this table.
	ChoiceOursForTable
	// ChoiceTheirsForTable applies "theirs" to all remaining in this table.
	ChoiceTheirsForTable
	// ChoiceOursForAll applies "ours" to all remaining conflicts.
	ChoiceOursForAll
	// ChoiceTheirsForAll applies "theirs" to all remaining conflicts.
	ChoiceTheirsForAll
	// ChoiceQuit saves and exits.
	ChoiceQuit
	// ChoiceInvalid represents an invalid choice.
	ChoiceInvalid
)

// PromptResolution prompts the user for a resolution decision.
// Returns the choice and any error.
func (p *Prompter) PromptResolution(tableName string) (ResolutionChoice, error) {
	options := []SelectOption{
		{Key: "1", Label: "Keep production (ours)", Description: "Discard dev changes"},
		{Key: "2", Label: "Use development (theirs)", Description: "Apply dev changes to prod"},
		{Key: "3", Label: "Skip", Description: "Leave pending for later"},
		{Key: "4", Label: fmt.Sprintf("Apply \"ours\" to all in %s", tableName)},
		{Key: "5", Label: fmt.Sprintf("Apply \"theirs\" to all in %s", tableName)},
		{Key: "6", Label: "Apply \"ours\" to ALL remaining"},
		{Key: "7", Label: "Apply \"theirs\" to ALL remaining"},
		{Key: "q", Label: "Quit and save progress"},
	}

	idx, err := p.PromptSelect("Choose resolution:", options)
	if err != nil {
		// Check if it's an invalid choice error
		if strings.Contains(err.Error(), "invalid choice") {
			return ChoiceInvalid, nil
		}
		return ChoiceInvalid, err
	}

	if idx == -1 {
		return ChoiceQuit, nil
	}

	switch idx {
	case 0:
		return ChoiceKeepProd, nil
	case 1:
		return ChoiceUseDev, nil
	case 2:
		return ChoiceSkip, nil
	case 3:
		return ChoiceOursForTable, nil
	case 4:
		return ChoiceTheirsForTable, nil
	case 5:
		return ChoiceOursForAll, nil
	case 6:
		return ChoiceTheirsForAll, nil
	case 7:
		return ChoiceQuit, nil
	default:
		return ChoiceInvalid, nil
	}
}
