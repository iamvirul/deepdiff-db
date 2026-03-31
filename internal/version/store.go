package version

import (
	"bytes"
	"compress/zlib"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Init creates the .deepdiffdb repository structure in dir.
// It is idempotent — calling it on an existing repo is a no-op.
// HEAD is written as a symbolic ref pointing to refs/heads/main.
func Init(dir string) error {
	objectsPath := filepath.Join(dir, RepoDirName, objectsDirName)
	if err := os.MkdirAll(objectsPath, 0o750); err != nil {
		return fmt.Errorf("create version repo: %w", err)
	}
	headsPath := filepath.Join(dir, RepoDirName, refsDirName, headsDirName)
	if err := os.MkdirAll(headsPath, 0o750); err != nil {
		return fmt.Errorf("create refs/heads: %w", err)
	}
	headPath := filepath.Join(dir, RepoDirName, headFileName)
	if _, err := os.Stat(headPath); os.IsNotExist(err) {
		symref := symbolicRefPrefix + refsDirName + "/" + headsDirName + "/" + defaultBranch
		if err := os.WriteFile(headPath, []byte(symref), 0o600); err != nil {
			return fmt.Errorf("create HEAD: %w", err)
		}
	}
	return nil
}

// IsInitialized reports whether dir contains an initialized version repo.
func IsInitialized(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, RepoDirName, headFileName))
	return err == nil
}

// SaveCommit serializes c as JSON, zlib-compresses the result, and writes it
// to objects/<hash[:2]>/<hash[2:]> — the same fanout layout Git uses.
// If HEAD is a symbolic ref, the referenced branch tip is advanced to c.Hash.
// Otherwise HEAD is updated directly (detached HEAD mode).
func SaveCommit(dir string, c *Commit) error {
	if err := validateHex(c.Hash); err != nil {
		return fmt.Errorf("invalid commit hash: %w", err)
	}
	raw, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal commit: %w", err)
	}

	compressed, err := zlibCompress(raw)
	if err != nil {
		return fmt.Errorf("compress commit: %w", err)
	}

	// Git-style fanout: first two hex chars are the directory name.
	objDir := filepath.Join(dir, RepoDirName, objectsDirName, c.Hash[:2])
	if err := os.MkdirAll(objDir, 0o750); err != nil {
		return fmt.Errorf("create object dir: %w", err)
	}

	objPath := filepath.Join(objDir, c.Hash[2:])
	if err := os.WriteFile(objPath, compressed, 0o400); err != nil { // #nosec G306 -- objects are intentionally read-only
		return fmt.Errorf("write commit object: %w", err)
	}

	// Advance the current branch tip (or HEAD directly for detached HEAD).
	return advanceHEAD(dir, c.Hash)
}

// LoadCommit reads, decompresses, and deserializes the commit identified by hash.
func LoadCommit(dir, hash string) (*Commit, error) {
	if len(hash) < 3 {
		return nil, fmt.Errorf("hash too short: %q", hash)
	}
	if err := validateHex(hash); err != nil {
		return nil, fmt.Errorf("invalid hash: %w", err)
	}

	objPath := filepath.Join(dir, RepoDirName, objectsDirName, hash[:2], hash[2:])
	compressed, err := os.ReadFile(objPath) // #nosec G703 -- path constructed from hex-validated hash
	if err != nil {
		return nil, fmt.Errorf("read commit %s: %w", shortHash(hash), err)
	}

	raw, err := zlibDecompress(compressed)
	if err != nil {
		return nil, fmt.Errorf("decompress commit %s: %w", shortHash(hash), err)
	}

	var c Commit
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("parse commit %s: %w", shortHash(hash), err)
	}
	return &c, nil
}

// ReadHEAD resolves HEAD and returns the current commit hash.
// Returns "" when the repo has no commits yet.
func ReadHEAD(dir string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(dir, RepoDirName, headFileName)) //nolint:gosec
	if err != nil {
		return "", fmt.Errorf("read HEAD: %w", err)
	}
	content := strings.TrimSpace(string(raw))
	if strings.HasPrefix(content, symbolicRefPrefix) {
		ref := strings.TrimPrefix(content, symbolicRefPrefix)
		return readRef(dir, ref)
	}
	return content, nil
}

