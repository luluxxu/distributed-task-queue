FROM --platform=linux/amd64  golang:1.24-alpine

WORKDIR /app

# Copy go.mod and go.sum from root (../../)
COPY ../../go.mod ../../go.sum ./

# Download dependencies
RUN go mod download

# Copy entire src directory
COPY ../../ .

# Build the API binary
WORKDIR /app/api/main
RUN go build -o api main.go

EXPOSE 8080

CMD ["./api"]