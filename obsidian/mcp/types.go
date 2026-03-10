package main

import (
	"bufio"
	"os"
	"path/filepath"
	"sync"
)

type JSONRPCRequest struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      interface{}    `json:"id"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  any         `json:"result,omitempty"`
	Error   *JSONError  `json:"error,omitempty"`
}

type JSONError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

var vaultPath = "/opt/obsidian-vault"
var port = "3333"

var apiKey string
var githubClientID string
var githubClientSecret string

type Session struct {
	id     string
	writer *bufio.Writer
	mu     sync.Mutex
}

var sessions = make(map[string]*Session)
var sessionsMu sync.RWMutex

func init() {
	apiKey = os.Getenv("API_KEY")
	githubClientID = os.Getenv("GITHUB_CLIENT_ID")
	githubClientSecret = os.Getenv("GITHUB_CLIENT_SECRET")
}

func resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(vaultPath, path)
}
