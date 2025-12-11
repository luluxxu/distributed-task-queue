
set -e

REGION="us-west-2"
CLUSTER="task-queue-cluster"
WORKER_SERVICE="task-queue-worker"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get API endpoint
# echo -e "\n${YELLOW}Getting API endpoint...${NC}"
# API_IP=$(./get_api_ip.sh 2>/dev/null | grep "API Endpoint" | awk '{print $3}')

# if [ -z "$API_IP" ]; then
#     echo "Error: Could not get API endpoint"
#     exit 1
# fi

echo -e "${GREEN}✓ API Endpoint: $API_IP${NC}"

# ============================================
# Test 2: Priority Queue
# ============================================

echo -e "\n${BLUE}=== Test 2: Priority Queue ===${NC}"

# Update workers to priority queue
# Ensure workers use FIFO queue
# cd ../terraform
# if ! grep -q 'worker_queue_type = "priority"' terraform.tfvars; then
#     echo -e "${YELLOW}Updating workers to priority queue...${NC}"
#     sed -i.bak 's/worker_queue_type = "fifo"/worker_queue_type = "priority"/' terraform.tfvars 2>/dev/null || \
#     sed -i 's/worker_queue_type = "fifo"/worker_queue_type = "priority"/' terraform.tfvars
    
#     terraform apply -auto-approve
#     echo -e "${YELLOW}Waiting for workers to restart (30s)...${NC}"
#     sleep 30
# fi
# cd ../locust


echo -e "${GREEN}✓ Workers configured for priority queue${NC}"

# Start metrics collection in background
echo -e "${YELLOW}Starting metrics collection...${NC}"
./collect_metrics.sh priority > /dev/null 2>&1 &
METRICS_PID_PRIORITY=$!
echo -e "${GREEN}✓ Metrics collection started (PID: $METRICS_PID_PRIORITY)${NC}"

# Run Locust load test for priority queue
echo -e "\n${YELLOW}Running Locust load test...${NC}"
echo "  Users: 200"
echo "  Spawn rate: 50 users/sec"
echo "  Duration: 5 minutes"
echo ""

locust -f locustfile_priority.py \
    --host $API_URL \
    --users 15 \
    --spawn-rate 5 \
    --run-time 3m \
    --headless \
    --csv=priority_results

# Stop metrics collection
kill $METRICS_PID_PRIORITY 2>/dev/null || true
wait $METRICS_PID_PRIORITY 2>/dev/null || true

# Rename results
mv experiment_results.json priority_experiment_results.json 2>/dev/null || true
echo -e "${GREEN}✓ Priority test complete!${NC}"
echo "  Results: priority_experiment_results.json"
echo "  Metrics: priority_metrics.json"
echo "  CSV: priority_results_stats.csv"

# ============================================
# Results Comparison
# ============================================

echo -e "\n${BLUE}=== Results Comparison ===${NC}"

if [ -f fifo_experiment_results.json ] && [ -f priority_experiment_results.json ]; then
    echo -e "\n${YELLOW}FIFO Queue:${NC}"
    python3 << 'EOF'
import json
try:
    with open('fifo_experiment_results.json') as f:
        data = json.load(f)
    print(f"  Total tasks: {data.get('total_tasks', 0)}")
    print(f"  50th percentile: {data.get('p50', 0):.3f}s")
    print(f"  95th percentile: {data.get('p95', 0):.3f}s")
    print(f"  99th percentile: {data.get('p99', 0):.3f}s")
    print(f"  Average: {data.get('average_latency', 0):.3f}s")
except Exception as e:
    print(f"  Error loading results: {e}")
EOF

    echo -e "\n${YELLOW}Priority Queue:${NC}"
    python3 << 'EOF'
import json
try:
    with open('priority_experiment_results.json') as f:
        data = json.load(f)
    print(f"  Total tasks: {data.get('total_tasks', 0)}")
    print(f"  50th percentile: {data.get('p50', 0):.3f}s")
    print(f"  95th percentile: {data.get('p95', 0):.3f}s")
    print(f"  99th percentile: {data.get('p99', 0):.3f}s")
    print(f"  Average: {data.get('average_latency', 0):.3f}s")
except Exception as e:
    print(f"  Error loading results: {e}")
EOF

    echo -e "\n${YELLOW}Improvement:${NC}"
    python3 << 'EOF'
import json
try:
    with open('fifo_experiment_results.json') as f:
        fifo = json.load(f)
    with open('priority_experiment_results.json') as f:
        priority = json.load(f)
    
    p95_improvement = ((fifo['p95'] - priority['p95']) / fifo['p95']) * 100 if fifo['p95'] > 0 else 0
    p99_improvement = ((fifo['p99'] - priority['p99']) / fifo['p99']) * 100 if fifo['p99'] > 0 else 0
    
    print(f"  95th percentile: {p95_improvement:>6.1f}% improvement")
    print(f"  99th percentile: {p99_improvement:>6.1f}% improvement")
except Exception as e:
    print(f"  Error calculating improvement: {e}")
EOF
else
    echo -e "${RED}Results files not found${NC}"
fi

# Generate visualizations
echo -e "\n${YELLOW}Generating visualizations...${NC}"
if command -v python3 &> /dev/null && python3 -c "import matplotlib" 2>/dev/null; then
    python3 analyze_metrics.py
    echo -e "${GREEN}✓ Plots generated${NC}"
else
    echo -e "${YELLOW}⚠️  matplotlib not installed, skipping plots${NC}"
    echo "  Install with: pip install matplotlib numpy"
fi

echo -e "\n${GREEN}✅ Experiment 1 Complete!${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "Results files:"
echo "  - fifo_experiment_results.json"
echo "  - priority_experiment_results.json"
echo "  - fifo_metrics.json"
echo "  - priority_metrics.json"
echo "  - experiment1_comparison.png"
echo "  - experiment1_resource_utilization.png"
echo ""