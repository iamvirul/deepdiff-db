package version

import (
	"fmt"
	"io"
	"strings"
	"time"
)

type treeNode struct {
	hash      string
	commit    *Commit
	branchTip []string // branch names whose tip == this commit
}

// RenderTree prints an ASCII commit graph to w, showing all branches.
// The output mimics `git log --oneline --graph --all --decorate`.
func RenderTree(dir string, w io.Writer) error {
	if !IsInitialized(dir) {
		return fmt.Errorf("no version repository found; run 'deepdiffdb version init' first")
	}

	branches, err := ListBranches(dir)
	if err != nil {
		return err
	}

	// Walk every branch tip back to root; deduplicate by hash.
	seen := map[string]bool{}
	var ordered []string
	for _, b := range branches {
		if b.Tip == "" {
			continue
		}
		hash := b.Tip
		for hash != "" && !seen[hash] {
			seen[hash] = true
			ordered = append(ordered, hash)
			c, err := LoadCommit(dir, hash)
			if err != nil {
				break
			}
			hash = c.Parent
		}
	}

	if len(ordered) == 0 {
		fmt.Fprintln(w, "(no commits yet)")
		return nil
	}

	// Build hash → branch names map.
	tipMap := map[string][]string{}
	for _, b := range branches {
		if b.Tip != "" {
			tipMap[b.Tip] = append(tipMap[b.Tip], b.Name)
		}
	}

	// Load all nodes.
	nodes := make([]treeNode, 0, len(ordered))
	for _, hash := range ordered {
		c, err := LoadCommit(dir, hash)
		if err != nil {
			return fmt.Errorf("load commit %s: %w", shortHash(hash), err)
		}
		nodes = append(nodes, treeNode{hash: hash, commit: c, branchTip: tipMap[hash]})
	}
	sortTreeNodes(nodes)

	// Assign each branch a lane (column).
	laneMap := map[string]int{}
	nextLane := 0
	for _, b := range branches {
		if b.Tip != "" {
			if _, ok := laneMap[b.Name]; !ok {
				laneMap[b.Name] = nextLane
				nextLane++
			}
		}
	}
	totalLanes := nextLane
	if totalLanes == 0 {
		totalLanes = 1
	}

	currentBranch, _ := CurrentBranch(dir)
	headHash, _ := ReadHEAD(dir)

	nodeLane := func(n treeNode) int {
		for _, br := range n.branchTip {
			if l, ok := laneMap[br]; ok {
				return l
			}
		}
		for _, b := range branches {
			if b.Tip == "" {
				continue
			}
			if reachableFrom(dir, b.Tip, n.hash) {
				if l, ok := laneMap[b.Name]; ok {
					return l
				}
			}
		}
		return 0
	}

	for _, n := range nodes {
		lane := nodeLane(n)

		var prefix strings.Builder
		for col := 0; col < totalLanes; col++ {
			if col == lane {
				prefix.WriteString("* ")
			} else {
				prefix.WriteString("| ")
			}
		}

		decoration := ""
		if len(n.branchTip) > 0 {
			parts := make([]string, 0, len(n.branchTip))
			for _, br := range n.branchTip {
				label := br
				if br == currentBranch {
					label = "HEAD -> " + br
				}
				parts = append(parts, label)
			}
			decoration = " (" + strings.Join(parts, ", ") + ")"
		} else if n.hash == headHash {
			decoration = " (HEAD)"
		}

		ts := n.commit.Timestamp.Format(time.DateOnly)
		fmt.Fprintf(w, "%s%s%s  %s  %s\n",
			prefix.String(),
			shortHash(n.hash),
			decoration,
			ts,
			n.commit.Message,
		)
	}
	return nil
}

// sortTreeNodes sorts nodes newest-first (insertion sort — small N expected).
func sortTreeNodes(nodes []treeNode) {
	for i := 1; i < len(nodes); i++ {
		for j := i; j > 0; j-- {
			if nodes[j].commit.Timestamp.After(nodes[j-1].commit.Timestamp) {
				nodes[j], nodes[j-1] = nodes[j-1], nodes[j]
			} else {
				break
			}
		}
	}
}

// reachableFrom reports whether targetHash is an ancestor of (or equal to) tipHash.
func reachableFrom(dir, tipHash, targetHash string) bool {
	visited := map[string]bool{}
	hash := tipHash
	for hash != "" && !visited[hash] {
		if hash == targetHash {
			return true
		}
		visited[hash] = true
		c, err := LoadCommit(dir, hash)
		if err != nil {
			return false
		}
		hash = c.Parent
	}
	return false
}

