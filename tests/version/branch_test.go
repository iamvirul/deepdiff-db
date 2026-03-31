package version_test

import (
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/version"
)

// ---------------------------------------------------------------------------
// ListBranches
// ---------------------------------------------------------------------------

func TestListBranches_OnlyMainAfterInit(t *testing.T) {
	dir := initRepo(t)
	branches, err := version.ListBranches(dir)
	if err != nil {
		t.Fatalf("ListBranches: %v", err)
	}
	if len(branches) != 1 {
		t.Fatalf("len(branches) = %d; want 1", len(branches))
	}
	if branches[0].Name != "main" {
		t.Errorf("branch name = %q; want %q", branches[0].Name, "main")
	}
	if !branches[0].Current {
		t.Error("main should be current")
	}
	if branches[0].Tip != "" {
		t.Errorf("tip = %q; want empty (no commits yet)", branches[0].Tip)
	}
}

func TestListBranches_TipUpdatesAfterCommit(t *testing.T) {
	dir := initRepo(t)
	c := makeCommit(t, dir, "initial", "")

	branches, err := version.ListBranches(dir)
	if err != nil {
		t.Fatalf("ListBranches: %v", err)
	}
	if branches[0].Tip != c.Hash {
		t.Errorf("tip = %q; want %q", branches[0].Tip, c.Hash)
	}
}

func TestListBranches_ShowsAllBranches(t *testing.T) {
	dir := initRepo(t)
	makeCommit(t, dir, "initial", "")

	if err := version.CreateBranch(dir, "feature", ""); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}

	branches, err := version.ListBranches(dir)
	if err != nil {
		t.Fatalf("ListBranches: %v", err)
	}

	names := map[string]bool{}
	for _, b := range branches {
		names[b.Name] = true
	}
	if !names["main"] {
		t.Error("main not in branch list")
	}
	if !names["feature"] {
		t.Error("feature not in branch list")
	}
}

func TestListBranches_MarksCurrentBranch(t *testing.T) {
	dir := initRepo(t)
	makeCommit(t, dir, "initial", "")
	if err := version.CreateBranch(dir, "feature", ""); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}
	if err := version.Checkout(dir, "feature"); err != nil {
		t.Fatalf("Checkout: %v", err)
	}

	branches, err := version.ListBranches(dir)
	if err != nil {
		t.Fatalf("ListBranches: %v", err)
	}
	for _, b := range branches {
		if b.Name == "feature" && !b.Current {
			t.Error("feature should be current after checkout")
		}
		if b.Name == "main" && b.Current {
			t.Error("main should not be current after checkout to feature")
		}
	}
}

// ---------------------------------------------------------------------------
// CreateBranch
// ---------------------------------------------------------------------------

func TestCreateBranch_CreatesAtCurrentHEAD(t *testing.T) {
	dir := initRepo(t)
	c := makeCommit(t, dir, "initial", "")

	if err := version.CreateBranch(dir, "feature", ""); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}

	tip, err := version.ReadBranchTip(dir, "feature")
	if err != nil {
		t.Fatalf("ReadBranchTip: %v", err)
	}
	if tip != c.Hash {
		t.Errorf("feature tip = %q; want %q", tip, c.Hash)
	}
}

func TestCreateBranch_CreatesAtSpecificHash(t *testing.T) {
	dir := initRepo(t)
	c1 := makeCommit(t, dir, "first", "")
	makeCommit(t, dir, "second", c1.Hash) // advances main

	if err := version.CreateBranch(dir, "hotfix", c1.Hash); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}

	tip, err := version.ReadBranchTip(dir, "hotfix")
	if err != nil {
		t.Fatalf("ReadBranchTip: %v", err)
	}
	if tip != c1.Hash {
		t.Errorf("hotfix tip = %q; want %q", tip, c1.Hash)
	}
}

func TestCreateBranch_ErrorOnDuplicate(t *testing.T) {
	dir := initRepo(t)
	makeCommit(t, dir, "initial", "")

	if err := version.CreateBranch(dir, "feature", ""); err != nil {
		t.Fatalf("first CreateBranch: %v", err)
	}
	if err := version.CreateBranch(dir, "feature", ""); err == nil {
		t.Error("expected error creating duplicate branch")
	}
}

