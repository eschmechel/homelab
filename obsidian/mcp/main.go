package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
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

func init() {
	apiKey = os.Getenv("API_KEY")
	githubClientID = os.Getenv("GITHUB_CLIENT_ID")
	githubClientSecret = os.Getenv("GITHUB_CLIENT_SECRET")
}

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

func main() {
	if len(os.Args) > 1 {
		vaultPath = os.Args[1]
	}
	if len(os.Args) > 2 {
		port = os.Args[2]
	}

	// Check if running in stdio mode (no port argument) or HTTP mode
	if len(os.Args) <= 2 {
		// Stdio mode
		scanner := bufio.NewScanner(os.Stdin)
		writer := bufio.NewWriter(os.Stdout)

		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			var req JSONRPCRequest
			if err := json.Unmarshal([]byte(line), &req); err != nil {
				continue
			}

			var resp JSONRPCResponse
			switch req.Method {
			case "initialize":
				resp = handleInitialize(req.ID)
			case "tools/list":
				resp = handleToolsList(req.ID)
			case "tools/call":
				resp = handleToolsCall(req.ID, req.Params)
			default:
				resp = JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Error:   &JSONError{Code: -32601, Message: "Method not found"},
				}
			}

			respBytes, _ := json.Marshal(resp)
			writer.Write(respBytes)
			writer.WriteString("\n")
			writer.Flush()
		}
		return
	}

	// HTTP/SSE mode
	http.HandleFunc("/sse", handleSSE)
	http.HandleFunc("/message", handleMessage)
	http.HandleFunc("/health", handleHealth)

	log.Printf("Starting obsidian-mcp on port %s", port)
	log.Printf("Vault path: %s", vaultPath)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

func handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		return
	}

	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		sessionID = fmt.Sprintf("session-%d", time.Now().UnixNano())
	}

	writer := bufio.NewWriter(w)

	session := &Session{
		id:     sessionID,
		writer: writer,
	}
	sessionsMu.Lock()
	sessions[sessionID] = session
	sessionsMu.Unlock()

	defer func() {
		sessionsMu.Lock()
		delete(sessions, sessionID)
		sessionsMu.Unlock()
	}()

	writer.Write([]byte("event: connected\n"))
	writer.Write([]byte(fmt.Sprintf("data: {\"sessionId\":\"%s\"}\n\n", sessionID)))
	writer.Flush()
	flusher.Flush()

	<-r.Context().Done()
}

func handleMessage(w http.ResponseWriter, r *http.Request) {
	// Check authentication (API key or OAuth)
	authorized := false

	// Check API key if configured
	if apiKey != "" {
		providedKey := r.Header.Get("Authorization")
		if providedKey == "" {
			providedKey = r.URL.Query().Get("api_key")
		}
		if providedKey == apiKey {
			authorized = true
		}
	}

	// Check OAuth token if configured
	if !authorized && githubClientID != "" && githubClientSecret != "" {
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if validateGitHubToken(token) {
				authorized = true
			}
		}
	}

	if !authorized {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		http.Error(w, "sessionId required", http.StatusBadRequest)
		return
	}

	sessionsMu.RLock()
	session, exists := sessions[sessionID]
	sessionsMu.RUnlock()

	if !exists || session == nil {
		http.Error(w, "no active session", http.StatusBadRequest)
		return
	}

	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	var resp JSONRPCResponse
	switch req.Method {
	case "initialize":
		resp = handleInitialize(req.ID)
	case "tools/list":
		resp = handleToolsList(req.ID)
	case "tools/call":
		resp = handleToolsCall(req.ID, req.Params)
	default:
		resp = JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &JSONError{Code: -32601, Message: "Method not found"},
		}
	}

	respBytes, _ := json.Marshal(resp)
	w.Header().Set("Content-Type", "application/json")
	w.Write(respBytes)
}

func handleInitialize(id interface{}) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"serverInfo": map[string]any{
				"name":    "obsidian-filesystem",
				"version": "0.1.0",
			},
		},
	}
}

func handleToolsList(id interface{}) JSONRPCResponse {
	tools := []map[string]any{
		{
			"name":        "list_directory",
			"description": "List files and directories in a folder",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{"type": "string", "description": "Relative path to directory"},
				},
				"required": []string{"path"},
			},
		},
		{
			"name":        "read_file",
			"description": "Read contents of a file",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{"type": "string", "description": "Relative path to file"},
				},
				"required": []string{"path"},
			},
		},
		{
			"name":        "write_file",
			"description": "Write content to a file",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path":    map[string]any{"type": "string", "description": "Relative path to file"},
					"content": map[string]any{"type": "string", "description": "Content to write"},
				},
				"required": []string{"path", "content"},
			},
		},
		{
			"name":        "delete_file",
			"description": "Delete a file",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{"type": "string", "description": "Relative path to file"},
				},
				"required": []string{"path"},
			},
		},
		{
			"name":        "create_directory",
			"description": "Create a new directory",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{"type": "string", "description": "Relative path to directory"},
				},
				"required": []string{"path"},
			},
		},
		{
			"name":        "search_files",
			"description": "Search for files by name pattern",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"pattern": map[string]any{"type": "string", "description": "Glob pattern"},
					"path":    map[string]any{"type": "string", "description": "Directory to search"},
				},
				"required": []string{"pattern"},
			},
		},
		{
			"name":        "search_content",
			"description": "Search for text within files",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string", "description": "Text to search for"},
					"path":  map[string]any{"type": "string", "description": "Directory to search"},
				},
				"required": []string{"query"},
			},
		},
		{
			"name":        "move_file",
			"description": "Move or rename a file",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"source": map[string]any{"type": "string", "description": "Source path"},
					"dest":   map[string]any{"type": "string", "description": "Destination path"},
				},
				"required": []string{"source", "dest"},
			},
		},
		{
			"name":        "get_file_info",
			"description": "Get file metadata",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{"type": "string", "description": "Relative path to file"},
				},
				"required": []string{"path"},
			},
		},
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]any{
			"tools": tools,
		},
	}
}

