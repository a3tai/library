package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/a3tai/library/internal/library"
	"github.com/a3tai/library/internal/mcpserver"
)

func (s *LibraryService) Stats() (library.Stats, error) {
	return s.store.Stats(context.Background())
}

func (s *LibraryService) StartMCPServer(port int) (MCPStatus, error) {
	s.mu.Lock()
	if s.mcp != nil {
		status := s.mcpStatusLocked()
		s.mu.Unlock()
		return status, nil
	}
	if port <= 0 {
		port = 8765
	}
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		s.mu.Unlock()
		return MCPStatus{}, err
	}
	actualPort := listener.Addr().(*net.TCPAddr).Port
	token, err := newMCPToken()
	if err != nil {
		_ = listener.Close()
		s.mu.Unlock()
		return MCPStatus{}, err
	}
	server := &http.Server{Handler: authenticatedMCPHandler(token, mcpserver.NewWithEmbedder(s.store, s.embed).HTTPHandler())}
	s.mcp = server
	s.mcpURL = fmt.Sprintf("http://127.0.0.1:%d/mcp", actualPort)
	s.mcpToken = token
	status := s.mcpStatusLocked()
	s.mu.Unlock()

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			s.mu.Lock()
			if s.mcp == server {
				s.mcp = nil
				s.mcpURL = ""
				s.mcpToken = ""
			}
			s.mu.Unlock()
		}
	}()
	return status, nil
}

func (s *LibraryService) StopMCPServer() (MCPStatus, error) {
	s.mu.Lock()
	server := s.mcp
	s.mcp = nil
	s.mcpURL = ""
	s.mcpToken = ""
	s.mu.Unlock()
	if server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			return MCPStatus{}, err
		}
	}
	return s.MCPStatus(), nil
}

func (s *LibraryService) MCPStatus() MCPStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mcpStatusLocked()
}

func (s *LibraryService) isBusy() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.busy
}

func (s *LibraryService) setBusy(value bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if value && s.busy {
		return false
	}
	s.busy = value
	return true
}

func (s *LibraryService) mcpStatusLocked() MCPStatus {
	status := MCPStatus{Running: s.mcp != nil, URL: s.mcpURL, Token: s.mcpToken}
	if s.mcpURL != "" {
		_, _ = fmt.Sscanf(s.mcpURL, "http://127.0.0.1:%d/mcp", &status.Port)
	}
	return status
}

func newMCPToken() (string, error) {
	var buf [32]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf[:]), nil
}

func authenticatedMCPHandler(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/mcp" && r.Method != http.MethodOptions {
			if !mcpAuthOK(r, token) {
				w.Header().Set("WWW-Authenticate", `Bearer realm="a3t-library-mcp"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func mcpAuthOK(r *http.Request, token string) bool {
	if token == "" {
		return false
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") && strings.TrimSpace(auth[len("bearer "):]) == token {
		return true
	}
	return strings.TrimSpace(r.Header.Get("X-A3T-MCP-Token")) == token
}
