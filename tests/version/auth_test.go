package version_test

import (
	"os"
	"path/filepath"
	"testing"

	vcs "github.com/iamvirul/deepdiff-db/internal/version"
)

// TestSaveAndLoadIdentity verifies the round-trip of SaveIdentity / LoadIdentity.
func TestSaveAndLoadIdentity(t *testing.T) {
	dir := t.TempDir()
	// Repo must be initialised first (SaveIdentity writes into .deepdiffdb/).
	if err := vcs.Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}

	const want = "iamvirul"
	if err := vcs.SaveIdentity(dir, want); err != nil {
		t.Fatalf("SaveIdentity: %v", err)
	}

	got, err := vcs.LoadIdentity(dir)
	if err != nil {
		t.Fatalf("LoadIdentity: %v", err)
	}
	if got != want {
		t.Errorf("LoadIdentity = %q, want %q", got, want)
	}
}

// TestLoadIdentityMissingFile returns empty string (not an error) when config does not exist.
func TestLoadIdentityMissingFile(t *testing.T) {
	dir := t.TempDir()
	if err := vcs.Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}

	username, err := vcs.LoadIdentity(dir)
	if err != nil {
		t.Fatalf("LoadIdentity unexpectedly returned error for missing config: %v", err)
	}
	if username != "" {
		t.Errorf("LoadIdentity = %q, want empty string for missing config", username)
	}
}

// TestSaveIdentityFilePermissions verifies the config file is written with 0600 permissions.
func TestSaveIdentityFilePermissions(t *testing.T) {
	dir := t.TempDir()
	if err := vcs.Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := vcs.SaveIdentity(dir, "testuser"); err != nil {
		t.Fatalf("SaveIdentity: %v", err)
	}

	info, err := os.Stat(filepath.Join(dir, vcs.RepoDirName, "config"))
	if err != nil {
		t.Fatalf("Stat config: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("config file permissions = %o, want 0600", perm)
	}
}

// TestSaveIdentityOverwrite verifies that calling SaveIdentity twice updates the stored value.
func TestSaveIdentityOverwrite(t *testing.T) {
	dir := t.TempDir()
	if err := vcs.Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}

	if err := vcs.SaveIdentity(dir, "old-user"); err != nil {
		t.Fatalf("SaveIdentity (first): %v", err)
	}
	if err := vcs.SaveIdentity(dir, "new-user"); err != nil {
		t.Fatalf("SaveIdentity (second): %v", err)
	}

	got, err := vcs.LoadIdentity(dir)
	if err != nil {
		t.Fatalf("LoadIdentity: %v", err)
	}
	if got != "new-user" {
		t.Errorf("LoadIdentity = %q, want %q", got, "new-user")
	}
}

// TestLoadIdentityCorruptFile returns an error on malformed JSON.
func TestLoadIdentityCorruptFile(t *testing.T) {
	dir := t.TempDir()
	if err := vcs.Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}

	configPath := filepath.Join(dir, vcs.RepoDirName, "config")
	if err := os.WriteFile(configPath, []byte("not valid json"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := vcs.LoadIdentity(dir)
	if err == nil {
		t.Error("LoadIdentity expected error for corrupt config, got nil")
	}
}

// TestResolveClientID prefers the environment variable over the build-time default.
func TestResolveClientID(t *testing.T) {
	const envKey = "DEEPDIFFDB_GITHUB_CLIENT_ID"

	// Save and restore original value
	original := os.Getenv(envKey)
	defer func() {
		if original == "" {
			os.Unsetenv(envKey) //nolint:errcheck
		} else {
			os.Setenv(envKey, original) //nolint:errcheck
		}
	}()

	// When env var is set it should take precedence
	os.Setenv(envKey, "env-client-id") //nolint:errcheck
	if got := vcs.ResolveClientID(); got != "env-client-id" {
		t.Errorf("ResolveClientID with env = %q, want %q", got, "env-client-id")
	}

	// When env var is cleared, falls back to GitHubClientID (may be empty in CI)
	os.Unsetenv(envKey) //nolint:errcheck
	// We just assert it doesn't panic — the value depends on build-time injection
	_ = vcs.ResolveClientID()
}
