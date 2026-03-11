package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func matchRegex(pattern, content string) bool {
	re := regexp.MustCompile(pattern)
	return re.FindString(content) != ""
}

func findDateMatches(pattern, content, startDate, endDate string) []string {
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(content, -1)

	var results []string
	for _, match := range matches {
		if len(match) > 1 {
			date := match[1]
			if startDate != "" && date >= startDate {
				if endDate == "" || date <= endDate {
					results = append(results, date)
				}
			}
		}
	}
	return results
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
		{
			"name":        "search_by_tag",
			"description": "Search for notes by YAML frontmatter tags using ripgrep",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"tag":  map[string]any{"type": "string", "description": "Tag to search for"},
					"path": map[string]any{"type": "string", "description": "Directory to search (default: vault root)"},
				},
				"required": []string{"tag"},
			},
		},
		{
			"name":        "search_by_date",
			"description": "Search for notes by date range in YAML frontmatter",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"start_date": map[string]any{"type": "string", "description": "Start date (YYYY-MM-DD)"},
					"end_date":   map[string]any{"type": "string", "description": "End date (YYYY-MM-DD)"},
					"path":       map[string]any{"type": "string", "description": "Directory to search"},
				},
				"required": []string{"start_date"},
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
		fullPath := resolvePath(path)
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
		fullPath := resolvePath(path)
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
		fullPath := resolvePath(path)
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
		fullPath := resolvePath(path)
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
		fullPath := resolvePath(path)
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
			fullPath = resolvePath(searchPath)
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
			fullPath = resolvePath(searchPath)
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
		srcPath := resolvePath(source)
		dstPath := resolvePath(dest)
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
		fullPath := resolvePath(path)
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

	case "search_by_tag":
		tag, _ := arguments["tag"].(string)
		searchPath, _ := arguments["path"].(string)
		fullPath := vaultPath
		if searchPath != "" {
			fullPath = resolvePath(searchPath)
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
			content := string(data)
			tagPattern := fmt.Sprintf(`tags:.*%s`, tag)
			if strings.Contains(content, tag) || matchRegex(tagPattern, content) {
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

	case "search_by_date":
		startDate, _ := arguments["start_date"].(string)
		endDate, _ := arguments["end_date"].(string)
		searchPath, _ := arguments["path"].(string)
		fullPath := vaultPath
		if searchPath != "" {
			fullPath = resolvePath(searchPath)
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
			content := string(data)
			datePattern := `date:\s*(\d{4}-\d{2}-\d{2})`
			matches := findDateMatches(datePattern, content, startDate, endDate)
			if len(matches) > 0 {
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
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &JSONError{Code: -32602, Message: "Unknown tool"},
	}
}
