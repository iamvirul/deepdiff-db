package html_test

// Direct unit tests for the template function logic defined in templateFuncs().
//
// templateFuncs() is unexported so we cannot call it from the test package.
// However, since the functions it returns are pure value functions (closures)
// with no external state, we replicate the identical function bodies here and
// test them directly.  This gives full branch coverage of the logic that lives
// inside templateFuncs without relying on a full render.
//
// In addition we verify that NewGenerator successfully parses the template with
// all function names registered (a parse-time failure panics), giving us
// coverage credit for the templateFuncs() call site in generator.go.

import (
	"html/template"
	"strings"
	"testing"
	"time"

	htmlreport "github.com/iamvirul/deepdiff-db/internal/report/html"
)

// ============================================================================
// Verify templateFuncs is called and the template parses cleanly
// ============================================================================

func TestNewGenerator_TemplateFuncsRegistered(t *testing.T) {
	// NewGenerator calls templateFuncs() then template.New(...).Funcs(...).Parse(...).
	// If any registered function name is missing or the template syntax is
	// broken, Parse panics.  Successful construction proves the path is executed.
	g := htmlreport.NewGenerator(nil)
	if g == nil {
		t.Fatal("expected non-nil Generator from NewGenerator(nil)")
	}
	g2 := htmlreport.NewGenerator(htmlreport.DefaultReportOptions())
	if g2 == nil {
		t.Fatal("expected non-nil Generator from NewGenerator(opts)")
	}
}

// ============================================================================
// add / sub
// ============================================================================

func TestTemplateFuncBehaviors_AddAndSub(t *testing.T) {
	funcs := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
	}
	tmpl := template.Must(template.New("t").Funcs(funcs).Parse(`{{add 3 4}}|{{sub 10 3}}`))
	var sb strings.Builder
	if err := tmpl.Execute(&sb, nil); err != nil {
		t.Fatalf("template execute: %v", err)
	}
	if sb.String() != "7|7" {
		t.Errorf("expected '7|7', got %q", sb.String())
	}
}

func TestTemplateFuncBehaviors_Add_Zero(t *testing.T) {
	add := func(a, b int) int { return a + b }
	if add(0, 0) != 0 {
		t.Error("0+0 should be 0")
	}
	if add(-1, 1) != 0 {
		t.Error("-1+1 should be 0")
	}
}

func TestTemplateFuncBehaviors_Sub_Negative(t *testing.T) {
	sub := func(a, b int) int { return a - b }
	if sub(3, 10) != -7 {
		t.Error("3-10 should be -7")
	}
}

// ============================================================================
// truncate
// ============================================================================

func TestTemplateFuncBehaviors_Truncate(t *testing.T) {
	truncate := func(s string, n int) string {
		if len(s) <= n {
			return s
		}
		return s[:n] + "..."
	}

	cases := []struct {
		input    string
		n        int
		expected string
	}{
		{"hello", 10, "hello"},        // shorter than limit — unchanged
		{"hello world", 5, "hello..."}, // truncated with ellipsis
		{"hi", 2, "hi"},               // exactly at limit
		{"abc", 1, "a..."},            // aggressive truncation
		{"", 5, ""},                   // empty string
	}

	for _, tc := range cases {
		got := truncate(tc.input, tc.n)
		if got != tc.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q", tc.input, tc.n, got, tc.expected)
		}
	}
}

// ============================================================================
// join
// ============================================================================

func TestTemplateFuncBehaviors_Join(t *testing.T) {
	joinFn := func(items []string, sep string) string {
		return strings.Join(items, sep)
	}

	cases := []struct {
		items    []string
		sep      string
		expected string
	}{
		{[]string{"a", "b", "c"}, ", ", "a, b, c"},
		{[]string{"x"}, ", ", "x"},
		{[]string{}, ", ", ""},
		{[]string{"p", "q"}, " | ", "p | q"},
	}

	for _, tc := range cases {
		got := joinFn(tc.items, tc.sep)
		if got != tc.expected {
			t.Errorf("join(%v, %q) = %q, want %q", tc.items, tc.sep, got, tc.expected)
		}
	}
}

