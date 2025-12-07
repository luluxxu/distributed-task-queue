# distributed-task-queue

## Project Structure

```
distributed-task-queue/
├── README.md
└── src/
    ├── api/                          # API Service
    │   ├── experiments/              # Experiment endpoints
    │   │   ├── experiment1.go      # Exp1: Task length distribution
    │   │   ├── experiment2.go       # Exp2: Worker scaling
    │   │   └── experiment3.go       # Exp3: Retry mechanism
    │   ├── main/                    # API main entry point
    │   │   ├── main.go
    │   │   └── Dockerfile
    │   ├── models/                  # Data models
    │   │   └── models.go
    │   └── ratelimit/               # Rate limiting
    │       └── ratelimit.go
    ├── client/                       # Load testing clients
    │   ├── exp1/
    │   │   └── exp1_loadtest.go
    │   ├── exp2/
    │   │   └── exp2_loadtest.go
    │   └── exp3/
    │       └── exp3_loadtest.go
    ├── worker/                       # Worker service
    │   ├── worker.go
    │   └── Dockerfile
    ├── redis/                        # Redis operations
    │   └── redis.go
    ├── docker-compose.yml
    ├── go.mod
    └── go.sum
```

### Components

- **API Service** (`api/main/main.go`): HTTP API server with task submission and status endpoints
- **Worker** (`worker/worker.go`): Task processor that pulls from Redis queue
- **Redis** (`redis/redis.go`): Queue and result store operations
- **Rate Limiter** (`api/ratelimit/ratelimit.go`): Per-client rate limiting
- **Experiments**: Three experiment endpoints for different testing scenarios
- **Client Load Tests**: Load testing scripts for each experiment

### Quick Start

1. **Start Redis**: `docker-compose up -d redis`
2. **Start API**: `go run ./api/main/main.go`
3. **Start Worker**: `go run ./worker/worker.go --queue=fifo --mode=simple`
4. **Run Tests**: See [TESTING.md](./TESTING.md) for detailed testing guide

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


## Experiment 1 

Run experiment 1 with exp1_loadtest.go
```
cd src

# Start Redis
docker-compose up -d redis

# Start API Service
go run ./api/main/main.go

# Start Worker.go with PQ
go run ./worker/worker.go --queue=priority --mode=simple  

# Start Worker.go with fifo
go run ./worker/worker.go --queue=fifo --mode=simple

# Start experiment 1 test
go run ./client/exp1/exp1_loadtest.go
```

## Experiment 2: Worker Scaling and Backlog Clearance

Run experiment 2 with exp2_loadtest.go
```
cd src

# Start Redis
docker-compose up -d redis

# Start API Service
go run ./api/main/main.go

# Start N workers (test with different counts: 1, 2, 5, 10)
# Example with 1 worker:
go run ./worker/worker.go --queue=fifo --mode=simple

# Example with 5 workers (run in separate terminals):
# Terminal 1: go run ./worker/worker.go --queue=fifo --mode=simple
# Terminal 2: go run ./worker/worker.go --queue=fifo --mode=simple
# Terminal 3: go run ./worker/worker.go --queue=fifo --mode=simple
# Terminal 4: go run ./worker/worker.go --queue=fifo --mode=simple
# Terminal 5: go run ./worker/worker.go --queue=fifo --mode=simple

# Start experiment 2 test (submits 500 tasks and measures clearance time)
go run ./client/exp2/exp2_loadtest.go

# Compare results with different worker counts to see scaling effects
# clean up 
docker-compose down
```

## Experiment 3: Retry Mechanism

Run experiment 3 with exp3_loadtest.go
```
cd src

# Start Redis
docker-compose up -d redis

# Start API Service
go run ./api/main/main.go

# Start Worker.go with fifo
go run ./worker/worker.go --queue=fifo --mode=retry

# Start Worker.go with pq
go run ./worker/worker.go --queue=priority --mode=retry

# Start experiment 3 test
go run ./client/exp3/exp3_loadtest.go

# clean up 
docker-compose down
```
