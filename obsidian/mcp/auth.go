package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func validateGitHubToken(token string) bool {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("GitHub token validation error: %v", err)
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}

func checkAuth(r *http.Request) bool {
	if apiKey != "" {
		providedKey := r.Header.Get("Authorization")
		if providedKey == "" {
			providedKey = r.URL.Query().Get("api_key")
		}
		if providedKey == apiKey {
			return true
		}
	}

	if githubClientID != "" && githubClientSecret != "" {
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if validateGitHubToken(token) {
				return true
			}
		}
	}

	return false
}

func handleAuthorize(w http.ResponseWriter, r *http.Request) {
	if githubClientID == "" || githubClientSecret == "" {
		http.Error(w, "OAuth not configured", http.StatusBadRequest)
		return
	}

	redirectURI := r.URL.Query().Get("redirect_uri")
	state := r.URL.Query().Get("state")

	authURL := fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=read:user&state=%s",
		githubClientID,
		url.QueryEscape(redirectURI),
		state,
	)

	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

func handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	redirectURI := r.URL.Query().Get("redirect_uri")
	state := r.URL.Query().Get("state")

	tokenURL := "https://github.com/login/oauth/access_token"
	reqBody := fmt.Sprintf("client_id=%s&client_secret=%s&code=%s",
		url.QueryEscape(githubClientID),
		url.QueryEscape(githubClientSecret),
		url.QueryEscape(code),
	)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(reqBody))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	accessToken, ok := result["access_token"].(string)
	if !ok {
		http.Error(w, "Failed to get access token", http.StatusInternalServerError)
		return
	}

	finalRedirect := redirectURI + "#access_token=" + accessToken
	if state != "" {
		finalRedirect += "&state=" + state
	}

	http.Redirect(w, r, finalRedirect, http.StatusTemporaryRedirect)
}
