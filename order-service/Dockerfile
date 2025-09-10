# Build stage
FROM golang:1.21-alpine AS builder

# Install Delve debugger for debug builds
ARG BUILD_MODE=production
RUN if [ "$BUILD_MODE" = "debug" ] ; then go install github.com/go-delve/delve/cmd/dlv@v1.21.2 ; fi

WORKDIR /app

# Copy shared domain first
COPY shared-domain/ ./shared-domain/

# Copy go mod files
COPY order-service/go.mod order-service/go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY order-service/ .

# Build binary - with or without debug symbols based on BUILD_MODE
RUN if [ "$BUILD_MODE" = "debug" ] ; then \
        go build -gcflags="all=-N -l" -o main cmd/main.go ; \
    else \
        CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/main.go ; \
    fi

# Runtime stage
FROM alpine:3.18

# Install ca-certificates for HTTPS calls
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary
ARG BUILD_MODE=production
COPY --from=builder /app/main .

# Copy delve if debug mode
COPY --from=builder /go/bin/dlv* ./

# Expose ports
EXPOSE 8001 2345

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8001/api/v1/health || exit 1

# Create startup script (v2 - fixed debug mode)
ARG BUILD_MODE=production
RUN if [ "$BUILD_MODE" = "debug" ] ; then \
        echo '#!/bin/sh' > start.sh && \
        echo 'echo "Starting in debug mode..."' >> start.sh && \
        echo './dlv exec ./main --listen=:2345 --headless=true --api-version=2 --accept-multiclient --continue' >> start.sh && \
        chmod +x start.sh ; \
    else \
        echo '#!/bin/sh' > start.sh && \
        echo './main' >> start.sh && \
        chmod +x start.sh ; \
    fi

CMD ["./start.sh"]