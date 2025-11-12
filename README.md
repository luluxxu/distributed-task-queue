# distributed-task-queue

## Experiment 1

Run experiment 1 with FIFO queue
```
cd src

# Start FIFO services
docker-compose --profile fifo up --build -d

# Monitor worker logs to see task completion
docker-compose logs -f worker-fifo-1 | grep "Completed"

# Check Redis fifo queue
docker exec -it task-queue-redis redis-cli LLEN task:queue

# Stop fifo experiment
docker-compose --profile fifo down
```

Run experiment 1 with priority queue
```
# Start in the same /src directory

# Start priority services
docker-compose --profile priority up --build -d

# Check if task is in priority queue
docker exec -it task-queue-redis redis-cli ZCARD task:priority_queue

# See the task
docker exec -it task-queue-redis redis-cli ZRANGE task:priority_queue 0 -1 WITHSCORES

# Monitor worker logs to see task completion
docker-compose logs -f worker-priority-1 | grep "Completed"

# Stop pq experiment
docker-compose --profile priority down
```


Submitting tasks for testing 
(same code for both fifo and pq, only difference is the path)
```
# Submit a short task for fifo testing
curl -X POST http://localhost:8080/task/fifo \
    -H "Content-Type: application/json" \
    -d '{"job_type":"short"}'

# Submit a long task for fifo testing
  curl -X POST http://localhost:8080/task/fifo \
    -H "Content-Type: application/json" \
    -d '{"job_type":"long"}'

# Submit a short task for pq testing
curl -X POST http://localhost:8080/task/pq \
  -H "Content-Type: application/json" \
  -d '{"job_type":"short"}'

# Submit a long task for pq testing
  curl -X POST http://localhost:8080/task/fifo \
    -H "Content-Type: application/json" \
    -d '{"job_type":"long"}'

# Use task ID from response to check status
curl http://localhost:8080/task/{task id}
```