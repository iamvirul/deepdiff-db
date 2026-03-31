package version

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// BranchInfo describes a single branch.
type BranchInfo struct {
	Name    string // branch name
	Tip     string // current tip commit hash (empty if no commits yet)
	Current bool   // true when this branch is checked out
}

// ListBranches returns all branches in the repo, marking the current one.
func ListBranches(dir string) ([]BranchInfo, error) {
	headsPath := filepath.Join(dir, RepoDirName, refsDirName, headsDirName)
	entries, err := os.ReadDir(headsPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list branches: %w", err)
	}

	current, err := CurrentBranch(dir)
	if err != nil {
		return nil, err
	}

	branches := make([]BranchInfo, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		refPath := refsDirName + "/" + headsDirName + "/" + name
		tip, err := readRef(dir, refPath)
		if err != nil {
			return nil, err
		}
		branches = append(branches, BranchInfo{
			Name:    name,
			Tip:     tip,
			Current: name == current,
		})
	}

	// If HEAD is a symbolic ref pointing to a branch that hasn't been committed
	// to yet (ref file doesn't exist), include it anyway.
	if current != "" {
		found := false
		for _, b := range branches {
			if b.Name == current {
				found = true
				break
			}
		}
		if !found {
			branches = append([]BranchInfo{{Name: current, Current: true}}, branches...)
		}
	}

	return branches, nil
}

// CreateBranch creates a new branch pointing to the commit identified by fromHash.
// If fromHash is empty, the current HEAD commit is used. Returns an error if the
// branch already exists.
func CreateBranch(dir, name, fromHash string) error {
	if name == "" {
		return fmt.Errorf("branch name cannot be empty")
	}
	if strings.ContainsAny(name, " \t\n/\\") {
		return fmt.Errorf("invalid branch name %q: must not contain spaces or slashes", name)
	}

	refPath := refsDirName + "/" + headsDirName + "/" + name
	refFile := filepath.Join(dir, RepoDirName, refPath)
	if _, err := os.Stat(refFile); err == nil {
		return fmt.Errorf("branch %q already exists", name)
	}

	var err error
	if fromHash == "" {
		fromHash, err = ReadHEAD(dir)
		if err != nil {
			return err
		}
	}
	return writeRef(dir, refPath, fromHash)
}

// Checkout switches HEAD to point to the named branch (symbolic ref mode).
// Returns an error if the branch does not exist.
func Checkout(dir, name string) error {
	refPath := refsDirName + "/" + headsDirName + "/" + name
	headsDir := filepath.Join(dir, RepoDirName, refsDirName, headsDirName)
	entries, err := os.ReadDir(headsDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("list branches: %w", err)
	}
	exists := false
	for _, e := range entries {
		if !e.IsDir() && e.Name() == name {
			exists = true
			break
		}
	}
	// Also treat a branch that has no commits yet (ref file absent) as valid
	// only if we're switching back to the default branch created at init.
	if !exists {
		current, _ := CurrentBranch(dir)
		if current == name {
			return nil // already on this branch
		}
		// Check if this is the default branch that was never committed to.
		// Init writes the HEAD symbolic ref but not the ref file.
		if name == defaultBranch {
			// Verify objects dir exists as a proxy for init having run.
			if !IsInitialized(dir) {
				return fmt.Errorf("repo not initialized")
			}
		} else {
			return fmt.Errorf("branch %q not found; create it with 'version branch %s'", name, name)
		}
	}

	symref := symbolicRefPrefix + refPath
	return writeHEAD(dir, symref)
}

// ReadBranchTip returns the tip commit hash for the named branch.
// Returns "" if the branch has no commits yet.
func ReadBranchTip(dir, name string) (string, error) {
	return readRef(dir, refsDirName+"/"+headsDirName+"/"+name)
}
