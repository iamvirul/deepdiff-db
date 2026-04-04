package version

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// GitHubClientID is the GitHub OAuth App client ID used for device flow authentication.
// It can be overridden at build time:
//
//	go build -ldflags "-X github.com/iamvirul/deepdiff-db/internal/version.GitHubClientID=<id>"
//
// At runtime the DEEPDIFFDB_GITHUB_CLIENT_ID environment variable takes precedence.
//
// To register an OAuth App: https://github.com/settings/applications/new
//   - Application name: DeepDiff DB
//   - Homepage URL: https://github.com/iamvirul/deepdiff-db
//   - Enable: Device Authorization Flow
var GitHubClientID = ""

const (
	// configFileName is the file inside .deepdiffdb/ that stores verified identity.
	configFileName = "config"

	githubDeviceURL = "https://github.com/login/device/code"
	githubTokenURL  = "https://github.com/login/oauth/access_token" // #nosec G101 -- URL endpoint, not a credential
	githubUserURL   = "https://api.github.com/user"
)

// IdentityConfig is the schema for .deepdiffdb/config.
type IdentityConfig struct {
	GitHubUser string `json:"github_user"`
}

// LoadIdentity reads the verified GitHub username from .deepdiffdb/config.
// Returns an empty string (not an error) when the file does not exist — the caller
// treats that as "unauthenticated, fall back to --author flag".
func LoadIdentity(dir string) (string, error) {
	path := filepath.Join(dir, RepoDirName, configFileName)
	data, err := os.ReadFile(path) // #nosec G304 — path is constructed from a trusted dir
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("reading identity config: %w", err)
	}
	var cfg IdentityConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", fmt.Errorf("parsing identity config: %w", err)
	}
	return cfg.GitHubUser, nil
}

// SaveIdentity writes the verified GitHub username to .deepdiffdb/config.
// The file is owner-readable only (0o600). No token is persisted.
func SaveIdentity(dir, username string) error {
	cfg := IdentityConfig{GitHubUser: username}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding identity config: %w", err)
	}
	path := filepath.Join(dir, RepoDirName, configFileName)
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil { // #nosec G306
		return fmt.Errorf("writing identity config: %w", err)
	}
	return nil
}

// ResolveClientID returns the GitHub OAuth App client ID to use. The
// DEEPDIFFDB_GITHUB_CLIENT_ID environment variable takes precedence over the
// build-time default.
func ResolveClientID() string {
	if v := os.Getenv("DEEPDIFFDB_GITHUB_CLIENT_ID"); v != "" {
		return v
	}
	return GitHubClientID
}

// RunGitHubDeviceFlow performs the GitHub OAuth device flow and returns the
// authenticated GitHub username. The access token is used only to verify identity
// and is never stored.
func RunGitHubDeviceFlow(clientID string) (string, error) {
	dc, err := requestDeviceCode(clientID)
	if err != nil {
		return "", err
	}

	fmt.Printf("\n  Open:  %s\n", dc.verificationURI)
	fmt.Printf("  Code:  %s\n\n", dc.userCode)
	fmt.Print("  Waiting for authorization")

	token, err := pollForToken(clientID, dc.deviceCode, dc.interval, dc.expiresIn)
	if err != nil {
		fmt.Println() // newline after dots
		return "", err
	}
	fmt.Println(" ✓")

	return resolveGitHubUsername(token)
}

// --- internal types ----------------------------------------------------------

type deviceCodeResp struct {
	deviceCode      string
	userCode        string
	verificationURI string
	expiresIn       int
	interval        int
}

type rawDeviceCode struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type rawTokenResp struct {
	AccessToken string `json:"access_token"`
	Error       string `json:"error"`
}

type rawGitHubUser struct {
	Login string `json:"login"`
}

// --- private helpers ---------------------------------------------------------

func requestDeviceCode(clientID string) (*deviceCodeResp, error) {
	params := url.Values{}
	params.Set("client_id", clientID)
	params.Set("scope", "read:user")

	req, err := http.NewRequest(http.MethodPost, githubDeviceURL, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, fmt.Errorf("building device code request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	c := &http.Client{Timeout: 15 * time.Second}
	res, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requesting device code from GitHub: %w", err)
	}
	defer res.Body.Close()

	var raw rawDeviceCode
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decoding device code response: %w", err)
	}
	if raw.DeviceCode == "" {
		return nil, fmt.Errorf("GitHub returned empty device code — verify DEEPDIFFDB_GITHUB_CLIENT_ID is set correctly")
	}
	interval := raw.Interval
	if interval < 5 {
		interval = 5
	}
	return &deviceCodeResp{
		deviceCode:      raw.DeviceCode,
		userCode:        raw.UserCode,
		verificationURI: raw.VerificationURI,
		expiresIn:       raw.ExpiresIn,
		interval:        interval,
	}, nil
}

func pollForToken(clientID, deviceCode string, interval, expiresIn int) (string, error) {
	deadline := time.Now().Add(time.Duration(expiresIn) * time.Second)
	c := &http.Client{Timeout: 15 * time.Second}

	for time.Now().Before(deadline) {
		time.Sleep(time.Duration(interval) * time.Second)
		fmt.Print(".")

		params := url.Values{}
		params.Set("client_id", clientID)
		params.Set("device_code", deviceCode)
		params.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

		req, err := http.NewRequest(http.MethodPost, githubTokenURL, strings.NewReader(params.Encode()))
		if err != nil {
			return "", fmt.Errorf("building token request: %w", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Accept", "application/json")

		res, err := c.Do(req)
		if err != nil {
			return "", fmt.Errorf("polling GitHub for token: %w", err)
		}

		var tr rawTokenResp
		decodeErr := json.NewDecoder(res.Body).Decode(&tr)
		res.Body.Close()
		if decodeErr != nil {
			return "", fmt.Errorf("decoding token response: %w", decodeErr)
		}

		switch tr.Error {
		case "":
			if tr.AccessToken != "" {
				return tr.AccessToken, nil
			}
		case "authorization_pending":
			continue
		case "slow_down":
			interval += 5
		case "expired_token":
			return "", fmt.Errorf("device code expired — run `version init --auth` again")
		case "access_denied":
			return "", fmt.Errorf("GitHub authorization was denied")
		default:
			return "", fmt.Errorf("GitHub auth error: %s", tr.Error)
		}
	}
	return "", fmt.Errorf("timed out waiting for GitHub authorization")
}

func resolveGitHubUsername(token string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, githubUserURL, nil)
	if err != nil {
		return "", fmt.Errorf("building user request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	c := &http.Client{Timeout: 10 * time.Second}
	res, err := c.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching GitHub user info: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned HTTP %d when fetching user", res.StatusCode)
	}

	var user rawGitHubUser
	if err := json.NewDecoder(res.Body).Decode(&user); err != nil {
		return "", fmt.Errorf("decoding GitHub user response: %w", err)
	}
	if user.Login == "" {
		return "", fmt.Errorf("GitHub returned an empty username")
	}
	return user.Login, nil
}
