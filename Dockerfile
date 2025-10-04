# Use a multi-stage build to compile the Go application.
FROM golang:1.17 AS builder

WORKDIR /app

# Install ZeroMQ libraries and other build essentials.
RUN apt-get update -y && apt-get upgrade -y && apt-get autoremove -y \
    && apt-get install -y libzmq5-dev pkg-config build-essential

# Add Go module files first for optimized caching.
COPY go.mod go.sum ./

# Download dependencies.
RUN go mod download

# Copy the rest of the application files into the image.
COPY . .

# Build the command inside the container.
RUN go build -o iota-zmq-bridge ./iota-zmq-bridge.go

# Use ubuntu:focal as the base image.
FROM ubuntu:focal

# Install runtime dependencies.
RUN apt-get update -y && apt-get upgrade -y && apt-get autoremove -y \
    && apt-get install -y libzmq5 net-tools iproute2 iputils-ping bash

# Copy the binary from the builder stage.
COPY --from=builder /app/iota-zmq-bridge /iota-zmq-bridge

# Copy the entrypoint script and make it executable
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Environment variables with default values.
ENV ZMQ_IP=0.0.0.0 \
    ZMQ_PORT=5556 \
    MQTT_IP=localhost \
    MQTT_PORT=1883 \
    INDEXES="" \
    BUFFER_SIZE=1000 \
    NUM_WORKERS=10 \
    VERBOSE_MESSAGES=false

# Set the ENTRYPOINT to our script and CMD to /bin/bash
ENTRYPOINT ["/bin/bash", "/entrypoint.sh"]

# Expose any necessary ports.
EXPOSE 5556