func handleToolsCall(id interface{}, params map[string]any) JSONRPCResponse {
	arguments, _ := params["arguments"].(map[string]any)
	if arguments == nil {
		arguments = params
	}

	toolName, _ := params["name"].(string)

	switch toolName {
	case "list_directory":
		path, _ := arguments["path"].(string)
		fullPath := filepath.Join(vaultPath, path)
		entries, err := os.ReadDir(fullPath)
		if err != nil {
			return JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      id,
				Error:   &JSONError{Code: -32602, Message: fmt.Sprintf("Error: %v", err)},
			}
		}
		var files []string
		for _, e := range entries {
			files = append(files, e.Name())
		}
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": fmt.Sprintf("%v", files)},
				},
			},
		}

	case "read_file":
		path, _ := arguments["path"].(string)
		fullPath := filepath.Join(vaultPath, path)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      id,
				Error:   &JSONError{Code: -32602, Message: fmt.Sprintf("Error: %v", err)},
			}
		}
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": string(data)},
				},
			},
		}

	case "write_file":
		path, _ := arguments["path"].(string)
		content, _ := arguments["content"].(string)
		fullPath := filepath.Join(vaultPath, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      id,
				Error:   &JSONError{Code: -32602, Message: fmt.Sprintf("Error: %v", err)},
			}
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      id,
				Error:   &JSONError{Code: -32602, Message: fmt.Sprintf("Error: %v", err)},
			}
		}
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": fmt.Sprintf("File written: %s", path)},
				},
			},
		}

	case "delete_file":
		path, _ := arguments["path"].(string)
		fullPath := filepath.Join(vaultPath, path)
		if err := os.Remove(fullPath); err != nil {
			return JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      id,
				Error:   &JSONError{Code: -32602, Message: fmt.Sprintf("Error: %v", err)},
			}
		}
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": fmt.Sprintf("File deleted: %s", path)},
				},
			},
		}

	case "create_directory":
		path, _ := arguments["path"].(string)
		fullPath := filepath.Join(vaultPath, path)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      id,
				Error:   &JSONError{Code: -32602, Message: fmt.Sprintf("Error: %v", err)},
			}
		}
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": fmt.Sprintf("Directory created: %s", path)},
				},
			},
		}

	case "search_files":
		pattern, _ := arguments["pattern"].(string)
		searchPath, _ := arguments["path"].(string)
		fullPath := vaultPath
		if searchPath != "" {
			fullPath = filepath.Join(vaultPath, searchPath)
		}
		var results []string
		filepath.Walk(fullPath, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			name := filepath.Base(p)
			matched, _ := filepath.Match(pattern, name)
			if matched {
				rel, _ := filepath.Rel(vaultPath, p)
				results = append(results, rel)
			}
			return nil
		})
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": strings.Join(results, "\n")},
				},
			},
		}

	case "search_content":
		query, _ := arguments["query"].(string)
		searchPath, _ := arguments["path"].(string)
		fullPath := vaultPath
		if searchPath != "" {
			fullPath = filepath.Join(vaultPath, searchPath)
		}
		var results []string
		filepath.Walk(fullPath, func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(p, ".md") {
				return nil
			}
			data, err := os.ReadFile(p)
			if err != nil {
				return nil
			}
			if strings.Contains(string(data), query) {
				rel, _ := filepath.Rel(vaultPath, p)
				results = append(results, rel)
			}
			return nil
		})
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": strings.Join(results, "\n")},
				},
			},
		}

	case "move_file":
		source, _ := arguments["source"].(string)
		dest, _ := arguments["dest"].(string)
		srcPath := filepath.Join(vaultPath, source)
		dstPath := filepath.Join(vaultPath, dest)
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      id,
				Error:   &JSONError{Code: -32602, Message: fmt.Sprintf("Error: %v", err)},
			}
		}
		if err := os.Rename(srcPath, dstPath); err != nil {
			return JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      id,
				Error:   &JSONError{Code: -32602, Message: fmt.Sprintf("Error: %v", err)},
			}
		}
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": fmt.Sprintf("File moved: %s -> %s", source, dest)},
				},
			},
		}

	case "get_file_info":
		path, _ := arguments["path"].(string)
		fullPath := filepath.Join(vaultPath, path)
		info, err := os.Stat(fullPath)
		if err != nil {
			return JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      id,
				Error:   &JSONError{Code: -32602, Message: fmt.Sprintf("Error: %v", err)},
			}
		}
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": fmt.Sprintf("Name: %s\nSize: %d bytes\nModified: %s\nIs Directory: %v", info.Name(), info.Size(), info.ModTime().Format(time.RFC3339), info.IsDir())},
				},
			},
		}
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &JSONError{Code: -32602, Message: "Unknown tool"},
	}
}

func validateGitHubToken(token string) bool {
	// Validate token against GitHub API
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