// CurrentBranch returns the branch name HEAD points to, or "" for detached HEAD.
func CurrentBranch(dir string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(dir, RepoDirName, headFileName)) //nolint:gosec
	if err != nil {
		return "", fmt.Errorf("read HEAD: %w", err)
	}
	content := strings.TrimSpace(string(raw))
	if !strings.HasPrefix(content, symbolicRefPrefix) {
		return "", nil // detached HEAD
	}
	ref := strings.TrimPrefix(content, symbolicRefPrefix)
	prefix := refsDirName + "/" + headsDirName + "/"
	return strings.TrimPrefix(ref, prefix), nil
}

// writeRef writes hash to the ref file at refs/<refPath> inside the repo.
// refPath is always constructed from validated internal constants + branch names.
func writeRef(dir, refPath, hash string) error {
	path := filepath.Join(dir, RepoDirName, refPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil { // #nosec G703 -- path constructed from internal constants
		return fmt.Errorf("create ref dir: %w", err)
	}
	if err := os.WriteFile(path, []byte(hash), 0o600); err != nil { // #nosec G703 -- path constructed from internal constants
		return fmt.Errorf("write ref %s: %w", refPath, err)
	}
	return nil
}

// readRef reads the hash stored in a ref file. Returns "" if the file does not
// exist yet (branch exists but has no commits).
// refPath is always constructed from validated internal constants + branch names.
func readRef(dir, refPath string) (string, error) {
	path := filepath.Join(dir, RepoDirName, refPath)
	data, err := os.ReadFile(path) // #nosec G703 -- path constructed from internal constants
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read ref %s: %w", refPath, err)
	}
	return strings.TrimSpace(string(data)), nil
}

// writeHEAD writes a raw value (symbolic ref or hash) into HEAD.
func writeHEAD(dir, value string) error {
	path := filepath.Join(dir, RepoDirName, headFileName)
	if err := os.WriteFile(path, []byte(value), 0o600); err != nil {
		return fmt.Errorf("write HEAD: %w", err)
	}
	return nil
}

// advanceHEAD advances the current branch tip to hash, or writes hash directly
// if HEAD is in detached state.
func advanceHEAD(dir, hash string) error {
	raw, err := os.ReadFile(filepath.Join(dir, RepoDirName, headFileName)) //nolint:gosec
	if err != nil {
		return fmt.Errorf("read HEAD: %w", err)
	}
	content := strings.TrimSpace(string(raw))
	if strings.HasPrefix(content, symbolicRefPrefix) {
		ref := strings.TrimPrefix(content, symbolicRefPrefix)
		return writeRef(dir, ref, hash)
	}
	// Detached HEAD: write hash directly.
	return writeHEAD(dir, hash)
}

// ComputeHash derives a deterministic SHA-256 hash for c.
//
// Following Git's content-addressable model the hash is computed over the
// uncompressed serialized bytes of the commit (with Hash cleared), prefixed
// by a header identical in spirit to Git's object header:
//
//	"commit <size>\x00<json>"
//
// This means the hash is a function of the content, not the metadata alone.
func ComputeHash(c *Commit) (string, error) {
	// Clear hash before serializing so it is not included in the content.
	saved := c.Hash
	c.Hash = ""
	raw, err := json.Marshal(c)
	c.Hash = saved // restore regardless of error
	if err != nil {
		return "", fmt.Errorf("marshal commit for hashing: %w", err)
	}

	header := fmt.Sprintf("commit %d\x00", len(raw))

	h := sha256.New()
	h.Write([]byte(header))
	h.Write(raw)
	hash := fmt.Sprintf("%x", h.Sum(nil))
	c.Hash = hash
	return hash, nil
}

// zlibCompress returns the zlib-compressed form of src.
func zlibCompress(src []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(src); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// zlibDecompress decompresses src and returns the raw bytes.
func zlibDecompress(src []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(src))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func shortHash(h string) string {
	if len(h) >= 8 {
		return h[:8]
	}
	return h
}

// validateHex returns an error if s contains any character outside [0-9a-f].
// This is called before using a hash as a filesystem path component to prevent
// path traversal via crafted hash strings.
func validateHex(s string) error {
	if s == "" {
		return fmt.Errorf("empty hash")
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return fmt.Errorf("invalid character %q in hash", c)
		}
	}
	return nil
}
