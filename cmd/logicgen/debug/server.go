//go:build debug

package debug

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// Server is an SSE (Server-Sent Events) server that broadcasts debug events to connected clients.
// It implements the DebugHook interface.
//
// Usage:
//
//	server := debug.NewServer()
//	go server.Run()
//	go server.ListenAndServe(":8080")
//
// Connect from browser:
//
//	const evtSource = new EventSource("http://localhost:8080/debug");
//	evtSource.onmessage = (e) => console.log(JSON.parse(e.data));
type Server struct {
	clients    map[chan *DebugMessage]bool
	broadcast  chan *DebugMessage
	register   chan chan *DebugMessage
	unregister chan chan *DebugMessage
	mu         sync.RWMutex
}

// NewServer creates a new debug SSE server.
func NewServer() *Server {
	return &Server{
		clients:    make(map[chan *DebugMessage]bool),
		broadcast:  make(chan *DebugMessage, 256),
		register:   make(chan chan *DebugMessage),
		unregister: make(chan chan *DebugMessage),
	}
}

// Run starts the server's main loop. Call this in a goroutine.
func (s *Server) Run() {
	for {
		select {
		case client := <-s.register:
			s.mu.Lock()
			s.clients[client] = true
			s.mu.Unlock()
			log.Printf("[DEBUG] Client connected. Total clients: %d", len(s.clients))

		case client := <-s.unregister:
			s.mu.Lock()
			if _, ok := s.clients[client]; ok {
				delete(s.clients, client)
				close(client)
			}
			s.mu.Unlock()
			log.Printf("[DEBUG] Client disconnected. Total clients: %d", len(s.clients))

		case msg := <-s.broadcast:
			s.mu.RLock()
			for client := range s.clients {
				select {
				case client <- msg:
				default:
					// Client channel full, skip
				}
			}
			s.mu.RUnlock()
		}
	}
}

// HandleSSE handles SSE (Server-Sent Events) connections.
func (s *Server) HandleSSE(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create client channel
	clientChan := make(chan *DebugMessage, 64)
	s.register <- clientChan

	// Remove client on disconnect
	defer func() {
		s.unregister <- clientChan
	}()

	// Get flusher for streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Send initial connection message
	fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"connected\"}\n\n")
	flusher.Flush()

	// Stream messages
	for {
		select {
		case msg, ok := <-clientChan:
			if !ok {
				return
			}
			data, err := json.Marshal(msg)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()

		case <-r.Context().Done():
			return
		}
	}
}

// ListenAndServe starts the HTTP server with SSE endpoint.
func (s *Server) ListenAndServe(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug", s.HandleSSE)
	mux.HandleFunc("/debug/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","clients":%d}`, s.ClientCount())
	})
	log.Printf("[DEBUG] Debug SSE server listening on %s/debug", addr)
	return http.ListenAndServe(addr, mux)
}

// send queues a message for broadcast to all clients.
func (s *Server) send(msg *DebugMessage) {
	select {
	case s.broadcast <- msg:
	default:
		// Channel full, drop message to avoid blocking
	}
}

// ClientCount returns the number of connected clients.
func (s *Server) ClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients)
}

// ============================================================================
// DebugHook Implementation
// ============================================================================

var _ DebugHook = (*Server)(nil)

func (s *Server) OnEventStart(sessionID, handler string, params map[string]any) {
	s.send(&DebugMessage{
		Type:      MsgEventStart,
		SessionID: sessionID,
		Handler:   handler,
		Params:    params,
		Timestamp: time.Now().UnixMilli(),
	})
}

func (s *Server) OnEventEnd(sessionID, handler string, durationMs float64, err error) {
	msg := &DebugMessage{
		Type:       MsgEventEnd,
		SessionID:  sessionID,
		Handler:    handler,
		DurationMs: durationMs,
		Timestamp:  time.Now().UnixMilli(),
	}
	if err != nil {
		msg.Error = err.Error()
	}
	s.send(msg)
}

func (s *Server) OnNodeStart(sessionID, handler, nodeID, nodeType string, inputs map[string]any) {
	s.send(&DebugMessage{
		Type:      MsgNodeStart,
		SessionID: sessionID,
		Handler:   handler,
		NodeID:    nodeID,
		NodeType:  nodeType,
		Inputs:    inputs,
		Timestamp: time.Now().UnixMilli(),
	})
}

func (s *Server) OnNodeEnd(sessionID, handler, nodeID string, outputs map[string]any, durationMs float64) {
	s.send(&DebugMessage{
		Type:       MsgNodeEnd,
		SessionID:  sessionID,
		Handler:    handler,
		NodeID:     nodeID,
		Outputs:    outputs,
		DurationMs: durationMs,
		Timestamp:  time.Now().UnixMilli(),
	})
}

func (s *Server) OnNodeError(sessionID, handler, nodeID string, err error) {
	msg := &DebugMessage{
		Type:      MsgNodeError,
		SessionID: sessionID,
		Handler:   handler,
		NodeID:    nodeID,
		Timestamp: time.Now().UnixMilli(),
	}
	if err != nil {
		msg.Error = err.Error()
	}
	s.send(msg)
}

func (s *Server) OnNodeWait(sessionID, handler, nodeID string, duration time.Duration) {
	s.send(&DebugMessage{
		Type:      MsgNodeWait,
		SessionID: sessionID,
		Handler:   handler,
		NodeID:    nodeID,
		WaitMs:    duration.Milliseconds(),
		ResumeAt:  time.Now().Add(duration).UnixMilli(),
		Timestamp: time.Now().UnixMilli(),
	})
}

func (s *Server) OnNodeResume(sessionID, handler, nodeID string) {
	s.send(&DebugMessage{
		Type:      MsgNodeResume,
		SessionID: sessionID,
		Handler:   handler,
		NodeID:    nodeID,
		Timestamp: time.Now().UnixMilli(),
	})
}
