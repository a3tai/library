package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
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
	server := &http.Server{Handler: mcpserver.NewWithEmbedder(s.store, s.embed).HTTPHandler()}
	s.mcp = server
	s.mcpURL = fmt.Sprintf("http://127.0.0.1:%d/mcp", actualPort)
	status := s.mcpStatusLocked()
	s.mu.Unlock()

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			s.mu.Lock()
			if s.mcp == server {
				s.mcp = nil
				s.mcpURL = ""
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
	status := MCPStatus{Running: s.mcp != nil, URL: s.mcpURL}
	if s.mcpURL != "" {
		_, _ = fmt.Sscanf(s.mcpURL, "http://127.0.0.1:%d/mcp", &status.Port)
	}
	return status
}
