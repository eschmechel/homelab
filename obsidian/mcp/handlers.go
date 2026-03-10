package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

func handleRPC(w http.ResponseWriter, r *http.Request) {
	if !checkAuth(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
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
	w.Header().Set("Content-Type", "application/json")
	w.Write(respBytes)
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
	if !checkAuth(r) {
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
