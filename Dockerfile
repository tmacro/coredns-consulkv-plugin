# Build stage
FROM golang:1.22 AS builder

WORKDIR /app

# Copy the local plugin files
COPY . /app/coredns-consulkv-plugin

# Clone CoreDNS repository
RUN git clone https://github.com/coredns/coredns.git

WORKDIR /app/coredns

# Add our plugin to plugin.cfg
RUN echo "consulkv:github.com/mwantia/coredns-consulkv-plugin" >> plugin.cfg

# Update go.mod to use the local plugin
RUN go mod edit -replace github.com/mwantia/coredns-consulkv-plugin=/app/coredns-consulkv-plugin

# Update dependencies and build
RUN go get github.com/mwantia/coredns-consulkv-plugin
RUN go mod tidy
RUN make

# Final stage
FROM debian:bullseye-slim

WORKDIR /app

# Copy the built CoreDNS binary from the builder stage
COPY --from=builder /app/coredns/coredns /app/coredns

# Expose DNS ports
EXPOSE 53/udp
EXPOSE 53/tcp

# Run CoreDNS
CMD ["/app/coredns"]