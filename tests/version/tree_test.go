package version_test

import (
	"strings"
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/version"
)

// ---------------------------------------------------------------------------
// RenderTree
// ---------------------------------------------------------------------------

func TestRenderTree_NoCommitsYet(t *testing.T) {
	dir := initRepo(t)
	var buf strings.Builder
	if err := version.RenderTree(dir, &buf); err != nil {
		t.Fatalf("RenderTree: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "no commits") {
		t.Errorf("expected 'no commits' message, got %q", out)
	}
}

func TestRenderTree_ErrorOnUninitializedRepo(t *testing.T) {
	dir := t.TempDir() // no Init
	var buf strings.Builder
	if err := version.RenderTree(dir, &buf); err == nil {
		t.Error("expected error for uninitialized repo")
	}
}

func TestRenderTree_SingleCommit(t *testing.T) {
	dir := initRepo(t)
	c := makeCommit(t, dir, "initial commit", "")

	var buf strings.Builder
	if err := version.RenderTree(dir, &buf); err != nil {
		t.Fatalf("RenderTree: %v", err)
	}
	out := buf.String()

	// Must contain short hash
	if !strings.Contains(out, c.Hash[:8]) {
		t.Errorf("output missing hash %s\n%s", c.Hash[:8], out)
	}
	// Must contain message
	if !strings.Contains(out, "initial commit") {
		t.Errorf("output missing commit message\n%s", out)
	}
	// Must contain HEAD decoration
	if !strings.Contains(out, "HEAD") {
		t.Errorf("output missing HEAD decoration\n%s", out)
	}
	// Must contain branch name
	if !strings.Contains(out, "main") {
		t.Errorf("output missing branch name\n%s", out)
	}
	// Must contain graph marker
	if !strings.Contains(out, "*") {
		t.Errorf("output missing '*' graph marker\n%s", out)
	}
}

func TestRenderTree_LinearChainOrderedNewestFirst(t *testing.T) {
	dir := initRepo(t)
	c1 := makeCommit(t, dir, "first", "")
	c2 := makeCommit(t, dir, "second", c1.Hash)
	c3 := makeCommit(t, dir, "third", c2.Hash)

	var buf strings.Builder
	if err := version.RenderTree(dir, &buf); err != nil {
		t.Fatalf("RenderTree: %v", err)
	}
	out := buf.String()

	// All three hashes must appear
	for _, c := range []*version.Commit{c1, c2, c3} {
		if !strings.Contains(out, c.Hash[:8]) {
			t.Errorf("output missing hash %s\n%s", c.Hash[:8], out)
		}
	}

	// newest (c3) must appear before oldest (c1)
	posC3 := strings.Index(out, c3.Hash[:8])
	posC1 := strings.Index(out, c1.Hash[:8])
	if posC3 >= posC1 {
		t.Errorf("expected c3 before c1 in output\n%s", out)
	}
}

func TestRenderTree_ShowsAllBranches(t *testing.T) {
	dir := initRepo(t)
	c1 := makeCommit(t, dir, "initial", "")

	// Create and populate feature branch.
	if err := version.CreateBranch(dir, "feature", ""); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}
	if err := version.Checkout(dir, "feature"); err != nil {
		t.Fatalf("Checkout: %v", err)
	}
	c2 := makeCommit(t, dir, "feature work", c1.Hash)

	// Switch back to main and add another commit.
	if err := version.Checkout(dir, "main"); err != nil {
		t.Fatalf("Checkout main: %v", err)
	}
	c3 := makeCommit(t, dir, "main work", c1.Hash)

	var buf strings.Builder
	if err := version.RenderTree(dir, &buf); err != nil {
		t.Fatalf("RenderTree: %v", err)
	}
	out := buf.String()

	for _, c := range []*version.Commit{c1, c2, c3} {
		if !strings.Contains(out, c.Hash[:8]) {
			t.Errorf("output missing hash %s\n%s", c.Hash[:8], out)
		}
	}
	if !strings.Contains(out, "main") {
		t.Errorf("output missing 'main' branch label\n%s", out)
	}
	if !strings.Contains(out, "feature") {
		t.Errorf("output missing 'feature' branch label\n%s", out)
	}
	if !strings.Contains(out, "HEAD") {
		t.Errorf("output missing HEAD marker\n%s", out)
	}
}

func TestRenderTree_MultipleBranchesHaveMultipleLanes(t *testing.T) {
	dir := initRepo(t)
	c1 := makeCommit(t, dir, "initial", "")

	if err := version.CreateBranch(dir, "feature", ""); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}
	if err := version.Checkout(dir, "feature"); err != nil {
		t.Fatalf("Checkout: %v", err)
	}
	makeCommit(t, dir, "feature work", c1.Hash)

	if err := version.Checkout(dir, "main"); err != nil {
		t.Fatalf("Checkout main: %v", err)
	}
	makeCommit(t, dir, "main work", c1.Hash)

	var buf strings.Builder
	if err := version.RenderTree(dir, &buf); err != nil {
		t.Fatalf("RenderTree: %v", err)
	}
	out := buf.String()

	// With 2 branches each line should have 2 lane columns (4 chars: "* |" or "| *")
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, line := range lines {
		// Each line starts with lane columns: "* " or "| " repeated totalLanes times
		if !strings.HasPrefix(line, "* ") && !strings.HasPrefix(line, "| ") {
			t.Errorf("line does not start with lane prefix: %q", line)
		}
	}
}

func TestRenderTree_HeadDecoration_PointsToCurrentBranch(t *testing.T) {
	dir := initRepo(t)
	c1 := makeCommit(t, dir, "initial", "")
	if err := version.CreateBranch(dir, "dev", ""); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}
	if err := version.Checkout(dir, "dev"); err != nil {
		t.Fatalf("Checkout: %v", err)
	}
	makeCommit(t, dir, "dev commit", c1.Hash)

	var buf strings.Builder
	if err := version.RenderTree(dir, &buf); err != nil {
		t.Fatalf("RenderTree: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "HEAD -> dev") {
		t.Errorf("expected 'HEAD -> dev' decoration\n%s", out)
	}
}
