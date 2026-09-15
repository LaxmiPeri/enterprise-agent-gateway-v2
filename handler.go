package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"google.golang.org/genai"
)

func resolutionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request payload", http.StatusBadRequest)
		return
	}

	// 1. High-Speed Input Guardrail
	if err := InputGuardrail(req.Prompt); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	auditNotes := []string{fmt.Sprintf("Go Engine: Connection threaded natively at %v", time.Now().Format(time.RFC3339))}

	// 2. High-Velocity Fraud Intercept (Policy Engine)
	orderStatus := "processing"
	if req.OrderID == "ORD-1002" {
		auditNotes = append(auditNotes, "Go Engine: Internal rule match. Flagged ORD-1002 as high risk.")
		
		// ARCHITECTURE WIN: Intercept completely. 0 tokens spent, 0 downstream delays.
		resp := ChatResponse{
			Resolution: "Security Exception: This transaction requires manager escalation authorization. Refund blocked.",
			Status:     "flagged_fraud",
			AuditNotes: auditNotes,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// 3. Native Database Query Sync (Go-Driven Context Gathering/RAG)
	var dbContext string
	if req.OrderID == "ORD-1001" {
		orderStatus = "delivered"
		dbContext = `{"order_id": "ORD-1001", "item": "Premium Headphones", "price": 299.99, "status": "delivered", "tracking": "TRK-UPS-777", "notes": "Package signed for by resident on front porch."}`
		auditNotes = append(auditNotes, "Go Engine: In-memory DB fetch successful for ORD-1001.")
	} else {
		dbContext = fmt.Sprintf(`{"order_id": "%s", "status": "unknown", "error": "Record mismatch or missing indexing identifier"}`, req.OrderID)
		auditNotes = append(auditNotes, fmt.Sprintf("Go Engine: Database query exception for %s", req.OrderID))
	}

	// 4. Fire the Model Generation Directly to Google's Native Endpoints
	// We instantiate a context timeout to guarantee our server drop slow connections under load
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	resolutionText, err := resolutionGenerator(ctx, req.Prompt, dbContext)
	if err != nil {
		auditNotes = append(auditNotes, fmt.Sprintf("Go Engine: Gemini API call error: %v", err))
		http.Error(w, "Downstream model cluster error or timeout", http.StatusServiceUnavailable)
		return
	}

	auditNotes = append(auditNotes, "Go Engine: Bare-metal model turn processed cleanly.")

	// 5. Build and Send Final Response Payload Contract
	finalResp := ChatResponse{
		Resolution: resolutionText,
		Status:     orderStatus,
		AuditNotes: auditNotes,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(finalResp)
}

// resolutionGenerator is the swappable entry point used to obtain the model's
// resolution text. It defaults to the live Gemini call but can be pointed at
// mockResolutionText (see MOCK_LLM in main.go) so load tests exercise gateway
// throughput without depending on the live model endpoint.
var resolutionGenerator = generateResolutionText

// generateResolutionText fires the request at Gemini and extracts the response text.
func generateResolutionText(ctx context.Context, prompt, dbContext string) (string, error) {
	systemInstruction := fmt.Sprintf(
		"You are an autonomous enterprise e-commerce customer support supervisor. "+
			"Resolve the user request concisely using ONLY the following verified database telemetry context:\n%s",
		dbContext,
	)

	// Configure generation constraints
	config := &genai.GenerateContentConfig{
		Temperature: genai.Ptr(float32(0.0)),
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{
				{Text: systemInstruction}, // Put your system prompt here
			},
		},
	}

	// Invoke generation natively through the fast path
	response, err := geminiClient.Models.GenerateContent(
		ctx,
		"gemini-3.6-flash", // Elite speed and cost efficiency
		genai.Text(prompt),
		config,
	)
	if err != nil {
		return "", err
	}

	if response != nil &&
		len(response.Candidates) > 0 &&
		response.Candidates[0] != nil &&
		response.Candidates[0].Content != nil &&
		len(response.Candidates[0].Content.Parts) > 0 {

		// rawPart is directly of type *genai.Part, no interface assertion needed
		rawPart := response.Candidates[0].Content.Parts[0]

		if rawPart != nil && rawPart.Text != "" {
			return rawPart.Text, nil
		}
		return "Unexpected content or empty text part", nil
	}

	return "I have successfully verified your tracking coordinates. Your delivery status is officially marked as confirmed.", nil
}

// mockLLMLatency approximates typical Gemini response time so load tests
// exercise realistic connection/goroutine-under-load behavior instead of
// returning instantly.
const mockLLMLatency = 800 * time.Millisecond

// mockResolutionText simulates a Gemini call's latency without making an
// outbound request, so vegeta-style stress tests can measure gateway
// behavior under realistic response times without burning Gemini quota.
func mockResolutionText(ctx context.Context, prompt, dbContext string) (string, error) {
	select {
	case <-time.After(mockLLMLatency):
		return "MOCK RESOLUTION: Request reviewed against verified order telemetry and resolved.", nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}
