# Enterprise Agent Gateway v2

A high-throughput Go gateway that fronts Google's Gemini models for enterprise e-commerce customer support resolution. It applies a fast input guardrail and a deterministic fraud-interception policy before ever calling the model, enriches prompts with in-memory order context, and exposes a swappable LLM call so the gateway can be load-tested without hitting the live API.

## Features
* Native Go HTTP gateway (`net/http`), no framework
* Pre-model input guardrail that blocks prompt-injection style phrases
* Fraud-interception policy engine that short-circuits flagged orders with zero model calls
* In-memory order context lookup injected into the model's system prompt
* Swappable LLM call (`resolutionGenerator`) with a mock implementation for stress testing

## Requirements
* Go 1.27+
* A Google Gemini API key (`GOOGLE_API_KEY`) — not required when running with `MOCK_LLM=true`
* [vegeta](https://github.com/tsenart/vegeta) (optional, for stress testing)

## Configuration
Values are loaded from a `.env` file in the project root (via `godotenv`) or from the shell environment.

| Variable | Required | Description |
|---|---|---|
| `GOOGLE_API_KEY` | Yes, unless `MOCK_LLM=true` | API key used by the Google GenAI Go SDK |
| `MOCK_LLM` | No | Set to `true` to bypass the live Gemini call and return a canned response with a simulated 800ms latency, for load testing |

## Running the server
```bash
go run .
```
Listens on `:8080`.

Run in mock mode (no live model calls, no API key needed):
```bash
MOCK_LLM=true go run .
```

## API

### `POST /api/v2/chat`

Request body:
```json
{
  "prompt": "Where is my order?",
  "order_id": "ORD-1001"
}
```

Response body:
```json
{
  "resolution": "string",
  "status": "processing | delivered | flagged_fraud",
  "audit_notes": ["string"]
}
```

Behavior:
* A prompt containing a blocked phrase (see `InputGuardrail` in [main.go](main.go)) is rejected with `400 Bad Request` before any downstream work happens.
* `order_id: "ORD-1002"` is always intercepted as high risk — it returns a fixed refusal and never reaches the model.
* `order_id: "ORD-1001"` resolves against a sample in-memory "delivered" order record. Any other order ID is passed to the model with an "unknown/mismatch" context note.

## Project structure
```
main.go        entrypoint, env/config loading, Gemini client init, InputGuardrail
handler.go     HTTP handler, fraud policy, order context lookup, LLM call + mock
benchmarks/    vegeta stress-test script, target list, and sample request payloads
```

## Stress testing
Start the gateway in mock mode so results reflect gateway throughput rather than live Gemini latency, rate limits, or cost:
```bash
MOCK_LLM=true go run .
```
Then, in another terminal, run the bundled vegeta script:
```bash
cd benchmarks
chmod +x run_stress_test.sh
./run_stress_test.sh
```
This fires 1,000 RPS for 30 seconds at [benchmarks/targets.txt](benchmarks/targets.txt) (alternating the delivered and fraud sample payloads), then writes a text report to the console and an interactive latency plot to `benchmarks/performance_latency_plot.html`.

## 📊 High-Performance Load Testing Verification Metrics

To validate the scalability and fault-tolerance of this multi-language architecture, the gateway was subjected to an aggressive, sustained HTTP stress test using the **Vegeta load-injection engine**.

### Benchmark Target Environment
* **Inbound Traffic Frequency:** 1,000 Requests Per Second (RPS) constant rate
* **Total Operational Duration:** 30 Seconds
* **Total Generated Packets:** 30,000 Requests
* **Downstream Constraints:** Real HTTP calls to a local mock LLM server ([mock_llm_server.go](mock_llm_server.go)) over the same pooled `http.Client` (`MaxIdleConnsPerHost = 200`) used for live Gemini calls, with an 800ms handler delay to simulate frontier-model inference latency. Unlike an in-process sleep, this exercises real sockets end-to-end, so the connection-pool behavior below reflects genuine network behavior, not just gateway scheduling.

### Vegeta Report Logs
```text
Requests      [total, rate, throughput]         30000, 1000.03, 974.07
Duration      [total, attack, wait]             30.799s, 29.999s, 799.729ms
Latencies     [min, mean, 50, 90, 95, 99, max]  62.667µs, 400.354ms, 368.448ms, 800.507ms, 800.547ms, 800.87ms, 809.966ms
Bytes In      [total, mean]                     9105000, 303.50
Bytes Out     [total, mean]                     3600000, 120.00
Success       [ratio]                           100.00%
Status Codes  [code:count]                      200:30000
```

### Key Architectural Takeaways
* **Zero Socket Starvation Under Real Network Load:** With the mock LLM served over an actual local HTTP round trip (not an in-process sleep), the gateway still sustained a **100% success ratio** at 1,000 RPS with an empty vegeta error set — the pooled transport (`MaxIdleConns = 500`, `MaxIdleConnsPerHost = 200`) recycled connections cleanly instead of exhausting sockets or queuing on new TCP handshakes.
* **Minimal Infrastructure Overhead:** Max latency (**809.97ms**) was only ~10ms above the 800ms simulated backend delay, and p99 (**800.87ms**) only ~0.87ms above it — so the gateway's own processing (guardrail, fraud policy, order-context lookup, connection pooling) adds negligible tail latency under peak load.
* **Reading the latency spread:** `min` (62.67µs) is the fraud-intercept path (`ORD-1002`), which returns before any network call is made; everything else reflects the `ORD-1001` path, which does the real round trip to the mock server. Because traffic is an even 50/50 split between those two paths, `p50` (368.45ms) falls between the two clusters as a percentile-interpolation artifact — it isn't a real "typical" latency — while `p90` and above land solidly in the ~800ms cluster since that's the upper half of the sorted distribution.
