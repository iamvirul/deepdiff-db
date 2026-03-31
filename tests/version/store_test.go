package version_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/iamvirul/deepdiff-db/internal/version"
)

// initRepo creates a temp directory, runs Init, and returns the dir path.
func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := version.Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}
	return dir
}

// makeCommit builds a minimal Commit, computes its hash, saves it, and returns it.
func makeCommit(t *testing.T, dir, message, parent string) *version.Commit {
	t.Helper()
	c := &version.Commit{
		Parent:    parent,
		Timestamp: time.Now().UTC(),
		Author:    "test",
		Message:   message,
		Driver:    "sqlite",
	}
	if _, err := version.ComputeHash(c); err != nil {
		t.Fatalf("ComputeHash: %v", err)
	}
	if err := version.SaveCommit(dir, c); err != nil {
		t.Fatalf("SaveCommit: %v", err)
	}
	return c
}

// ---------------------------------------------------------------------------
// Init
// ---------------------------------------------------------------------------

func TestInit_CreatesExpectedStructure(t *testing.T) {
	dir := t.TempDir()
	if err := version.Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// objects/ directory
	if _, err := os.Stat(filepath.Join(dir, ".deepdiffdb", "objects")); err != nil {
		t.Errorf("objects dir missing: %v", err)
	}
	// refs/heads/ directory
	if _, err := os.Stat(filepath.Join(dir, ".deepdiffdb", "refs", "heads")); err != nil {
		t.Errorf("refs/heads dir missing: %v", err)
	}
	// HEAD contains symbolic ref to main
	head, err := os.ReadFile(filepath.Join(dir, ".deepdiffdb", "HEAD"))
	if err != nil {
		t.Fatalf("read HEAD: %v", err)
	}
	if !strings.HasPrefix(string(head), "ref: refs/heads/main") {
		t.Errorf("HEAD = %q; want symbolic ref to main", string(head))
	}
}

func TestInit_Idempotent(t *testing.T) {
	dir := t.TempDir()
	if err := version.Init(dir); err != nil {
		t.Fatalf("first Init: %v", err)
	}
	if err := version.Init(dir); err != nil {
		t.Fatalf("second Init: %v", err)
	}
}

func TestIsInitialized_FalseOnEmptyDir(t *testing.T) {
	dir := t.TempDir()
	if version.IsInitialized(dir) {
		t.Error("expected false for empty dir")
	}
}

func TestIsInitialized_TrueAfterInit(t *testing.T) {
	dir := initRepo(t)
	if !version.IsInitialized(dir) {
		t.Error("expected true after Init")
	}
}

// ---------------------------------------------------------------------------
// ComputeHash
// ---------------------------------------------------------------------------

func TestComputeHash_Deterministic(t *testing.T) {
	c := &version.Commit{
		Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Author:    "alice",
		Message:   "initial",
		Driver:    "sqlite",
	}
	h1, err := version.ComputeHash(c)
	if err != nil {
		t.Fatalf("first hash: %v", err)
	}
	h2, err := version.ComputeHash(c)
	if err != nil {
		t.Fatalf("second hash: %v", err)
	}
	if h1 != h2 {
		t.Errorf("non-deterministic: %s != %s", h1, h2)
	}
}

func TestComputeHash_ChangesWithContent(t *testing.T) {
	base := &version.Commit{
		Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Author:    "alice",
		Message:   "one",
		Driver:    "sqlite",
	}
	h1, _ := version.ComputeHash(base)

	other := &version.Commit{
		Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Author:    "alice",
		Message:   "two",
		Driver:    "sqlite",
	}
	h2, _ := version.ComputeHash(other)

	if h1 == h2 {
		t.Error("different messages produced same hash")
	}
}

func TestComputeHash_SetsHashField(t *testing.T) {
	c := &version.Commit{
		Timestamp: time.Now().UTC(),
		Author:    "alice",
		Message:   "test",
		Driver:    "sqlite",
	}
	h, err := version.ComputeHash(c)
	if err != nil {
		t.Fatalf("ComputeHash: %v", err)
	}
	if c.Hash != h {
		t.Errorf("c.Hash = %q; want %q", c.Hash, h)
	}
	if len(h) != 64 {
		t.Errorf("hash length = %d; want 64", len(h))
	}
}

// ---------------------------------------------------------------------------
// SaveCommit / LoadCommit
// ---------------------------------------------------------------------------