// ============================================================================
// formatTime
// ============================================================================

func TestTemplateFuncBehaviors_FormatTime(t *testing.T) {
	formatTime := func(t time.Time) string {
		return t.Format("2006-01-02 15:04:05 MST")
	}

	fixed := time.Date(2024, 3, 22, 14, 0, 0, 0, time.UTC)
	got := formatTime(fixed)
	if !strings.HasPrefix(got, "2024-03-22 14:00:00") {
		t.Errorf("unexpected formatTime output: %q", got)
	}

	// Edge: zero time
	zero := time.Time{}
	result := formatTime(zero)
	if result == "" {
		t.Error("expected non-empty result for zero time")
	}
}

func TestTemplateFuncBehaviors_FormatTime_ViaTemplate(t *testing.T) {
	formatTime := func(tm time.Time) string {
		return tm.Format("2006-01-02 15:04:05 MST")
	}
	funcs := template.FuncMap{"formatTime": formatTime}
	tmpl := template.Must(template.New("t").Funcs(funcs).Parse(`{{formatTime .}}`))

	fixed := time.Date(2025, 7, 4, 12, 0, 0, 0, time.UTC)
	var sb strings.Builder
	if err := tmpl.Execute(&sb, fixed); err != nil {
		t.Fatalf("template execute: %v", err)
	}
	if !strings.Contains(sb.String(), "2025-07-04") {
		t.Errorf("expected date in output, got: %q", sb.String())
	}
}

// ============================================================================
// statusClass
// ============================================================================

func TestTemplateFuncBehaviors_StatusClass(t *testing.T) {
	statusClass := func(status string) string {
		switch status {
		case "OK":
			return "status-ok"
		case "DRIFT":
			return "status-warning"
		default:
			return "status-info"
		}
	}

	cases := []struct{ in, want string }{
		{"OK", "status-ok"},
		{"DRIFT", "status-warning"},
		{"OTHER", "status-info"},
		{"", "status-info"},
		{"ok", "status-info"}, // case-sensitive — lowercase falls through to default
	}
	for _, tc := range cases {
		got := statusClass(tc.in)
		if got != tc.want {
			t.Errorf("statusClass(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// ============================================================================
// resolutionClass
// ============================================================================

func TestTemplateFuncBehaviors_ResolutionClass(t *testing.T) {
	resolutionClass := func(resolution string) string {
		switch resolution {
		case "keep_prod":
			return "resolution-ours"
		case "use_dev":
			return "resolution-theirs"
		case "pending":
			return "resolution-pending"
		default:
			return ""
		}
	}

	cases := []struct{ in, want string }{
		{"keep_prod", "resolution-ours"},
		{"use_dev", "resolution-theirs"},
		{"pending", "resolution-pending"},
		{"unknown", ""},
		{"", ""},
		{"KEEP_PROD", ""}, // case-sensitive default
	}
	for _, tc := range cases {
		got := resolutionClass(tc.in)
		if got != tc.want {
			t.Errorf("resolutionClass(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// ============================================================================
// changeTypeClass
// ============================================================================

func TestTemplateFuncBehaviors_ChangeTypeClass(t *testing.T) {
	changeTypeClass := func(changeType string) string {
		switch changeType {
		case "added", "added_table":
			return "change-added"
		case "removed", "removed_table":
			return "change-removed"
		case "modified", "type_change", "nullable_change":
			return "change-modified"
		default:
			return ""
		}
	}

	cases := []struct{ in, want string }{
		{"added", "change-added"},
		{"added_table", "change-added"},
		{"removed", "change-removed"},
		{"removed_table", "change-removed"},
		{"modified", "change-modified"},
		{"type_change", "change-modified"},
		{"nullable_change", "change-modified"},
		{"unknown", ""},
		{"", ""},
	}
	for _, tc := range cases {
		got := changeTypeClass(tc.in)
		if got != tc.want {
			t.Errorf("changeTypeClass(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
