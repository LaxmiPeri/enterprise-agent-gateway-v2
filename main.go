package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

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

func init() {
    var err error
	// Automatically scans your current root folder for a file named '.env'
	err = godotenv.Load()
	if err != nil {
		log.Println("⚠️  Warning: No local .env file found. Falling back to native system environment shell variables.")
	} else {
		log.Println("✅ Environment configuration values loaded from local .env cleanly.")
	}

	// Initialize the Google GenAI SDK client context
	// It automatically hooks into os.Getenv("GOOGLE_API_KEY")
	ctx := context.Background()
	
	geminiClient, err = genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatalf("❌ Failed to instantiate native Google GenAI Go Client: %v", err)
	}
	log.Println("✅ Native Google GenAI Client pool compiled and ready.")

	// Set MOCK_LLM=true to bypass the live Gemini call with a canned response.
	// Use this during vegeta/load testing so results reflect gateway throughput
	// rather than upstream model latency, rate limits, or cost.
	if strings.EqualFold(os.Getenv("MOCK_LLM"), "true") {
		resolutionGenerator = mockResolutionText
		log.Println("🧪 MOCK_LLM=true — resolutionGenerator swapped to mockResolutionText for stress testing.")
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
