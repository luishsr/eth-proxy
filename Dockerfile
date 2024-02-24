# Start from a lightweight Golang base image
FROM golang:1.18-alpine as builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod, go.sum, and the rest of the code
COPY go.* ./
COPY . .

# Download dependencies
RUN go mod download

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o eth-proxy ./cmd/api

# Use a small Alpine Linux image to run the app
FROM alpine:latest

# Install ca-certificates in case we need to call HTTPS endpoints
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/eth-proxy .

# Copy the .env file
COPY .env .

# Expose port 8080 to the outside world
EXPOSE 8088

# Command to run the executable
CMD ["./eth-proxy"]