func TestSaveAndLoadCommit_RoundTrip(t *testing.T) {
	dir := initRepo(t)
	c := makeCommit(t, dir, "initial", "")

	got, err := version.LoadCommit(dir, c.Hash)
	if err != nil {
		t.Fatalf("LoadCommit: %v", err)
	}
	if got.Hash != c.Hash {
		t.Errorf("hash mismatch: got %s want %s", got.Hash, c.Hash)
	}
	if got.Message != c.Message {
		t.Errorf("message mismatch: got %q want %q", got.Message, c.Message)
	}
}

func TestSaveCommit_FanoutLayout(t *testing.T) {
	dir := initRepo(t)
	c := makeCommit(t, dir, "fanout test", "")

	objPath := filepath.Join(dir, ".deepdiffdb", "objects", c.Hash[:2], c.Hash[2:])
	if _, err := os.Stat(objPath); err != nil {
		t.Errorf("expected object at %s: %v", objPath, err)
	}
}

func TestSaveCommit_AdvancesBranchTip(t *testing.T) {
	dir := initRepo(t)
	c := makeCommit(t, dir, "initial", "")

	tip, err := version.ReadBranchTip(dir, "main")
	if err != nil {
		t.Fatalf("ReadBranchTip: %v", err)
	}
	if tip != c.Hash {
		t.Errorf("branch tip = %q; want %q", tip, c.Hash)
	}
}

func TestLoadCommit_ErrorOnUnknownHash(t *testing.T) {
	dir := initRepo(t)
	_, err := version.LoadCommit(dir, strings.Repeat("a", 64))
	if err == nil {
		t.Error("expected error for unknown hash")
	}
}

// ---------------------------------------------------------------------------
// ReadHEAD / CurrentBranch — symbolic ref resolution
// ---------------------------------------------------------------------------

func TestReadHEAD_EmptyBeforeFirstCommit(t *testing.T) {
	dir := initRepo(t)
	hash, err := version.ReadHEAD(dir)
	if err != nil {
		t.Fatalf("ReadHEAD: %v", err)
	}
	if hash != "" {
		t.Errorf("expected empty HEAD before first commit, got %q", hash)
	}
}

func TestReadHEAD_ResolvesAfterCommit(t *testing.T) {
	dir := initRepo(t)
	c := makeCommit(t, dir, "first", "")

	hash, err := version.ReadHEAD(dir)
	if err != nil {
		t.Fatalf("ReadHEAD: %v", err)
	}
	if hash != c.Hash {
		t.Errorf("HEAD = %q; want %q", hash, c.Hash)
	}
}

func TestCurrentBranch_DefaultIsMain(t *testing.T) {
	dir := initRepo(t)
	branch, err := version.CurrentBranch(dir)
	if err != nil {
		t.Fatalf("CurrentBranch: %v", err)
	}
	if branch != "main" {
		t.Errorf("current branch = %q; want %q", branch, "main")
	}
}

func TestCurrentBranch_DetachedHead(t *testing.T) {
	dir := initRepo(t)
	c := makeCommit(t, dir, "first", "")

	// Simulate detached HEAD by writing raw hash to HEAD.
	headPath := filepath.Join(dir, ".deepdiffdb", "HEAD")
	if err := os.WriteFile(headPath, []byte(c.Hash), 0o640); err != nil {
		t.Fatalf("write HEAD: %v", err)
	}

	branch, err := version.CurrentBranch(dir)
	if err != nil {
		t.Fatalf("CurrentBranch: %v", err)
	}
	if branch != "" {
		t.Errorf("expected empty branch for detached HEAD, got %q", branch)
	}
}

// ---------------------------------------------------------------------------
// Commit chain (Parent links)
// ---------------------------------------------------------------------------

func TestCommitChain_ParentLinks(t *testing.T) {
	dir := initRepo(t)
	c1 := makeCommit(t, dir, "first", "")
	c2 := makeCommit(t, dir, "second", c1.Hash)
	c3 := makeCommit(t, dir, "third", c2.Hash)

	// Walk chain from HEAD.
	head, _ := version.ReadHEAD(dir)
	if head != c3.Hash {
		t.Errorf("HEAD = %s; want %s", head, c3.Hash)
	}

	loaded, _ := version.LoadCommit(dir, c3.Hash)
	if loaded.Parent != c2.Hash {
		t.Errorf("c3.Parent = %s; want %s", loaded.Parent, c2.Hash)
	}
	loaded2, _ := version.LoadCommit(dir, c2.Hash)
	if loaded2.Parent != c1.Hash {
		t.Errorf("c2.Parent = %s; want %s", loaded2.Parent, c1.Hash)
	}
	loaded1, _ := version.LoadCommit(dir, c1.Hash)
	if loaded1.Parent != "" {
		t.Errorf("c1.Parent = %q; want empty", loaded1.Parent)
	}
}
