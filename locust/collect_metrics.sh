#!/bin/bash

# collect_metrics.sh - Collect CloudWatch metrics during load test
# Usage: ./collect_metrics.sh <test_name>
# Example: ./collect_metrics.sh fifo

TEST_NAME=${1:-"test"}
REGION="us-west-2"
CLUSTER="task-queue-cluster"
WORKER_SERVICE="task-queue-worker"
API_SERVICE="task-queue-api"
REDIS_CLUSTER="task-queue-redis"
OUTPUT_FILE="${TEST_NAME}_metrics.json"

echo "Collecting metrics for $TEST_NAME test..."
echo "Metrics will be saved to $OUTPUT_FILE"

# Collect metrics every 30 seconds
while true; do
    TIMESTAMP=$(date -u +%Y-%m-%dT%H:%M:%SZ)
    
    # Get Worker CPU
    WORKER_CPU=$(aws cloudwatch get-metric-statistics \
        --namespace AWS/ECS \
        --metric-name CPUUtilization \
        --dimensions Name=ServiceName,Value=$WORKER_SERVICE Name=ClusterName,Value=$CLUSTER \
        --start-time $(date -u -v-2M +%Y-%m-%dT%H:%M:%S) \
        --end-time $(date -u +%Y-%m-%dT%H:%M:%S) \
        --period 60 \
        --statistics Average \
        --region $REGION \
        --query 'Datapoints[0].Average' \
        --output text)
    
    # Get Worker Memory
    WORKER_MEM=$(aws cloudwatch get-metric-statistics \
        --namespace AWS/ECS \
        --metric-name MemoryUtilization \
        --dimensions Name=ServiceName,Value=$WORKER_SERVICE Name=ClusterName,Value=$CLUSTER \
        --start-time $(date -u -v-2M +%Y-%m-%dT%H:%M:%S) \
        --end-time $(date -u +%Y-%m-%dT%H:%M:%S) \
        --period 60 \
        --statistics Average \
        --region $REGION \
        --query 'Datapoints[0].Average' \
        --output text)
    
    # Get API CPU
    API_CPU=$(aws cloudwatch get-metric-statistics \
        --namespace AWS/ECS \
        --metric-name CPUUtilization \
        --dimensions Name=ServiceName,Value=$API_SERVICE Name=ClusterName,Value=$CLUSTER \
        --start-time $(date -u -v-2M +%Y-%m-%dT%H:%M:%S) \
        --end-time $(date -u +%Y-%m-%dT%H:%M:%S) \
        --period 60 \
        --statistics Average \
        --region $REGION \
        --query 'Datapoints[0].Average' \
        --output text)
    
    # Get ElastiCache CPU
    REDIS_CPU=$(aws cloudwatch get-metric-statistics \
        --namespace AWS/ElastiCache \
        --metric-name CPUUtilization \
        --dimensions Name=CacheClusterId,Value=$REDIS_CLUSTER \
        --start-time $(date -u -v-2M +%Y-%m-%dT%H:%M:%S) \
        --end-time $(date -u +%Y-%m-%dT%H:%M:%S) \
        --period 60 \
        --statistics Average \
        --region $REGION \
        --query 'Datapoints[0].Average' \
        --output text)
    
    # Get ElastiCache Memory
    REDIS_MEM=$(aws cloudwatch get-metric-statistics \
        --namespace AWS/ElastiCache \
        --metric-name DatabaseMemoryUsagePercentage \
        --dimensions Name=CacheClusterId,Value=$REDIS_CLUSTER \
        --start-time $(date -u -v-2M +%Y-%m-%dT%H:%M:%S) \
        --end-time $(date -u +%Y-%m-%dT%H:%M:%S) \
        --period 60 \
        --statistics Average \
        --region $REGION \
        --query 'Datapoints[0].Average' \
        --output text)
    
    # Get number of running workers
    WORKER_COUNT=$(aws ecs describe-services \
        --cluster $CLUSTER \
        --services $WORKER_SERVICE \
        --region $REGION \
        --query 'services[0].runningCount' \
        --output text)
    
    # Print current metrics
    echo "[$TIMESTAMP] Workers: $WORKER_COUNT, Worker CPU: ${WORKER_CPU}%, API CPU: ${API_CPU}%, Redis CPU: ${REDIS_CPU}%, Redis Mem: ${REDIS_MEM}%"
    
    # Append to JSON file
    echo "{\"timestamp\":\"$TIMESTAMP\",\"worker_cpu\":\"$WORKER_CPU\",\"worker_memory\":\"$WORKER_MEM\",\"api_cpu\":\"$API_CPU\",\"redis_cpu\":\"$REDIS_CPU\",\"redis_memory\":\"$REDIS_MEM\",\"worker_count\":\"$WORKER_COUNT\"}" >> $OUTPUT_FILE
    
    sleep 30  # Collect every 30 seconds
done