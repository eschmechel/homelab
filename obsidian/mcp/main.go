package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

func main() {
	if len(os.Args) > 1 {
		vaultPath = os.Args[1]
	}

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
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &JSONError{Code: -32602, Message: "Unknown tool"},
	}
}
