FROM --platform=linux/amd64 golang:1.24-alpine

WORKDIR /app

# Copy go.mod and go.sum from parent (../)
COPY ../go.mod ../go.sum ./

# Download dependencies
RUN go mod download

# Copy entire src directory
COPY ../ .

# Build the worker binary
WORKDIR /app/worker
RUN go build -o worker worker.go

# Default to FIFO queue (can be overridden)
CMD ["./worker", "-queue=fifo"]