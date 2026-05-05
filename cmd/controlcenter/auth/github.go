package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const (
	githubAuthorizeURL = "https://github.com/login/oauth/authorize"
	githubTokenURL     = "https://github.com/login/oauth/access_token"
	githubUserURL      = "https://api.github.com/user"
)

// GitHubEnabled reports whether GitHub OAuth is configured.
// Returns true when GITHUB_CLIENT_ID is set.
func GitHubEnabled() bool {
	return os.Getenv("GITHUB_CLIENT_ID") != ""
}

// GitHubBegin redirects the browser to GitHub's OAuth authorization page.
// Route: GET /auth/github
func GitHubBegin(w http.ResponseWriter, r *http.Request) {
	clientID := os.Getenv("GITHUB_CLIENT_ID")
	if clientID == "" {
		http.Error(w, "GitHub OAuth not configured (GITHUB_CLIENT_ID not set)", http.StatusServiceUnavailable)
		return
	}

	params := url.Values{}
	params.Set("client_id", clientID)
	params.Set("scope", "read:user")

	http.Redirect(w, r, githubAuthorizeURL+"?"+params.Encode(), http.StatusFound)
}

// GitHubCallback handles the OAuth callback from GitHub.
// Exchanges the code for an access token, fetches the GitHub username,
// creates a session cookie, and redirects to the Control Center.
// Route: GET /auth/github/callback
func GitHubCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code parameter", http.StatusBadRequest)
		return
	}

	clientID := os.Getenv("GITHUB_CLIENT_ID")
	clientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		http.Error(w, "GitHub OAuth not configured", http.StatusServiceUnavailable)
		return
	}

	// Exchange code for access token
	token, err := exchangeGitHubCode(code, clientID, clientSecret)
	if err != nil {
		http.Error(w, "GitHub OAuth failed: "+err.Error(), http.StatusBadGateway)
		return
	}

	// Fetch the GitHub username
	login, err := fetchGitHubLogin(token)
	if err != nil {
		http.Error(w, "could not fetch GitHub user: "+err.Error(), http.StatusBadGateway)
		return
	}

	// Create session — same mechanism as username/password login
	sessionToken := signSession(login)
	http.SetCookie(w, &http.Cookie{
		Name:     orkSession,
		Value:    sessionToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
	})

	http.Redirect(w, r, "/controlcenter", http.StatusFound)
}

func exchangeGitHubCode(code, clientID, clientSecret string) (string, error) {
	params := url.Values{}
	params.Set("client_id", clientID)
	params.Set("client_secret", clientSecret)
	params.Set("code", code)

	req, err := http.NewRequest(http.MethodPost, githubTokenURL, strings.NewReader(params.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token exchange: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decoding token response: %w", err)
	}
	if result.Error != "" {
		return "", fmt.Errorf("%s: %s", result.Error, result.ErrorDesc)
	}
	if result.AccessToken == "" {
		return "", fmt.Errorf("empty access token from GitHub")
	}
	return result.AccessToken, nil
}

func fetchGitHubLogin(token string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, githubUserURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("user request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API HTTP %d: %s", resp.StatusCode, body)
	}

	var user struct {
		Login string `json:"login"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", fmt.Errorf("decoding user: %w", err)
	}
	if user.Login == "" {
		return "", fmt.Errorf("empty login from GitHub")
	}
	return user.Login, nil
}