func TestCreateBranch_ErrorOnEmptyName(t *testing.T) {
	dir := initRepo(t)
	if err := version.CreateBranch(dir, "", ""); err == nil {
		t.Error("expected error for empty branch name")
	}
}

func TestCreateBranch_ErrorOnInvalidName(t *testing.T) {
	dir := initRepo(t)
	makeCommit(t, dir, "initial", "")

	for _, name := range []string{"with space", "with/slash", "with\ttab"} {
		if err := version.CreateBranch(dir, name, ""); err == nil {
			t.Errorf("expected error for invalid name %q", name)
		}
	}
}

// ---------------------------------------------------------------------------
// Checkout
// ---------------------------------------------------------------------------

func TestCheckout_SwitchesToBranch(t *testing.T) {
	dir := initRepo(t)
	makeCommit(t, dir, "initial", "")
	if err := version.CreateBranch(dir, "feature", ""); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}

	if err := version.Checkout(dir, "feature"); err != nil {
		t.Fatalf("Checkout: %v", err)
	}

	branch, err := version.CurrentBranch(dir)
	if err != nil {
		t.Fatalf("CurrentBranch: %v", err)
	}
	if branch != "feature" {
		t.Errorf("current branch = %q; want %q", branch, "feature")
	}
}

func TestCheckout_ErrorOnNonExistentBranch(t *testing.T) {
	dir := initRepo(t)
	if err := version.Checkout(dir, "nonexistent"); err == nil {
		t.Error("expected error checking out nonexistent branch")
	}
}

func TestCheckout_CommitsGoToCheckedOutBranch(t *testing.T) {
	dir := initRepo(t)
	c1 := makeCommit(t, dir, "initial", "")

	if err := version.CreateBranch(dir, "feature", ""); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}
	if err := version.Checkout(dir, "feature"); err != nil {
		t.Fatalf("Checkout: %v", err)
	}

	c2 := makeCommit(t, dir, "feature work", c1.Hash)

	// feature tip must be c2
	featureTip, _ := version.ReadBranchTip(dir, "feature")
	if featureTip != c2.Hash {
		t.Errorf("feature tip = %q; want %q", featureTip, c2.Hash)
	}
	// main tip must still be c1
	mainTip, _ := version.ReadBranchTip(dir, "main")
	if mainTip != c1.Hash {
		t.Errorf("main tip = %q; want %q (should not have moved)", mainTip, c1.Hash)
	}
}

func TestCheckout_SwitchBackToMain(t *testing.T) {
	dir := initRepo(t)
	makeCommit(t, dir, "initial", "")
	if err := version.CreateBranch(dir, "feature", ""); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}
	if err := version.Checkout(dir, "feature"); err != nil {
		t.Fatalf("Checkout feature: %v", err)
	}
	if err := version.Checkout(dir, "main"); err != nil {
		t.Fatalf("Checkout main: %v", err)
	}
	branch, _ := version.CurrentBranch(dir)
	if branch != "main" {
		t.Errorf("current branch = %q; want %q", branch, "main")
	}
}

// ---------------------------------------------------------------------------
// ReadBranchTip
// ---------------------------------------------------------------------------

func TestReadBranchTip_EmptyForNonexistentBranch(t *testing.T) {
	dir := initRepo(t)
	tip, err := version.ReadBranchTip(dir, "ghost")
	if err != nil {
		t.Fatalf("ReadBranchTip: %v", err)
	}
	if tip != "" {
		t.Errorf("tip = %q; want empty", tip)
	}
}

func TestReadBranchTip_TracksLatestCommit(t *testing.T) {
	dir := initRepo(t)
	c1 := makeCommit(t, dir, "first", "")
	c2 := makeCommit(t, dir, "second", c1.Hash)

	tip, _ := version.ReadBranchTip(dir, "main")
	if tip != c2.Hash {
		t.Errorf("main tip = %q; want %q", tip, c2.Hash)
	}
}
