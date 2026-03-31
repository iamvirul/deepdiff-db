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
)

// Init creates the .deepdiffdb repository structure in dir.
// It is idempotent — calling it on an existing repo is a no-op.
func Init(dir string) error {
	objectsPath := filepath.Join(dir, RepoDirName, objectsDirName)
	if err := os.MkdirAll(objectsPath, 0o750); err != nil {
		return fmt.Errorf("create version repo: %w", err)
	}
	headPath := filepath.Join(dir, RepoDirName, headFileName)
	if _, err := os.Stat(headPath); os.IsNotExist(err) {
		if err := os.WriteFile(headPath, []byte(""), 0o640); err != nil {
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
// to objects/<hash[:2]>/<hash[2:]> — the same fanout layout Git uses to avoid
// directory entry limits on large object stores. HEAD is updated to c.Hash.
func SaveCommit(dir string, c *Commit) error {
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
	if err := os.WriteFile(objPath, compressed, 0o440); err != nil {
		return fmt.Errorf("write commit object: %w", err)
	}

	return writeHEAD(dir, c.Hash)
}

// LoadCommit reads, decompresses, and deserializes the commit identified by hash.
func LoadCommit(dir, hash string) (*Commit, error) {
	if len(hash) < 3 {
		return nil, fmt.Errorf("hash too short: %q", hash)
	}

	objPath := filepath.Join(dir, RepoDirName, objectsDirName, hash[:2], hash[2:])
	compressed, err := os.ReadFile(objPath) //nolint:gosec
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

// ReadHEAD returns the hash stored in HEAD, or "" when the repo is empty.
func ReadHEAD(dir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(dir, RepoDirName, headFileName)) //nolint:gosec
	if err != nil {
		return "", fmt.Errorf("read HEAD: %w", err)
	}
	return string(data), nil
}

func writeHEAD(dir, hash string) error {
	path := filepath.Join(dir, RepoDirName, headFileName)
	if err := os.WriteFile(path, []byte(hash), 0o640); err != nil {
		return fmt.Errorf("write HEAD: %w", err)
	}
	return nil
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
