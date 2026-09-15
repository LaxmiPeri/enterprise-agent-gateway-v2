package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"google.golang.org/genai"
)

const ServerPort = ":8080"

type ChatRequest struct {
	Prompt  string `json:"prompt"`
	OrderID string `json:"order_id"`
}

type ChatResponse struct {
	Resolution string   `json:"resolution"`
	Status     string   `json:"status"`
	AuditNotes []string `json:"audit_notes"`
}

// Global thread-safe bare-metal Gemini Client instance
var geminiClient *genai.Client

// httpConnPool is the shared, tuned outbound transport used for every real
// network call the gateway makes — both to Gemini and (in MOCK_LLM mode) to
// the local mock LLM server, so a stress test actually exercises the same
// connection-pooling path production traffic would.
var httpConnPool = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:        500,
		MaxIdleConnsPerHost: 200,
		IdleConnTimeout:     90 * time.Second,
	},
	Timeout: 10 * time.Second, // Maximum duration constraint to protect the gateway
}

func init() {
    var err error
	// Automatically scans your current root folder for a file named '.env'
	err = godotenv.Load()
	if err != nil {
		log.Println("⚠️  Warning: No local .env file found. Falling back to native system environment shell variables.")
	} else {
		log.Println("✅ Environment configuration values loaded from local .env cleanly.")
	}

	ctx := context.Background()

	// We pass our optimized pool directly into the Client configuration block
	config := &genai.ClientConfig{
		HTTPClient: httpConnPool, // Forces Gemini to use our socket transport pool
	}

	geminiClient, err = genai.NewClient(ctx, config)
	if err != nil {
		log.Fatalf("❌ Failed to instantiate native Google GenAI Go Client: %v", err)
	}
	log.Println("✅ Gemini Client successfully bound to customized HTTP Connection Pool.")


	// Set MOCK_LLM=true to bypass the live Gemini call with a call to a local
	// mock LLM server instead. This keeps stress tests on a real network/socket
	// path (exercising httpConnPool) without depending on the live model endpoint.
	if strings.EqualFold(os.Getenv("MOCK_LLM"), "true") {
		startMockLLMServer(mockLLMAddr)
		resolutionGenerator = mockResolutionText
		log.Println("🧪 MOCK_LLM=true — resolutionGenerator swapped to mockResolutionText (local HTTP mock) for stress testing.")
	}
}

// InputGuardrail operates at microsecond speeds natively inside Go middleware
func InputGuardrail(prompt string) error {
	maliciousPhrases := []string{"ignore previous instructions", "system override", "override system prompt"}
	cleanPrompt := strings.ToLower(prompt)
	for _, phrase := range maliciousPhrases {
		if strings.Contains(cleanPrompt, phrase) {
			return fmt.Errorf("security exception: malicious activity signature blocked")
		}
	}
	return nil
}


func main() {
	// Simple route binding using standard library http primitives
	http.HandleFunc("/api/v2/chat", resolutionHandler)
	
	log.Printf("🚀 High-Throughput 100%% Pure Go Gateway active on port %s\n", ServerPort)
	if err := http.ListenAndServe(ServerPort, nil); err != nil {
		log.Fatalf("Server startup exception: %v", err)
	}
}
