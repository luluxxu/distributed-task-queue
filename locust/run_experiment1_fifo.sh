#!/bin/bash

# run_experiment1.sh - Complete Experiment 1 with Metrics Collection

set -e

REGION="us-west-2"
CLUSTER="task-queue-cluster"
WORKER_SERVICE="task-queue-worker"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Experiment 1: FIFO vs Priority Queue${NC}"
echo -e "${BLUE}========================================${NC}"

# Check dependencies
if ! command -v locust &> /dev/null; then
    echo -e "${RED}Error: Locust not installed${NC}"
    echo "Install with: pip install locust"
    exit 1
fi

# Get API endpoint
# echo -e "\n${YELLOW}Getting API endpoint...${NC}"
# cd ../terraform
# API_URL=$(./get_api_ip.sh 2>&1 | grep "API Endpoint" | cut -d' ' -f3)
# cd ../locust

# if [ -z "$API_URL" ]; then
#     echo -e "${RED}Error: Could not get API endpoint${NC}"
#     exit 1
# fi

echo -e "${GREEN}✓ API Endpoint: $API_URL${NC}"

# Test API is reachable
echo -e "\n${YELLOW}Testing API connectivity...${NC}"
if curl -s --max-time 5 $API_URL/task/fifo -X POST -H "Content-Type: application/json" -d '{"job_type":"short"}' > /dev/null 2>&1; then
    echo -e "${GREEN}✓ API is reachable${NC}"
else
    echo -e "${RED}❌ API is not responding${NC}"
    echo "Wait a few minutes for containers to start, then try again"
    exit 1
fi

# ============================================
# Test 1: FIFO Queue
# ============================================

echo -e "\n${BLUE}=== Test 1: FIFO Queue ===${NC}"

# Ensure workers use FIFO queue
# cd ../terraform
# if ! grep -q 'worker_queue_type = "fifo"' terraform.tfvars; then
#     echo -e "${YELLOW}Updating workers to FIFO queue...${NC}"
#     sed -i.bak 's/worker_queue_type = "priority"/worker_queue_type = "fifo"/' terraform.tfvars 2>/dev/null || \
#     sed -i 's/worker_queue_type = "priority"/worker_queue_type = "fifo"/' terraform.tfvars
    
#     terraform apply -auto-approve
#     echo -e "${YELLOW}Waiting for workers to restart (30s)...${NC}"
#     sleep 30
# fi
# cd ../locust

echo -e "${GREEN}✓ Workers configured for FIFO queue${NC}"

# Start metrics collection in background
echo -e "${YELLOW}Starting metrics collection...${NC}"
./collect_metrics.sh fifo > /dev/null 2>&1 &
METRICS_PID_FIFO=$!
echo -e "${GREEN}✓ Metrics collection started (PID: $METRICS_PID_FIFO)${NC}"

# Run Locust load test
echo -e "\n${YELLOW}Running Locust load test...${NC}"
echo "  Users: 50"
echo "  Spawn rate: 10 users/sec"
echo "  Duration: 3 minutes"
echo ""

locust -f locustfile_fifo.py \
    --host $API_URL \
    --users 15 \
    --spawn-rate 5 \
    --run-time 3m \
    --headless \
    --csv=fifo_results

sleep 5m
# Stop metrics collection
kill $METRICS_PID_FIFO 2>/dev/null || true
wait $METRICS_PID_FIFO 2>/dev/null || true

# Rename results
mv experiment_results.json fifo_experiment_results.json 2>/dev/null || true
echo -e "${GREEN}✓ FIFO test complete!${NC}"
echo "  Results: fifo_experiment_results.json"
echo "  Metrics: fifo_metrics.json"
echo "  CSV: fifo_results_stats.csv"
