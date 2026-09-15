#!/bin/bash

echo "⚡ Initializing 1,000 RPS Stress Test Matrix..."
echo "👉 Target: Go Gateway Proxy on http://localhost:8080"
echo "⏱️  Duration: 30 Seconds | Total Scheduled Packets: 30,000"
echo "--------------------------------------------------------"

# Ensure vegeta tool binary utility is available
if ! command -v vegeta &> /dev/null
then
    echo "❌ Error: 'vegeta' is not installed."
    echo "Install via Mac: 'brew install vegeta' or Linux: 'apt-get install vegeta'"
    exit 1
fi

# Run the high-performance attack execution
# -rate=1000 targets constant 1,000 requests per second
# -duration=30s runs the test for 30 consecutive seconds
vegeta attack \
    -targets=targets.txt \
    -rate=1000 \
    -duration=30s \
    -timeout=5s > results.bin

echo "✅ Attack completed. Generating statistical performance diagnostics report..."
echo "--------------------------------------------------------"

# 1. Output a clean text report directly into the console
vegeta report results.bin

# 2. Senior Flex: Generate an interactive vector HTML dashboard plot
vegeta plot results.bin > performance_latency_plot.html
echo "📊 Success. Interactive HTML latency chart generated: benchmarks/performance_latency_plot.html"
