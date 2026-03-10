package main

import (
	"bufio"
	"encoding/json"
	"log"
	"net/http"
	"os"
)

func main() {
	if len(os.Args) > 1 {
		vaultPath = os.Args[1]
	}
	if len(os.Args) > 2 {
		port = os.Args[2]
	}

	if len(os.Args) <= 2 {
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

	http.HandleFunc("/sse", handleSSE)
	http.HandleFunc("/message", handleMessage)
	http.HandleFunc("/rpc", handleRPC)
	http.HandleFunc("/health", handleHealth)

	if githubClientID != "" && githubClientSecret != "" {
		http.HandleFunc("/authorize", handleAuthorize)
		http.HandleFunc("/oauth/callback", handleOAuthCallback)
	}

	log.Printf("Starting obsidian-mcp on port %s", port)
	log.Printf("Vault path: %s", vaultPath)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
