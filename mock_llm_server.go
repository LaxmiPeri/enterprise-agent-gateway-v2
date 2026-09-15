package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// mockLLMAddr is the local address the mock LLM server binds to when
// MOCK_LLM=true. Loopback-only: this stands in for the real Gemini endpoint
// during load tests, it must never be reachable from outside the box.
const mockLLMAddr = "127.0.0.1:9099"

// mockLLMLatency approximates typical Gemini response time so load tests
// exercise realistic connection/goroutine-under-load behavior instead of
// returning instantly.
const mockLLMLatency = 800 * time.Millisecond

type mockLLMRequest struct {
	Prompt    string `json:"prompt"`
	DBContext string `json:"db_context"`
}

type mockLLMResponse struct {
	Text string `json:"text"`
}

// startMockLLMServer runs a tiny local HTTP server that stands in for the
// Gemini endpoint: it accepts the same shape of request the gateway would
// send, waits mockLLMLatency to simulate model inference time, and returns a
// canned response. This lets vegeta-style stress tests exercise a real
// network round trip (and the shared httpConnPool) instead of an in-process
// sleep.
func startMockLLMServer(addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/generate", mockLLMHandler)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		log.Printf("🧪 Mock LLM HTTP server listening on http://%s (simulating %v latency)", addr, mockLLMLatency)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Mock LLM server failed: %v", err)
		}
	}()
}

func mockLLMHandler(w http.ResponseWriter, r *http.Request) {
	var req mockLLMRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad mock request payload", http.StatusBadRequest)
		return
	}

	select {
	case <-time.After(mockLLMLatency):
	case <-r.Context().Done():
		return
	}

	resp := mockLLMResponse{
		Text: "MOCK RESOLUTION: Request reviewed against verified order telemetry and resolved.",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
